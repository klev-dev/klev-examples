package main

import (
	"context"
	"fmt"
	"os"
	"time"

	api "github.com/klev-dev/klev-api-go"
)

func main() {
	fmt.Println(hello(context.Background()))
}

func hello(ctx context.Context) error {
	cfg := api.NewConfig(os.Getenv("KLEV_TOKEN_DEMO"))
	client := api.New(cfg)

	log, err := client.LogCreate(ctx, api.LogCreate{})
	if err != nil {
		return err
	}

	_, err = client.Post(ctx, log.LogID, time.Time{}, nil, []byte("hello world!"))
	if err != nil {
		return err
	}

	msg, err := client.Get(ctx, log.LogID, 0)
	if err != nil {
		return err
	}
	fmt.Println(string(msg.Value))

	return nil
}
