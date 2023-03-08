package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/events"
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

func NewAuthorizer(ctx context.Context) (*Authorizer, error) {
	r, err := registry.NewDefaultClient(ctx)
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
		return nil, errors.New("scanner is not enabled")
	}

	assigned, err := a.r.IsAssigned(botCtx.Scanner, botCtx.BotID)
	if err != nil {
		return nil, err
	}
	if !assigned {
		return nil, errors.New("botId is not assigned to scanner")
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
