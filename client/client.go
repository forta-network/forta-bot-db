package client

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

const urlPattern = "%s/database/%s/%s"

var ErrNotFound = errors.New("not found")

type Client interface {
	Get(scope Scope, objID string) ([]byte, error)
	Put(scope Scope, objID string, payload []byte) error
	Del(scope Scope, objID string) error
}

type Scope string

var ScopeBot Scope = "bot"
var ScopeScanner Scope = "scanner"

var ScopeOwner Scope = "owner"

type client struct {
	apiHost        string
	jwtProviderUrl string
}

func gzipBytes(b []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(b)
	if err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func gunzipBytes(b []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func (c *client) Put(scope Scope, objID string, payload []byte) error {
	pl := payload
	if strings.HasSuffix(objID, ".gz") {
		gzipPayload, err := gzipBytes(payload)
		if err != nil {
			return err
		}
		pl = gzipPayload
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf(urlPattern, c.apiHost, scope, objID), bytes.NewReader(pl))
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
	if resp.StatusCode == 404 {
		return ErrNotFound
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("response %d", resp.StatusCode)
	}
	return nil
}

func (c *client) Del(scope Scope, objID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf(urlPattern, c.apiHost, scope, objID), nil)
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
	if resp.StatusCode == 404 {
		return ErrNotFound
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("response %d", resp.StatusCode)
	}
	return nil
}

func (c *client) Get(scope Scope, objID string) ([]byte, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf(urlPattern, c.apiHost, scope, objID), nil)
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
	if resp.StatusCode == 404 {
		return nil, ErrNotFound
	}
	if resp.StatusCode == 500 {
		log.WithError(err).Error("500 error...coercing to 404")
		return nil, ErrNotFound
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("response %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(objID, ".gz") {
		bts, err := gunzipBytes(b)
		if err != nil {
			return nil, err
		}
		b = bts
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
	// negotiate token
	res, err := http.Post(c.jwtProviderUrl, "", nil)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	var jwtResp CreateJWTResponse
	if err := json.NewDecoder(res.Body).Decode(&jwtResp); err != nil {
		return "", err
	}
	return jwtResp.Token, nil
}

func NewDefaultClient(apiHost string) (Client, error) {
	return NewClient(apiHost, os.Getenv("FORTA_JWT_PROVIDER_HOST"), os.Getenv("FORTA_JWT_PROVIDER_PORT"))
}

func NewClient(apiHost, jwtProviderHost, jwtProviderPort string) (Client, error) {
	return &client{
		apiHost:        apiHost,
		jwtProviderUrl: fmt.Sprintf("http://%s:%s/create", jwtProviderHost, jwtProviderPort),
	}, nil
}
