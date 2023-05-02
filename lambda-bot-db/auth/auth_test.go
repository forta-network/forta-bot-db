package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/forta-network/forta-core-go/registry"
	mock_registry "github.com/forta-network/forta-core-go/registry/mocks"
	"github.com/forta-network/forta-core-go/security"
	"github.com/golang-jwt/jwt/v4"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	mock_store "forta-bot-db/store/mocks"
)

const testOwner = "0x8eedD1358997A3B48406cD5335aE9438C28b4128"
const testScanner = "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
const testBotID = "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefc0e4493c993e060e89c09ed1"
const testKey = "secrets.json"

var testErr = errors.New("nope")

var authHeader = map[string]string{
	"authorization": "bearer test",
}

func testReq(method string, pathParams map[string]string, headers map[string]string) events.APIGatewayV2HTTPRequest {
	return events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: method,
			},
		},
		Headers:        headers,
		PathParameters: pathParams,
	}
}
func testToken(botID, scanner string) *security.ScannerToken {
	return &security.ScannerToken{
		Scanner: scanner,
		Token: &jwt.Token{
			Claims: jwt.MapClaims{
				"bot-id": botID,
			},
			Valid: true,
		},
	}
}

func testParams(scope, key string) map[string]string {
	return map[string]string{
		"scope": scope,
		"key":   key,
	}
}

func TestAuthorize(t *testing.T) {
	type given struct {
		Scope   string
		Request events.APIGatewayV2HTTPRequest
	}
	type when struct {
		AuthErr  error
		Token    *security.ScannerToken
		Assigned bool
		Enabled  bool
		Agent    *registry.Agent
	}
	type expect struct {
		Error        error
		HandlerCtx   *HandlerCtx
		ObjectKey    string
		ObjectKeyErr error
	}
	type keyTest struct {
		Given  given
		When   when
		Expect expect
	}

	tests := []keyTest{
		{
			Given: given{
				Scope:   "owner",
				Request: testReq("GET", testParams("owner", testKey), authHeader),
			},
			When: when{
				Token:    testToken(testBotID, testScanner),
				Assigned: true,
				Enabled:  true,
				Agent:    &registry.Agent{Owner: testOwner},
			},
			Expect: expect{
				ObjectKey: fmt.Sprintf("%s/%s/%s", "owner", strings.ToLower(testOwner), testKey),
				HandlerCtx: &HandlerCtx{
					BotID:   testBotID,
					Scanner: testScanner,
					Owner:   testOwner,
					PathKey: testKey,
					Scope:   ScopeOwner,
				},
			},
		},
		{
			Given: given{
				Scope:   "bot",
				Request: testReq("GET", testParams("bot", testKey), authHeader),
			},
			When: when{
				Token:    testToken(testBotID, testScanner),
				Assigned: true,
				Enabled:  true,
			},
			Expect: expect{
				ObjectKey: fmt.Sprintf("%s/%s", testBotID, testKey),
				HandlerCtx: &HandlerCtx{
					BotID:   testBotID,
					Scanner: testScanner,
					PathKey: testKey,
					Scope:   ScopeBot,
				},
			},
		},
		{
			Given: given{
				Scope:   "scanner",
				Request: testReq("GET", testParams("scanner", testKey), authHeader),
			},
			When: when{
				Token:    testToken(testBotID, testScanner),
				Assigned: true,
				Enabled:  true,
			},
			Expect: expect{
				ObjectKey: fmt.Sprintf("%s/%s/%s", testBotID, testScanner, testKey),
				HandlerCtx: &HandlerCtx{
					BotID:   testBotID,
					Scanner: testScanner,
					PathKey: testKey,
					Scope:   ScopeScanner,
				},
			},
		},
		{
			Given: given{
				Scope:   "scanner",
				Request: testReq("GET", testParams("scanner", testKey), authHeader),
			},
			When: when{
				AuthErr: testErr,
			},
			Expect: expect{
				Error: testErr,
			},
		},
		{
			Given: given{
				Scope:   "scanner",
				Request: testReq("GET", testParams("scanner", testKey), authHeader),
			},
			When: when{
				Token:    testToken(testBotID, testScanner),
				Assigned: false,
				Enabled:  true,
			},
			Expect: expect{
				Error: ErrNotAssigned,
			},
		},
		{
			Given: given{
				Scope:   "scanner",
				Request: testReq("GET", testParams("scanner", testKey), authHeader),
			},
			When: when{
				Token:    testToken(testBotID, testScanner),
				Assigned: true,
				Enabled:  false,
			},
			Expect: expect{
				Error: ErrNotEnabled,
			},
		},
	}
	for _, test := range tests {
		jwtVerifier = func(tokenString string) (*security.ScannerToken, error) {
			return test.When.Token, test.When.AuthErr
		}
		ctrl := gomock.NewController(t)
		r := mock_registry.NewMockClient(ctrl)
		d := mock_store.NewMockDynamoDB(ctrl)

		a := &Authorizer{
			r:     r,
			d:     d,
			table: "table",
		}
		if test.When.AuthErr == nil {
			d.EXPECT().GetItem(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
			r.EXPECT().IsEnabledScanner(gomock.Any()).Return(test.When.Enabled, nil).Times(1)
			if test.When.Enabled {
				r.EXPECT().IsAssigned(gomock.Any(), gomock.Any()).Return(test.When.Assigned, nil).Times(1)
			}

			if test.When.Assigned && test.Given.Scope == "owner" {
				r.EXPECT().GetAgent(gomock.Any()).Return(test.When.Agent, nil).Times(1)
			}
		}

		if test.Expect.HandlerCtx != nil {
			d.EXPECT().PutItem(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
		}

		hc, err := a.Authorize(context.Background(), test.Given.Request)
		assert.ErrorIs(t, test.Expect.Error, err)
		if test.Expect.HandlerCtx == nil {
			assert.Nil(t, hc)
		} else {
			assert.NotNil(t, hc)
			assert.Equal(t, test.Expect.HandlerCtx.Scope, hc.Scope)
			assert.Equal(t, test.Expect.HandlerCtx.Scanner, hc.Scanner)
			assert.Equal(t, test.Expect.HandlerCtx.BotID, hc.BotID)
			assert.Equal(t, test.Expect.HandlerCtx.PathKey, hc.PathKey)

			key, err := hc.GetObjectKey()
			if test.Expect.ObjectKeyErr == nil {
				assert.NoError(t, err)
				assert.Equal(t, test.Expect.ObjectKey, key)
			} else {
				assert.Error(t, err)
			}
		}
	}
}
