package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/forta-network/forta-core-go/registry"
	"github.com/forta-network/forta-core-go/security"
	"github.com/golang-jwt/jwt/v4"
	log "github.com/sirupsen/logrus"

	"forta-bot-db/store"
)

type HandlerCtx struct {
	Ctx     context.Context
	BotID   string
	Scanner string
	Key     string
	Body    []byte
	PathKey string
	Scope   string
	Method  string
	Logger  *log.Entry
	Store   store.S3
}

func parseBodyFromGateway(r events.APIGatewayV2HTTPRequest) ([]byte, error) {
	if r.RequestContext.HTTP.Method != "post" && r.RequestContext.HTTP.Method != "put" {
		return nil, nil
	}
	bodyStr := r.Body
	b := []byte(bodyStr)
	if r.IsBase64Encoded {
		bs, err := base64.StdEncoding.DecodeString(bodyStr)
		if err != nil {
			return nil, err
		}
		b = bs
	}
	return b, nil
}

func extractGwContext(ctx context.Context, request events.APIGatewayV2HTTPRequest) (*HandlerCtx, error) {
	// headers are lowercased via lambda
	h, ok := request.Headers["authorization"]
	if !ok {
		return nil, errors.New("no Authorization header")
	}
	parts := strings.Split(h, " ")
	if len(parts) != 2 {
		return nil, errors.New("invalid Authorization header")
	}
	st, err := security.VerifyScannerJWT(parts[1])
	if err != nil {
		return nil, err
	}

	if c, ok := st.Token.Claims.(jwt.MapClaims); ok {
		if botId, botOk := c["bot-id"]; botOk {
			s, err := store.NewS3Client(ctx)
			if err != nil {
				return nil, err
			}
			b, err := parseBodyFromGateway(request)
			if err != nil {
				return nil, err
			}

			pathKey, ok := request.PathParameters["key"]
			if !ok {
				return nil, errors.New("no key defined")
			}
			scope, _ := request.PathParameters["scope"]

			return &HandlerCtx{
				Ctx:     ctx,
				BotID:   botId.(string),
				Scanner: st.Scanner,
				Method:  request.RequestContext.HTTP.Method,
				Store:   s,
				PathKey: pathKey,
				Scope:   scope,
				Body:    b,
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

func authorize(botCtx *HandlerCtx) error {
	r, err := registry.NewDefaultClient(botCtx.Ctx)
	if err != nil {
		return err
	}

	enabled, err := r.IsEnabledScanner(botCtx.Scanner)
	if err != nil {
		return err
	}
	if !enabled {
		return errors.New("scanner is not enabled")
	}

	assigned, err := r.IsAssigned(botCtx.Scanner, botCtx.BotID)
	if err != nil {
		return err
	}
	if !assigned {
		return errors.New("botId is not assigned to scanner")
	}

	return nil
}

func AuthorizeGWRequest(ctx context.Context, request events.APIGatewayV2HTTPRequest) (*HandlerCtx, error) {
	botCtx, err := extractGwContext(ctx, request)
	if err != nil {
		return nil, err
	}

	if err := authorize(botCtx); err != nil {
		return nil, err
	}

	return botCtx, nil
}
