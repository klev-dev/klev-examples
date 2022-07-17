package main

import (
	"embed"
	"io/fs"
	"os"
)

//go:embed html
var templates embed.FS

func getTemplates(reload bool) (fs.FS, error) {
	if reload {
		return os.DirFS("chat/html"), nil
	}
	return fs.Sub(templates, "html")
}
