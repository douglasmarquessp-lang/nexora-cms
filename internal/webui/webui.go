// Package webui embeds the compiled Admin SPA (Vite build output) into the
// API binary so a single Railway service serves both the API and the panel
// on the same origin — no CORS, no proxy, no second service.
//
// The production build (root Dockerfile, stage web-builder) runs
// `npm run build` in web/ and copies the result over internal/webui/dist
// BEFORE `go build`, so the binary embeds the real files. The repo commits
// internal/webui/dist/.gitkeep so `go build`/`go test` work without a Node
// build; in that case the embedded FS is empty and the SPA returns a 404
// with a hint (development uses `npm run dev` in web/ on :3000 instead).
package webui

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// SPAHandler returns the handler that serves the embedded Admin SPA with
// history-API fallback: any unknown non-API path returns index.html so
// React Router (/admin, /admin/login, ...) works on a static serving.
func SPAHandler() http.Handler {
	return NewSPAHandler(mustSub(distFS, "dist"))
}

// NewSPAHandler builds the SPA handler over an arbitrary filesystem
// (used by tests).
func NewSPAHandler(fsys fs.FS) http.Handler {
	return spaHandler{fsys: fsys}
}

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic("webui: cannot open embedded dist: " + err.Error())
	}
	return sub
}

type spaHandler struct {
	fsys fs.FS
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, `{"error":{"code":"METHOD_NOT_ALLOWED","message":"method not allowed"}}`, http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// Never fall back to the SPA for API paths: unmatched /api/* requests
	// keep a JSON 404 so API clients can rely on the error shape.
	if path == "api" || strings.HasPrefix(path, "api/") {
		http.Error(w, `{"error":{"code":"NOT_FOUND","message":"route not found"}}`, http.StatusNotFound)
		return
	}

	data, err := fs.ReadFile(h.fsys, path)
	if err != nil {
		data, err = fs.ReadFile(h.fsys, "index.html")
		if err != nil {
			http.Error(w, "webui: frontend not embedded - build the admin SPA (see internal/webui)", http.StatusNotFound)
			return
		}
		path = "index.html"
	}

	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(path)))
	if strings.HasPrefix(path, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		_, _ = w.Write(data)
	}
}
