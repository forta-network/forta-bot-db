package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/ethereum/go-ethereum/common"
	rd "github.com/forta-network/forta-core-go/domain/registry"
	"github.com/forta-network/forta-core-go/registry"
	"github.com/forta-network/forta-core-go/security"
	"github.com/golang-jwt/jwt/v4"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
	"time"

	"forta-bot-db/store"
)

type Scope string

const ScopeScanner Scope = "scanner"
const ScopeBot Scope = "bot"
const ScopeOwner Scope = "owner"
const DefaultScope = ScopeScanner

var ErrNotAssigned = errors.New("botId is not assigned to scanner")

var ErrNotEnabled = errors.New("scanner is not enabled")

type CtxState struct {
	AuthID    string `dynamodbav:"authId"`
	BotID     string `dynamodbav:"botId"`
	Scanner   string `dynamodbav:"scanner"`
	Owner     string `dynamodbav:"owner"`
	ExpiresAt int64  `dynamodbav:"expiresAt"`
}

type HandlerCtx struct {
	Ctx       context.Context
	AuthID    string
	BotID     string
	Scanner   string
	Owner     string
	ExpiresAt int64
	PathKey   string
	Scope     Scope
	Logger    *log.Entry
	Store     store.S3
}

type JwtVerifier func(tokenString string) (*security.ScannerToken, error)

var jwtVerifier JwtVerifier = security.VerifyScannerJWT

func calculateAuthID(botID, scanner string) string {
	return strings.ToLower(fmt.Sprintf("%s|%s", botID, scanner))
}

func (hc *HandlerCtx) GetObjectKey() (string, error) {
	switch hc.Scope {
	case ScopeScanner:
		return fmt.Sprintf("%s/%s/%s", hc.BotID, hc.Scanner, hc.PathKey), nil
	case ScopeBot:
		return fmt.Sprintf("%s/%s", hc.BotID, hc.PathKey), nil
	case ScopeOwner:
		return fmt.Sprintf("owner/%s/%s", hc.Owner, hc.PathKey), nil
	default:
		return "", errors.New("scope must be scanner, owner, or bot")
	}
}

func extractContext(ctx context.Context, request events.APIGatewayV2HTTPRequest) (*HandlerCtx, error) {
	// headers are lowercased via lambda
	h, ok := request.Headers["authorization"]
	if !ok {
		return nil, errors.New("no Authorization header")
	}
	var scope Scope
	scopeStr, ok := request.PathParameters["scope"]
	if !ok {
		scope = DefaultScope
	} else {
		scope = Scope(scopeStr)
	}

	pathKey, ok := request.PathParameters["key"]
	if !ok {
		return nil, errors.New("no key defined")
	}

	parts := strings.Split(h, " ")
	if len(parts) != 2 {
		return nil, errors.New("invalid Authorization header")
	}
	st, err := jwtVerifier(parts[1])
	if err != nil {
		return nil, err
	}

	if c, ok := st.Token.Claims.(jwt.MapClaims); ok {
		if botId, botOk := c["bot-id"]; botOk {
			s, err := store.NewS3Client(ctx)
			if err != nil {
				return nil, err
			}
			return &HandlerCtx{
				AuthID:  calculateAuthID(botId.(string), st.Scanner),
				Ctx:     ctx,
				BotID:   botId.(string),
				Scanner: st.Scanner,
				Store:   s,
				PathKey: pathKey,
				Scope:   scope,
				Logger: log.WithFields(log.Fields{
					"botId":   botId,
					"scanner": st.Scanner,
					"path":    request.RawPath,
					"method":  request.RequestContext.HTTP.Method,
				}),
			}, nil
		}
	}
	return nil, errors.New("could not extract BotID")
}

type Authorizer struct {
	r     registry.Client
	d     store.DynamoDB
	table string
}

type ensStore struct{}

func (es *ensStore) Resolve(input string) (common.Address, error) {
	return common.HexToAddress("0x0"), nil
}

