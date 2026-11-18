package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
)

const (
	alphabet       = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	cacheTTL       = time.Hour
	cacheKeyPrefix = "link:"
)

func encodeBase62(num uint64) string {
	if num == 0 {
		return "0"
	}
	var result string
	for num > 0 {
		result = string(alphabet[num%62]) + result
		num = num / 62
	}
	return result
}

// ShortCode es la key base62 de un link; el tipo la distingue de otras strings.
type ShortCode string

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Code     ShortCode `json:"code"`
	ShortURL string    `json:"short_url"`
	LongURL  string    `json:"long_url"`
}

func (app *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/shorten", app.handleShorten)
	mux.HandleFunc("GET /api/stats/{code}", app.handleStats)
	mux.HandleFunc("GET /health", app.handleHealth)
	mux.HandleFunc("/", app.handleRedirect)
	return withCORS(mux)
}

// El frontend corre en otro puerto durante el desarrollo, así que el browser
// hace preflight antes del POST. CORS abierto alcanza para este prototipo local.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (app *App) handleShorten(w http.ResponseWriter, r *http.Request) {
	var request shortenRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}

	longURL, err := normalizeURL(request.URL)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	id, err := app.generator.NextID()
	if err != nil {
		log.Printf("generar id: %v", err)
		writeError(w, http.StatusInternalServerError, "no se pudo generar la key")
		return
	}

	key := ShortCode(encodeBase62(uint64(id)))
	if _, err := app.db.ExecContext(r.Context(), "INSERT INTO links (short_code, long_url) VALUES ($1, $2)", string(key), longURL); err != nil {
		log.Printf("guardar link: %v", err)
		writeError(w, http.StatusInternalServerError, "no se pudo guardar el link")
		return
	}

	app.cacheLink(r, key, longURL)
	writeJSON(w, http.StatusOK, app.shortenResponse(key, longURL))
}

func (app *App) handleRedirect(w http.ResponseWriter, r *http.Request) {
	code := ShortCode(strings.TrimPrefix(r.URL.Path, "/"))
	if code == "" || !validCode(code) {
		http.NotFound(w, r)
		return
	}

	key := cacheKey(code)
	longURL, err := app.cache.Get(r.Context(), key).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			log.Printf("leer caché %s: %v", key, err)
		}
		err = app.db.QueryRowContext(r.Context(), "SELECT long_url FROM links WHERE short_code = $1", string(code)).Scan(&longURL)
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			log.Printf("buscar link %s: %v", code, err)
			writeError(w, http.StatusInternalServerError, "no se pudo resolver el link")
			return
		}
		app.cacheLink(r, code, longURL)
	}

	app.recordClick(code, countryFrom(r))
	http.Redirect(w, r, longURL, http.StatusFound)
}

func (app *App) handleStats(w http.ResponseWriter, r *http.Request) {
	code := ShortCode(r.PathValue("code"))
	if !validCode(code) {
		writeError(w, http.StatusBadRequest, "código inválido")
		return
	}

	stats, err := app.clickhouse.Stats(r.Context(), code)
	if err != nil {
		log.Printf("stats %s: %v", code, err)
		writeError(w, http.StatusInternalServerError, "no se pudieron leer las estadísticas")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (app *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (app *App) recordClick(code ShortCode, country string) {
	go func() {
		event := ClickEvent{Code: code, Country: country, ClickedAt: time.Now().UTC()}
		payload, err := json.Marshal(event)
		if err != nil {
			log.Printf("serializar click: %v", err)
			return
		}

		wait := func(d time.Duration) error { time.Sleep(d); return nil }
		err = retry(wait, func() error {
			_, _, err := app.producer.SendMessage(&sarama.ProducerMessage{
				Topic: app.cfg.KafkaTopic,
				Key:   sarama.StringEncoder(string(code)),
				Value: sarama.ByteEncoder(payload),
			})
			return err
		})
		if err != nil {
			log.Printf("no se pudo publicar el click de %s: %v", code, err)
		}
	}()
}

func cacheKey(code ShortCode) string {
	return cacheKeyPrefix + string(code)
}

func (app *App) cacheLink(r *http.Request, code ShortCode, longURL string) {
	if err := app.cache.Set(r.Context(), cacheKey(code), longURL, cacheTTL).Err(); err != nil {
		log.Printf("cachear link %s: %v", code, err)
	}
}

func (app *App) shortenResponse(code ShortCode, longURL string) shortenResponse {
	return shortenResponse{
		Code:     code,
		ShortURL: strings.TrimRight(app.cfg.BaseURL, "/") + "/" + string(code),
		LongURL:  longURL,
	}
}

func normalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("la URL no puede estar vacía")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", errors.New("URL inválida")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("la URL debe usar http o https")
	}
	if parsed.Host == "" {
		return "", errors.New("la URL debe tener un host")
	}
	return parsed.String(), nil
}

func validCode(code ShortCode) bool {
	if len(code) == 0 || len(code) > 32 {
		return false
	}
	for _, char := range code {
		if !strings.ContainsRune(alphabet, char) {
			return false
		}
	}
	return true
}

func countryFrom(r *http.Request) string {
	for _, header := range []string{"CF-IPCountry", "X-Country"} {
		if value := strings.TrimSpace(r.Header.Get(header)); value != "" {
			return strings.ToUpper(value)
		}
	}
	return "XX"
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("escribir respuesta: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
