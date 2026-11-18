package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
