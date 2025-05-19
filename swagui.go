package swagui

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
)

//go:embed swaggerui/*

var staticFiles embed.FS

var swaggerTemplate *template.Template

func init() {
	// Инициализируем шаблон Swagger UI при запуске
	subFS, err := fs.Sub(staticFiles, "swaggerui")
	if err != nil {
		panic(fmt.Errorf("failed to create sub filesystem: %w", err))
	}

	swig, err := fs.ReadFile(subFS, "swaggerui.html")
	if err != nil {
		panic(fmt.Errorf("failed to read swaggerui.html: %w", err))
	}

	swaggerTemplate = template.Must(template.New("swaggerui").Parse(string(swig)))
}

// SetupSwagger configures an HTTP handler to serve Swagger UI and OpenAPI specification.
//
// Parameters:
//   - docsPath: the URL path where Swagger UI will be available (e.g., "/api/docs")
//   - openapiFile: the content of the OpenAPI specification in YAML format
//
// Returns:
//
//	An http.Handler that can be registered with an HTTP server
//
// Sets up the following routes:
//  1. /openapi.yaml - serves the provided OpenAPI specification
//  2. {docsPath}/swaggerui(.html)? - serves the main Swagger UI page
//  3. {docsPath}/oauth2-redirect(.html)? - serves the OAuth2 redirect page
//  4. {docsPath}/* - serves all other static files for Swagger UI
//
// Features:
//   - Uses embedded files from the "swaggerui" directory
//   - Automatically sets correct Content-Type headers
//   - Supports both paths with and without .html extension
//   - Returns 404 for the root {docsPath} to avoid ambiguous routing
//
// Example usage:
//
//	openapiSpec, _ := os.ReadFile("openapi.yaml")
//	swaggerHandler := swagui.SetupSwagger("/api/docs", openapiSpec)
//	http.Handle("/", swaggerHandler)
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

	path := filepath.Join(docsPath, "oauth2-redirect")
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, subFS, "oauth2-redirect.html")
	})
	mux.HandleFunc(path+".html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, subFS, "oauth2-redirect.html")
	})

	// Обработчик для swaggerui.html с использованием шаблона
	openAPIPath := "/openapi.yaml"
	swaggerHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := struct {
			OpenAPIPath string
		}{
			OpenAPIPath: openAPIPath,
		}

		if err := swaggerTemplate.Execute(w, data); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}

	mux.HandleFunc(filepath.Join(docsPath, "swaggerui"), swaggerHandler)
	mux.HandleFunc(filepath.Join(docsPath, "swaggerui.html"), swaggerHandler)

	// Обработчик для остальных статических файлов
	fsHandler := http.FileServerFS(subFS)
	mux.Handle(docsPath, http.StripPrefix(docsPath, fsHandler))

	return mux
}
