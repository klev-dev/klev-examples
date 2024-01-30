package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/klev-dev/klev-api-go"
	"github.com/klev-dev/klev-api-go/clients"
)

func main() {
	fmt.Println(hello(context.Background()))
}

func hello(ctx context.Context) error {
	cfg := klev.NewConfig(os.Getenv("KLEV_TOKEN_DEMO"))
	client := clients.New(cfg)

	log, err := client.Logs.Create(ctx, klev.LogCreateParams{})
	if err != nil {
		return err
	}

	_, err = client.Messages.Post(ctx, log.LogID, time.Time{}, nil, []byte("hello world!"))
	if err != nil {
		return err
	}

	msg, err := client.Messages.GetByOffset(ctx, log.LogID, 0)
	if err != nil {
		return err
	}
	fmt.Println(string(msg.Value))

	return nil
}