// ResolveRegistryContracts this helps with speed
// WARNING: this needs to be updated if the contract addresses change...this is just faster than resolving ENS
func (es *ensStore) ResolveRegistryContracts() (*rd.RegistryContracts, error) {
	return &rd.RegistryContracts{
		Dispatch:            common.HexToAddress("0xd46832f3f8ea8bdefe5316696c0364f01b31a573"),
		AgentRegistry:       common.HexToAddress("0x61447385b019187daa48e91c55c02af1f1f3f863"),
		ScannerRegistry:     common.HexToAddress("0xbf2920129f83d75dec95d97a879942cce3dcd387"),
		ScannerPoolRegistry: common.HexToAddress("0x90ff9c193d6714e0e7a923b2bd481fb73fec731d"),
		ScannerNodeVersion:  common.HexToAddress("0x4720c872425876b6f4b4e9130cdef667ade553b2"),
		FortaStaking:        common.HexToAddress("0xd2863157539b1d11f39ce23fc4834b62082f6874"),
		Forta:               common.HexToAddress("0x9ff62d1fc52a907b6dcba8077c2ddca6e6a9d3e1"),
		Migration:           common.HexToAddress("0x1365fa3fe7f52db912dabc8e439f0843461fee16"),
		Rewards:             common.HexToAddress("0xf7239f26b79145297737166b0c66f4919af9c507"),
		StakeAllocator:      common.HexToAddress("0x5b73756e637a77fa52e5ce71ec6189a4c775c6fa"),
	}, nil
}

func NewAuthorizer(ctx context.Context) (*Authorizer, error) {
	url := os.Getenv("POLYGON_JSON_RPC")
	if url == "" {
		url = "https://polygon-rpc.com"
	}
	table := os.Getenv("table")
	if table == "" {
		return nil, errors.New("table env var is required")
	}

	r, err := registry.NewClientWithENSStore(ctx, registry.ClientConfig{
		JsonRpcUrl: url,
		NoRefresh:  true,
	}, &ensStore{})
	if err != nil {
		return nil, err
	}

	d, err := store.NewDynamoDBClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Authorizer{r: r, d: d, table: table}, nil
}

func (a *Authorizer) authorizeCtx(ctx context.Context, hc *HandlerCtx) error {
	item, err := a.d.GetItem(ctx, &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"authId": &types.AttributeValueMemberS{Value: calculateAuthID(hc.BotID, hc.Scanner)},
		},
		TableName: &a.table,
	})
	if err != nil {
		return err
	}

	if item != nil && item.Item != nil {
		var saved CtxState
		if err := attributevalue.UnmarshalMap(item.Item, &saved); err != nil {
			return err
		}
		// copy over owner from one from jwt
		hc.Owner = saved.Owner
		return nil
	}
	enabled, err := a.r.IsEnabledScanner(hc.Scanner)
	if err != nil {
		return err
	}
	if !enabled {
		return ErrNotEnabled
	}

	assigned, err := a.r.IsAssigned(hc.Scanner, hc.BotID)
	if err != nil {
		return err
	}
	if !assigned {
		return ErrNotAssigned
	}
	if hc.Scope == ScopeOwner {
		agt, err := a.r.GetAgent(hc.BotID)
		if err != nil {
			return err
		}
		hc.Owner = strings.ToLower(agt.Owner)
	}

	hcItem, err := attributevalue.MarshalMap(&CtxState{
		AuthID:    hc.AuthID,
		BotID:     hc.BotID,
		Scanner:   hc.Scanner,
		Owner:     hc.Owner,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})
	if err != nil {
		return err
	}

	_, err = a.d.PutItem(ctx, &dynamodb.PutItemInput{
		Item:      hcItem,
		TableName: &a.table,
	})
	return err
}

func (a *Authorizer) Authorize(ctx context.Context, request events.APIGatewayV2HTTPRequest) (*HandlerCtx, error) {
	botCtx, err := extractContext(ctx, request)
	if err != nil {
		return nil, err
	}

	if err := a.authorizeCtx(ctx, botCtx); err != nil {
		return nil, err
	}

	return botCtx, nil
}
