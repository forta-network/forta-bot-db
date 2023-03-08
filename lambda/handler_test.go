package main

import (
	"context"
	"forta-bot-db/auth"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/golang/mock/gomock"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	m "forta-bot-db/store/mocks"
)

func TestRoute(t *testing.T) {
	bucket = "test-bucket"
	ctrl := gomock.NewController(t)
	s := m.NewMockS3(ctrl)

	hc := &auth.HandlerCtx{
		Ctx:     context.Background(),
		BotID:   "0xbotId",
		Scanner: "0xscanner",
		Scope:   auth.ScopeBot,
		Logger:  log.WithField("test", true),
		Store:   s,
	}

	body := "test"
	s.EXPECT().GetObject(hc.Ctx, gomock.Any()).Return(&s3.GetObjectOutput{Body: io.NopCloser(strings.NewReader(body))}, nil)

	_, err := getObj(hc)

	assert.NoError(t, err)
}
