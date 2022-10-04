package auth

import (
	"context"
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
	Logger  *log.Entry
	Store   store.S3
}

func extractContext(ctx context.Context, request events.APIGatewayV2HTTPRequest) (*HandlerCtx, error) {
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
			return &HandlerCtx{
				Ctx:     ctx,
				BotID:   botId.(string),
				Scanner: st.Scanner,
				Store:   s,
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

func Authorize(ctx context.Context, request events.APIGatewayV2HTTPRequest) (*HandlerCtx, error) {
	botCtx, err := extractContext(ctx, request)
	if err != nil {
		return nil, err
	}

	r, err := registry.NewDefaultClient(ctx)
	if err != nil {
		return nil, err
	}

	enabled, err := r.IsEnabledScanner(botCtx.Scanner)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, errors.New("scanner is not enabled")
	}

	assigned, err := r.IsAssigned(botCtx.Scanner, botCtx.BotID)
	if err != nil {
		return nil, err
	}
	if !assigned {
		return nil, errors.New("botId is not assigned to scanner")
	}

	return botCtx, nil
}
