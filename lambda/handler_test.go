package main

import (
	"context"
	"forta-bot-db/auth"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/golang/mock/gomock"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	m "forta-bot-db/store/mocks"
)

func testCtx(t *testing.T, botID, scannerID string) *auth.HandlerCtx {
	ctrl := gomock.NewController(t)
	s := m.NewMockS3(ctrl)
	return &auth.HandlerCtx{
		Ctx:     context.Background(),
		BotID:   botID,
		Scanner: scannerID,
		Logger:  log.WithField("test", true),
		Store:   s,
	}
}

func TestObjectKey(t *testing.T) {
	type keyTest struct {
		Path        string
		Scanner     string
		BotId       string
		ExpectedKey string
		PathParams  map[string]string
	}
	tests := []keyTest{
		{
			Scanner:     "scanner",
			BotId:       "botId",
			ExpectedKey: "botId/scanner/key",
			PathParams: map[string]string{
				"key": "key",
			},
		},
		{
			Scanner:     "scanner",
			BotId:       "botId",
			ExpectedKey: "botId/key",
			PathParams: map[string]string{
				"key":   "key",
				"scope": "bot",
			},
		},
	}
	for _, test := range tests {
		key, err := getObjKey(testCtx(t, test.BotId, test.Scanner), events.APIGatewayV2HTTPRequest{
			RequestContext: events.APIGatewayV2HTTPRequestContext{
				HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
					Method: "GET",
				},
			},
			PathParameters: test.PathParams,
		})
		assert.NoError(t, err)
		assert.Equal(t, test.ExpectedKey, key)
	}
}

func TestRoute(t *testing.T) {
	bucket = "test-bucket"
	ctrl := gomock.NewController(t)
	s := m.NewMockS3(ctrl)

	hc := &auth.HandlerCtx{
		Ctx:     context.Background(),
		BotID:   "0xbotId",
		Scanner: "0xscanner",
		Logger:  log.WithField("test", true),
		Store:   s,
	}

	r := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "GET",
			},
		},
		PathParameters: map[string]string{
			"key": "key",
		},
	}

	body := "test"
	s.EXPECT().GetObject(hc.Ctx, gomock.Any()).Return(&s3.GetObjectOutput{Body: io.NopCloser(strings.NewReader(body))}, nil)

	_, err := getObj(hc, r)

	assert.NoError(t, err)
}
