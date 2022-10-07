package main

import (
	"forta-bot-db/client"
	"os"
)

func main() {
	botID := os.Getenv("BOT_ID")
	dir := os.Getenv("FORTA_DIRECTORY")
	pp := os.Getenv("FORTA_PASSPHRASE")
	c, err := client.NewClient(botID, dir, pp)
	if err != nil {
		panic(err)
	}

	key := "test"
	payload := "payload"
	scope := client.ScopeBot

	if err := c.Put(scope, key, []byte(payload)); err != nil {
		panic(err)
	}

	resp, err := c.Get(scope, key)
	if err != nil {
		panic(err)
	}
	if string(resp) != payload {
		panic("response != payload")
	}

	if err := c.Del(scope, key); err != nil {
		panic(err)
	}

}
