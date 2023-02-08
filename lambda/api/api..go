package api

import (
	"bytes"
	"errors"
	"forta-bot-db/auth"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io/ioutil"
	"os"
)

var bucket = os.Getenv("bucket")

func getObj(hc *auth.HandlerCtx) ([]byte, error) {
	res, err := hc.Store.GetObject(hc.Ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &hc.Key,
	})
	if err != nil {
		hc.Logger.WithError(err).Error("error getting object from s3")
		return nil, err
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {

		hc.Logger.WithError(err).Error("error reading body from object")
		return nil, err
	}
	return b, nil
}

func putObj(hc *auth.HandlerCtx) error {
	_, err := hc.Store.PutObject(hc.Ctx, &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &hc.Key,
		Body:   bytes.NewReader(hc.Body),
	})
	if err != nil {
		hc.Logger.WithError(err).Error("could not write object")
		return err
	}
	return nil
}

func delObj(hc *auth.HandlerCtx) error {
	_, err := hc.Store.DeleteObject(hc.Ctx, &s3.DeleteObjectInput{
		Bucket: &bucket,
		Key:    &hc.Key,
	})
	if err != nil {
		hc.Logger.WithError(err).Error("could not delete object")
		return err
	}
	return nil
}

func Route(hc *auth.HandlerCtx) ([]byte, error) {
	switch hc.Method {
	case "get":
		return getObj(hc)
	case "put":
		if err := putObj(hc); err != nil {
			return nil, err
		}
		return nil, nil
	case "post":
		if err := putObj(hc); err != nil {
			return nil, err
		}
		return nil, nil
	case "delete":
		if err := delObj(hc); err != nil {
			return nil, err
		}
		return nil, nil
	case "options":
		return nil, nil
	default:
		return nil, errors.New("method not allowed")
	}
}
