package client

import (
	"bytes"
	"fmt"
	"github.com/forta-network/forta-core-go/security"
	"io/ioutil"
	"net/http"
)

const urlPattern = "https://research.forta.network/database/%s/%s"

type Client interface {
	Get(scope Scope, objID string) ([]byte, error)
	Put(scope Scope, objID string, payload []byte) error
	Del(scope Scope, objID string) error
}

type Scope string

var ScopeBot Scope = "bot"
var ScopeScanner Scope = "scanner"

type client struct {
	botID      string
	keyDir     string
	passphrase string
}

func (c *client) Put(scope Scope, objID string, payload []byte) error {
	req, err := http.NewRequest("PUT", fmt.Sprintf(urlPattern, scope, objID), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	if err := c.addAuth(req); err != nil {
		return err
	}

	hc := &http.Client{}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("response %d", resp.StatusCode)
	}
	return nil
}

func (c *client) Del(scope Scope, objID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf(urlPattern, scope, objID), nil)
	if err != nil {
		return err
	}
	if err := c.addAuth(req); err != nil {
		return err
	}

	hc := &http.Client{}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("response %d", resp.StatusCode)
	}
	return nil
}

func (c *client) Get(scope Scope, objID string) ([]byte, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf(urlPattern, scope, objID), nil)
	if err != nil {
		return nil, err
	}
	if err := c.addAuth(req); err != nil {
		return nil, err
	}

	hc := &http.Client{}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("response %d", resp.StatusCode)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (c *client) addAuth(r *http.Request) error {
	token, err := c.token()
	if err != nil {
		return err
	}
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	return nil
}

func (c *client) token() (string, error) {
	k, err := security.LoadKeyWithPassphrase(c.keyDir, c.passphrase)
	if err != nil {
		panic(err)
	}
	return security.CreateScannerJWT(k, map[string]interface{}{
		"bot-id": c.botID,
	})
}

func NewClient(botID, keyDir, passphrase string) (Client, error) {
	return &client{
		botID:      botID,
		keyDir:     keyDir,
		passphrase: passphrase,
	}, nil
}
