package main

import (
	"fmt"
	"github.com/forta-network/forta-core-go/security"
	"io"
	"net/http"
	"os"
)

func main() {
	dir := os.Getenv("FORTA_DIR")
	pp := os.Getenv("FORTA_PASSPHRASE")
	key, err := security.LoadKeyWithPassphrase(dir, pp)
	if err != nil {
		panic(err)
	}

	token, err := security.CreateScannerJWT(key, map[string]interface{}{
		"bot-id": "0xfa7f60b5b1f9bb5758268506a978bacc17c42af6b7d0beb61ac64c681d9c9126",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(token)

	req, err := http.NewRequest("GET", "https://research.forta.network/database/key", nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}
