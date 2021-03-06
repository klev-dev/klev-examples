package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	klev "github.com/klev-dev/klev-api-go"
	"github.com/segmentio/ksuid"
	"github.com/spf13/cobra"
)

var klient *klev.Client

var rootCmd = &cobra.Command{
	Use:   "klev-example-files",
	Short: "upload/download files via klev",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cfg := klev.NewConfig(os.Getenv("KLEV_TOKEN_DEMO"))
		klient = klev.New(cfg)
	},
}

type fileMetadata struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "upload files to klev",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fileName := args[0]
		file, err := os.Open(fileName)
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

		md, err := json.Marshal(fileMetadata{Name: fileName, Size: dataLen})
		if err != nil {
			return err
		}

		log, err := klient.LogCreate(cmd.Context(), klev.LogIn{
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

			if _, err := klient.Publish(cmd.Context(), log.LogID, msgs[low:high]); err != nil {
				return err
			}
		}

		fmt.Printf("Done. Upload log: %s\n", log.LogID)

		return nil
	},
}

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "download files from klev",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		logID, err := ksuid.Parse(args[0])
		if err != nil {
			return err
		}

		log, err := klient.LogGet(cmd.Context(), logID)
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
			next, msgs, err := klient.Consume(cmd.Context(), logID, offset, 32)
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

		fileName := args[1]
		file, err := os.OpenFile(fileName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := file.Write(data); err != nil {
			return fmt.Errorf("could not write: %w", err)
		}

		return nil
	},
}

func main() {
	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(downloadCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
