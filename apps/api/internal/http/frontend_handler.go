package httpx

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

func registerFrontendRoutes(r interface {
	Get(string, http.HandlerFunc)
	Handle(string, http.Handler)
}) {
	webDir := resolveWebDir()
	fileServer := http.FileServer(http.Dir(webDir))

	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, filepath.Join(webDir, "index.html"))
	})
	r.Handle("/*", fileServer)
}

func resolveWebDir() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Clean("../web")
	}
	base := filepath.Dir(currentFile)
	candidate := filepath.Clean(filepath.Join(base, "..", "..", "..", "web"))
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return filepath.Clean("../web")
}
