package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	log "github.com/sirupsen/logrus"

	"forta-bot-db/api"
	"forta-bot-db/auth"
)

var bucket = os.Getenv("bucket")

func getObjKey(hc *auth.HandlerCtx, r events.APIGatewayV2HTTPRequest) (string, error) {
	pathKey, ok := r.PathParameters["key"]
	if !ok {
		return "", errors.New("no key defined")
	}
	key := fmt.Sprintf("%s/%s/%s", hc.BotID, hc.Scanner, pathKey)
	hc.Logger = hc.Logger.WithFields(log.Fields{
		"bucket": bucket,
		"key":    key,
	})
	return pathKey, nil
}

func getObj(hc *auth.HandlerCtx, r events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	key, err := getObjKey(hc, r)
	if err != nil {
		return api.NotFound(), nil
	}

	res, err := hc.Store.GetObject(hc.Ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})

	if err != nil {
		hc.Logger.WithError(err).Error("error getting object from s3")
		return api.InternalError(), nil
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {

		hc.Logger.WithError(err).Error("error reading body from object")
		return api.InternalError(), nil
	}
	return api.OKBytes(b), nil
}
func putObj(hc *auth.HandlerCtx, r events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var b []byte
	bodyStr := r.Body
	b = []byte(bodyStr)
	if r.IsBase64Encoded {
		bs, err := base64.StdEncoding.DecodeString(bodyStr)
		if err != nil {
			hc.Logger.WithError(err).Error("could not decode body")
			return api.InternalError(), nil
		}
		b = bs
	}
	key, err := getObjKey(hc, r)
	if err != nil {
		return api.NotFound(), nil
	}
	_, err = hc.Store.PutObject(hc.Ctx, &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   bytes.NewReader(b),
	})
	if err != nil {
		hc.Logger.WithError(err).Error("could not write object")
		return api.InternalError(), nil
	}
	return api.OK(), nil
}
func delObj(hc *auth.HandlerCtx, r events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	key, err := getObjKey(hc, r)
	if err != nil {
		return api.NotFound(), nil
	}
	_, err = hc.Store.DeleteObject(hc.Ctx, &s3.DeleteObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		hc.Logger.WithError(err).Error("could not delete object")
		return api.InternalError(), nil
	}
	return api.OK(), nil
}

func route(hc *auth.HandlerCtx, r events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	switch strings.ToLower(r.RequestContext.HTTP.Method) {
	case "get":
		return getObj(hc, r)
	case "put":
		return putObj(hc, r)
	case "post":
		return putObj(hc, r)
	case "delete":
		return delObj(hc, r)
	default:
		hc.Logger.Warn("method not allowed")
		return api.MethodNotAllowed(), nil
	}
}

// Handler function Using AWS Lambda Proxy Request
func Handler(r events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.WithFields(log.Fields{
		"path":   r.RawPath,
		"method": r.RequestContext.HTTP.Method,
	}).Info("request")

	hc, err := auth.Authorize(ctx, r)
	if err != nil {
		log.WithError(err).Error("unauthorized")
		return api.Unauthorized(), nil
	}

	return route(hc, r)
}

func main() {
	lambda.Start(Handler)
}
