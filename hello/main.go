package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/klev-dev/klev-api-go"
	"github.com/klev-dev/klev-api-go/logs"
	"github.com/klev-dev/klev-api-go/messages"
)

func main() {
	fmt.Println(hello(context.Background()))
}

func hello(ctx context.Context) error {
	cfg := klev.NewConfig(os.Getenv("KLEV_TOKEN_DEMO"))
	logs := logs.New(cfg)
	messages := messages.New(cfg)

	log, err := logs.Create(ctx, klev.LogCreateParams{})
	if err != nil {
		return err
	}

	_, err = messages.Post(ctx, log.LogID, time.Time{}, nil, []byte("hello world!"))
	if err != nil {
		return err
	}

	msg, err := messages.GetByOffset(ctx, log.LogID, 0)
	if err != nil {
		return err
	}
	fmt.Println(string(msg.Value))

	return nil
}
