package swagui

import (
	"bytes"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetupSwagger(t *testing.T) {
	// Инициализация шаблона для теста
	subFS, err := fs.Sub(staticFiles, "swaggerui")
	if err != nil {
		t.Fatalf("Failed to create sub filesystem: %v", err)
	}

	// Чтение оригинального шаблона
	swig, err := fs.ReadFile(subFS, "swaggerui.html")
	if err != nil {
		t.Fatalf("Failed to read swaggerui.html: %v", err)
	}

	// Парсинг шаблона
	swaggerTemplate := template.Must(template.New("swaggerui").Parse(string(swig)))

	// Генерация ожидаемого содержимого
	var expectedSwagContent bytes.Buffer
	data := struct {
		OpenAPIPath string
	}{
		OpenAPIPath: "/api/v1/docs/openapi.yaml",
	}
	if err := swaggerTemplate.Execute(&expectedSwagContent, data); err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	// Пример OpenAPI спецификации
	openapi := []byte(`openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0`)

	// Настройка обработчика
	handler, err := SetupSwagger("/api/v1/docs", openapi, ".yaml")
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(handler)
	defer ts.Close()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedHeader string
		expectedBody   []byte
	}{
		// ... другие тесты без изменений ...
		{
			name:           "Swagger UI",
			path:           "/api/v1/docs/swaggerui",
			expectedStatus: http.StatusOK,
			expectedHeader: "text/html; charset=utf-8",
			expectedBody:   expectedSwagContent.Bytes(),
		},
		{
			name:           "Static file - swaggerui.html",
			path:           "/api/v1/docs/swaggerui.html",
			expectedStatus: http.StatusOK,
			expectedHeader: "text/html; charset=utf-8",
			expectedBody:   expectedSwagContent.Bytes(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(ts.URL + tt.path)
			if err != nil {
				t.Errorf("Error making request to %s: %v", tt.path, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Status code mismatch for %s: expected %d, got %d",
					tt.path, tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectedHeader != "" {
				contentType := resp.Header.Get("Content-Type")
				if contentType != tt.expectedHeader {
					t.Errorf("Content-Type mismatch for %s: expected %s, got %s",
						tt.path, tt.expectedHeader, contentType)
				}
			}

			if tt.expectedBody != nil {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Errorf("Error reading response body for %s: %v", tt.path, err)
					return
				}

				if string(body) != string(tt.expectedBody) {
					t.Errorf("Response body mismatch for %s: expected length %d, got %d",
						tt.path, len(tt.expectedBody), len(body))

					// Выводим первые 100 символов для диагностики
					expectedSample := string(tt.expectedBody)
					if len(expectedSample) > 100 {
						expectedSample = expectedSample[:100] + "..."
					}

					gotSample := string(body)
					if len(gotSample) > 100 {
						gotSample = gotSample[:100] + "..."
					}

					t.Errorf("Expected sample: %s", expectedSample)
					t.Errorf("Got sample:      %s", gotSample)
				}
			} else if tt.expectedStatus == http.StatusOK && tt.expectedBody == nil {
				// Если ожидаем пустое тело
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Errorf("Error reading response body for %s: %v", tt.path, err)
					return
				}

				if len(body) > 0 {
					t.Errorf("Expected empty body for %s, got %d bytes", tt.path, len(body))
				}
			}
		})
	}
}
