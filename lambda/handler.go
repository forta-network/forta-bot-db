package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	log "github.com/sirupsen/logrus"

	"forta-bot-db/api"
	"forta-bot-db/auth"
)

const ScopeScanner = "scanner"
const ScopeBot = "bot"

const DefaultScope = ScopeScanner

func getObjKey(hc *auth.HandlerCtx) (string, error) {
	scope := hc.Scope
	if scope == "" {
		scope = DefaultScope
	}
	var key string
	switch hc.Scope {
	case ScopeScanner:
		key = fmt.Sprintf("%s/%s/%s", hc.BotID, hc.Scanner, hc.PathKey)
	case ScopeBot:
		key = fmt.Sprintf("%s/%s", hc.BotID, hc.PathKey)
	default:
		return "", errors.New("scope must be scanner or bot")
	}

	hc.Logger = hc.Logger.WithFields(log.Fields{
		"key": key,
	})
	return key, nil
}

// Handler function Using AWS Lambda Proxy Request
func Handler(r events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.WithFields(log.Fields{
		"path":   r.RawPath,
		"method": r.RequestContext.HTTP.Method,
	}).Info("request")

	hc, err := auth.AuthorizeGWRequest(ctx, r)
	if err != nil {
		log.WithError(err).Error("unauthorized")
		return api.Unauthorized(), nil
	}

	key, err := getObjKey(hc)
	if err != nil {
		return api.NotFound(), nil
	}
	hc.Key = key
	b, err := api.Route(hc)
	if err != nil {
		return api.InternalError(), nil
	}
	if b != nil {
		return api.OKBytes(b), nil
	}
	return api.OK(), nil
}

func main() {
	lambda.Start(Handler)
}
