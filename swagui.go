package swagui

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
)

//go:embed swaggerui/*
var staticFiles embed.FS

func SetupSwagger(docsPath string, openapiFile []byte) http.Handler {
	mux := http.NewServeMux()

	// Обработчик для /openapi.yaml
	mux.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(openapiFile)
	})

	// Создаем под-FS для директории swaggerui
	subFS, err := fs.Sub(staticFiles, "swaggerui")
	if err != nil {
		panic(fmt.Errorf("failed to create sub filesystem: %w", err))
	}

	// Обработчики для конкретных файлов
	for _, route := range []struct {
		Name     string
		Filename string
	}{
		{"swaggerui", "swaggerui.html"},
		{"oauth2-redirect", "oauth2-redirect.html"},
	} {
		path := filepath.Join(docsPath, route.Name)
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, subFS, route.Filename)
		})
		mux.HandleFunc(path+".html", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, subFS, route.Filename)
		})
	}

	// Обработчик для остальных статических файлов
	fsHandler := http.FileServerFS(subFS)
	mux.Handle(docsPath, http.StripPrefix(docsPath, fsHandler))

	return mux
}
