package main

import (
	"context"
	"fmt"
	"os"
	"time"

	api "github.com/klev-dev/klev-api-go"
	"github.com/klev-dev/klev-api-go/client"
	"github.com/klev-dev/klev-api-go/logs"
)

func main() {
	fmt.Println(hello(context.Background()))
}

func hello(ctx context.Context) error {
	cfg := client.NewConfig(os.Getenv("KLEV_TOKEN_DEMO"))
	client := api.New(cfg)

	log, err := client.Logs.Create(ctx, logs.CreateParams{})
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
