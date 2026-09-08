package httpapi

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// Встроенное дерево UI: index.html + статические ассеты (css/js).
//go:embed web
var webFS embed.FS

// assetsHandler раздаёт /assets/* из встроенной ФС (http.FileServer сам
// выставляет Content-Type по расширению и умеет ETag/If-Modified-Since).
func assetsHandler() http.Handler {
	sub, err := fs.Sub(webFS, "web/assets")
	if err != nil {
		panic(err) // невозможно при корректной директории web/assets в репо
	}
	return http.StripPrefix("/assets/", http.FileServer(http.FS(sub)))
}

// indexFor раздаёт встроенный UI в заданном режиме. Страница содержит
// плейсхолдер __OTKLIK_UI_MODE__, который заменяется на "applicant"
// (публичный порт, только заявитель) или "staff" (служебный порт, только
// сотрудники). Дальше страница догружает css/js из /assets/.
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

