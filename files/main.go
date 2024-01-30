package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/klev-dev/klev-api-go"
	"github.com/klev-dev/klev-api-go/logs"
	"github.com/klev-dev/klev-api-go/messages"
	"github.com/spf13/cobra"
)

type fileMetadata struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type files struct {
	logs     *logs.Client
	messages *messages.Client
}

func (f *files) upload(ctx context.Context, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return fmt.Errorf("cannot upload directory")
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	dataLen := int64(len(data))

	var msgs []klev.PublishMessage
	for low := int64(0); low < dataLen; low += 64 * 1024 {
		high := low + 64*1024
		if high > dataLen {
			high = dataLen
		}

		msgs = append(msgs, klev.PublishMessage{
			Value: data[low:high],
		})
	}

	md, err := json.Marshal(fileMetadata{Name: filename, Size: dataLen})
	if err != nil {
		return err
	}

	log, err := f.logs.Create(ctx, klev.LogCreateParams{
		Metadata: string(md),
	})
	if err != nil {
		return err
	}

	for low := 0; low < len(msgs); low += 32 {
		high := low + 32
		if high > len(msgs) {
			high = len(msgs)
		}

		if _, err := f.messages.Publish(ctx, log.LogID, msgs[low:high]); err != nil {
			return err
		}
	}

	fmt.Printf("Done. Upload log: %s\n", log.LogID)

	return nil
}

func (f *files) download(ctx context.Context, logID klev.LogID, filename string) error {
	log, err := f.logs.Get(ctx, logID)
	if err != nil {
		return err
	}
	var md fileMetadata
	if err := json.Unmarshal([]byte(log.Metadata), &md); err != nil {
		return err
	}

	var data = make([]byte, 0, md.Size)
	offset := klev.OffsetOldest
	for {
		next, msgs, err := f.messages.Consume(ctx, logID,
			klev.ConsumeOffset(offset), klev.ConsumeLen(32))
		if err != nil {
			return err
		}
		if next == offset {
			break
		}
		offset = next

		for _, msg := range msgs {
			data = append(data, msg.Value...)
		}
	}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("could not write: %w", err)
	}

	return nil
}

func main() {
	cfg := klev.NewConfig(os.Getenv("KLEV_TOKEN_DEMO"))
	f := &files{
		logs:     logs.New(cfg),
		messages: messages.New(cfg),
	}

	rootCmd := &cobra.Command{
		Use:   "files",
		Short: "upload/download files via klev",
	}
	rootCmd.AddCommand(&cobra.Command{
		Use:   "upload",
		Short: "upload files to klev",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return f.upload(cmd.Context(), args[0])
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "download",
		Short: "download files from klev",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return f.download(cmd.Context(), klev.LogID(args[0]), args[1])
		},
	})

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
