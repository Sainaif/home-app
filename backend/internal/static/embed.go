package static

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var staticFS embed.FS

// GetFileSystem returns an http.FileSystem for the embedded static files
// This is used by Fiber's filesystem middleware to serve the Vue.js SPA
func GetFileSystem() (http.FileSystem, error) {
	fsys, err := fs.Sub(staticFS, "dist")
	if err != nil {
		return nil, err
	}
	return http.FS(fsys), nil
}

// GetFS returns the raw embed.FS for direct access if needed
func GetFS() embed.FS {
	return staticFS
}

// HasStaticFiles checks if static files are embedded
// Returns false during development when dist/ doesn't exist
func HasStaticFiles() bool {
	entries, err := staticFS.ReadDir("dist")
	if err != nil {
		return false
	}
	return len(entries) > 0
}
