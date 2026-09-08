package httpapi

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed web
var webFS embed.FS

func assetsHandler() http.Handler {
	sub, err := fs.Sub(webFS, "web/assets")
	if err != nil {
		panic(err)
	}
	return http.StripPrefix("/assets/", http.FileServer(http.FS(sub)))
}

func (s *Server) indexFor(mode string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		data, err := webFS.ReadFile("web/index.html")
		if err != nil {
			http.Error(w, "ui not found", http.StatusNotFound)
			return
		}
		page := strings.ReplaceAll(string(data), "__OTKLIK_UI_MODE__", mode)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page))
	}
}
