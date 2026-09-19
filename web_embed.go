package dotsynx

import (
	"embed"
	"io/fs"
)

//go:embed web/dist/*
var WebDistFS embed.FS

// GetWebDistFS returns an fs.FS rooted at web/dist
func GetWebDistFS() (fs.FS, error) {
	return fs.Sub(WebDistFS, "web/dist")
}
