package main

import (
	"context"
	"fmt"
	"os"
	"time"

	klev "github.com/klev-dev/klev-api-go"
)

func main() {
	fmt.Println(hello(context.Background()))
}

func hello(ctx context.Context) error {
	cfg := klev.NewConfig(os.Getenv("KLEV_TOKEN_DEMO"))
	client := klev.New(cfg)

	log, err := client.LogCreate(ctx, klev.LogIn{})
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
