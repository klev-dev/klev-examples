package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	api "github.com/klev-dev/klev-api-go"
	"github.com/klev-dev/kleverr"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	messages := make(chan api.PublishMessage, 32)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return tailJournal(ctx, messages)
	})

	g.Go(func() error {
		return publishBatched(ctx, messages)
	})

	if err := g.Wait(); err != nil {
		panic(err.Error())
	}
}

func tailJournal(ctx context.Context, messages chan<- api.PublishMessage) error {
	defer close(messages)

	cmd := exec.CommandContext(ctx, "/usr/bin/journalctl", "--system", "-f", "-o", "json")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return kleverr.Ret(err)
	}
	if err := cmd.Start(); err != nil {
		return kleverr.Ret(err)
	}

	reader := bufio.NewReader(stdout)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return kleverr.Ret(err)
		}
		messages <- api.NewPublishMessageValue(strings.TrimSpace(line))
	}
}

func publishBatched(ctx context.Context, messages <-chan api.PublishMessage) error {
	cfg := api.NewConfig(os.Getenv("KLEV_TOKEN_DEMO"))
	client := api.New(cfg)

	log, err := client.LogCreate(ctx, api.LogCreate{
		Metadata: fmt.Sprintf(`{"source": "journal", "unit": "system", "start": %d}`, time.Now().Unix()),
	})
	if err != nil {
		return kleverr.Ret(err)
	}

	var pending []api.PublishMessage

	var publishAny = func() (bool, error) {
		publish := len(pending) > 0
		if publish {
			if _, err := client.Publish(ctx, log.LogID, pending); err != nil {
				return false, kleverr.Ret(err)
			}
			pending = nil
		}
		return publish, nil
	}

	var publishBatch = func() error {
		if len(pending) > 24 {
			if _, err := client.Publish(ctx, log.LogID, pending[0:24]); err != nil {
				return kleverr.Ret(err)
			}
			pending = slices.Delete(pending, 0, 24)
		}
		return nil
	}

	for {
		select {
		case msg, ok := <-messages:
			if ok {
				// received a new message, append and try publish
				pending = append(pending, msg)
				if err := publishBatch(); err != nil {
					return err
				}
			} else {
				// messages channel is closing, publish anything pending
				_, err := publishAny()
				return err
			}
		default:
			// no new messages, try publishing anything pending
			if pub, err := publishAny(); err != nil {
				return err
			} else if !pub {
				// no pending messages, wait for any change
				msg, ok := <-messages
				if ok {
					pending = append(pending, msg)
				} else {
					return nil
				}
			}
		}
	}
}
