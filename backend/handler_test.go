package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/redis/go-redis/v9"
)

func TestCORS(t *testing.T) {
	handler := (&App{}).routes()

	preflight := httptest.NewRequest(http.MethodOptions, "/api/shorten", nil)
	preflight.Header.Set("Origin", "http://localhost:5173")
	preflight.Header.Set("Access-Control-Request-Method", "POST")
	preflight.Header.Set("Access-Control-Request-Headers", "Content-Type")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, preflight)

	if response.Code != http.StatusNoContent {
		t.Fatalf("preflight: esperaba 204, obtuve %d", response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got == "" {
		t.Error("preflight sin Access-Control-Allow-Origin")
	}

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Origin"); got == "" {
		t.Error("respuesta sin Access-Control-Allow-Origin")
	}
}

func TestHandleShortenUsaElGenerator(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	generator := NewGenerator(7)
	generator.now = func() int64 { return 63158400123 }
	generator.sleep = func(time.Duration) {}

	mock.ExpectExec(`INSERT INTO links \(short_code, long_url\) VALUES \(\$1, \$2\)`).
		WithArgs("jzgKpHNygg", "https://example.com/desde-generator").
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := &App{
		db:        db,
		generator: generator,
		cache:     redis.NewClient(&redis.Options{Addr: "127.0.0.1:0", DialTimeout: 100 * time.Millisecond, MaxRetries: -1}),
	}
	defer app.cache.Close()

	body := strings.NewReader(`{"url":"https://example.com/desde-generator"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/shorten", body)
	response := httptest.NewRecorder()
	app.handleShorten(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("esperaba 200, obtuve %d: %s", response.Code, response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
