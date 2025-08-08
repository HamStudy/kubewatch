package resources

import (
	"embed"
	"io/fs"
)

// EmbeddedFS contains all embedded resource definitions
//
//go:embed all:embedded
var EmbeddedFS embed.FS

// GetFS returns the embedded filesystem rooted at "embedded"
func GetFS() (fs.FS, error) {
	return fs.Sub(EmbeddedFS, "embedded")
}
