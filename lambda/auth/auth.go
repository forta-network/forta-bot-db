package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	rd "github.com/forta-network/forta-core-go/domain/registry"
	"github.com/forta-network/forta-core-go/registry"
	"github.com/forta-network/forta-core-go/security"
	"github.com/golang-jwt/jwt/v4"
	log "github.com/sirupsen/logrus"

	"forta-bot-db/store"
)

type Scope string

const ScopeScanner Scope = "scanner"
const ScopeBot Scope = "bot"
const ScopeOwner Scope = "owner"
const DefaultScope = ScopeScanner

var ErrNotAssigned = errors.New("botId is not assigned to scanner")

var ErrNotEnabled = errors.New("scanner is not enabled")

type HandlerCtx struct {
	Ctx     context.Context
	BotID   string
	Scanner string
	Owner   string
	PathKey string
	Scope   Scope
	Logger  *log.Entry
	Store   store.S3
}

type JwtVerifier func(tokenString string) (*security.ScannerToken, error)

var jwtVerifier JwtVerifier = security.VerifyScannerJWT

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
	r registry.Client
}

type ensStore struct{}

func (es *ensStore) Resolve(input string) (common.Address, error) {
	return common.HexToAddress("0x0"), nil
}

// ResolveRegistryContracts this helps with speed
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
	r, err := registry.NewClientWithENSStore(ctx, registry.ClientConfig{
		JsonRpcUrl: url,
		NoRefresh:  true,
	}, &ensStore{})
	if err != nil {
		return nil, err
	}
	return &Authorizer{r: r}, nil
}

func (a *Authorizer) Authorize(ctx context.Context, request events.APIGatewayV2HTTPRequest) (*HandlerCtx, error) {
	botCtx, err := extractContext(ctx, request)
	if err != nil {
		return nil, err
	}

	enabled, err := a.r.IsEnabledScanner(botCtx.Scanner)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, ErrNotEnabled
	}

	assigned, err := a.r.IsAssigned(botCtx.Scanner, botCtx.BotID)
	if err != nil {
		return nil, err
	}
	if !assigned {
		return nil, ErrNotAssigned
	}

	if botCtx.Scope == ScopeOwner {
		agt, err := a.r.GetAgent(botCtx.BotID)
		if err != nil {
			return nil, err
		}
		botCtx.Owner = strings.ToLower(agt.Owner)
	}

	return botCtx, nil
}
