package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

type config struct {
	Port               string
	DatabaseURL        string
	RedisAddr          string
	KafkaBrokers       []string
	KafkaTopic         string
	ClickHouseURL      string
	ClickHouseUser     string
	ClickHousePassword string
	BaseURL            string
	WorkerID           int64
}

func loadConfig() config {
	return config{
		Port:               env("PORT", "8080"),
		DatabaseURL:        env("DATABASE_URL", "postgres://bitly:bitly@localhost:5432/bitly?sslmode=disable"),
		RedisAddr:          env("REDIS_ADDR", "localhost:6379"),
		KafkaBrokers:       strings.Split(env("KAFKA_BROKERS", "localhost:9092"), ","),
		KafkaTopic:         env("KAFKA_TOPIC", "clicks"),
		ClickHouseURL:      env("CLICKHOUSE_URL", "http://localhost:8123"),
		ClickHouseUser:     env("CLICKHOUSE_USER", "default"),
		ClickHousePassword: env("CLICKHOUSE_PASSWORD", ""),
		BaseURL:            env("BASE_URL", "http://localhost:8080"),
		WorkerID:           workerIDFromEnv(),
	}
}

func workerIDFromEnv() int64 {
	raw := env("WORKER_ID", "0")
	workerID, err := parseWorkerID(raw)
	if err != nil {
		log.Fatalf("%v", err)
	}
	return workerID
}

func parseWorkerID(raw string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("WORKER_ID debe ser un entero, obtuve %q", raw)
	}
	if value < 0 || value > maxWorkerID {
		return 0, fmt.Errorf("WORKER_ID debe estar entre 0 y %d, obtuve %d", maxWorkerID, value)
	}
	return value, nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

type App struct {
	cfg        config
	db         *sql.DB
	cache      *redis.Client
	producer   sarama.SyncProducer
	clickhouse *ClickHouse
	generator  *Generator
}

func main() {
	cfg := loadConfig()

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancelStartup()

	db, err := initPostgres(startupCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Postgres: %v", err)
	}
	defer db.Close()

	cache := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := waitFor(startupCtx, "Redis", func() error { return cache.Ping(startupCtx).Err() }); err != nil {
		log.Fatalf("Redis: %v", err)
	}
	defer cache.Close()

	producer, err := initProducer(startupCtx, cfg.KafkaBrokers)
	if err != nil {
		log.Fatalf("Kafka: %v", err)
	}
	defer producer.Close()

	clickhouse := NewClickHouse(cfg.ClickHouseURL, cfg.ClickHouseUser, cfg.ClickHousePassword)
	if err := clickhouse.EnsureSchema(startupCtx); err != nil {
		log.Fatalf("ClickHouse: %v", err)
	}

	app := &App{cfg: cfg, db: db, cache: cache, producer: producer, clickhouse: clickhouse, generator: NewGenerator(cfg.WorkerID)}

	appCtx, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()
	go app.consumeClicks(appCtx)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("API lista en :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("servidor HTTP: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown HTTP: %v", err)
	}
	cancelApp()
}

func initPostgres(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(32)
	db.SetMaxIdleConns(32)
	db.SetConnMaxLifetime(time.Hour)

	if err := waitFor(ctx, "Postgres", func() error { return db.PingContext(ctx) }); err != nil {
		db.Close()
		return nil, err
	}

	schema := `
		CREATE TABLE IF NOT EXISTS links (
			short_code TEXT PRIMARY KEY,
			long_url   TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_links_long_url ON links(long_url);
	`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func initProducer(ctx context.Context, brokers []string) (sarama.SyncProducer, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	cfg.Producer.RequiredAcks = sarama.WaitForLocal
	cfg.Producer.Retry.Max = 2

	var producer sarama.SyncProducer
	err := waitFor(ctx, "Kafka producer", func() error {
		created, err := sarama.NewSyncProducer(brokers, cfg)
		if err == nil {
			producer = created
		}
		return err
	})
	return producer, err
}

// Espera simple para los servicios que todavía están levantando. Sólo se usa al arrancar.
func waitFor(ctx context.Context, name string, fn func() error) error {
	var lastErr error
	for attempt := 0; ; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		delay := time.Duration(200*(attempt+1)) * time.Millisecond
		if delay > 2*time.Second {
			delay = 2 * time.Second
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s no respondió: %v (%w)", name, lastErr, ctx.Err())
		case <-time.After(delay):
		}
	}
}

// Alcance: Ejercicio de System Design. Decisiones intencionales: reintento simple con backoff, entrega at-least-once, publicación directa a Kafka sin Transactional Outbox. Ver README. Estos patrones están fuera de alcance a propósito para mantener el ejercicio legible. No son omisiones por desconocimiento.
func (app *App) consumeClicks(ctx context.Context) {
	cfg := sarama.NewConfig()
	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest

	initCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var group sarama.ConsumerGroup
	if err := waitFor(initCtx, "Kafka consumer group", func() error {
		created, err := sarama.NewConsumerGroup(app.cfg.KafkaBrokers, "clickhouse-writer", cfg)
		if err == nil {
			group = created
		}
		return err
	}); err != nil {
		log.Printf("consumer de Kafka: %v", err)
		return
	}
	defer group.Close()

	handler := &clickConsumer{app: app}
	for {
		if err := group.Consume(ctx, []string{app.cfg.KafkaTopic}, handler); err != nil && ctx.Err() == nil {
			log.Printf("consume: %v", err)
		}
		if ctx.Err() != nil {
			return
		}
	}
}

type clickConsumer struct {
	app *App
}

func (c *clickConsumer) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (c *clickConsumer) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (c *clickConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			var event ClickEvent
			if err := json.Unmarshal(message.Value, &event); err != nil {
				log.Printf("evento de click inválido, se descarta: %v", err)
				session.MarkMessage(message, "")
				continue
			}

			if err := c.app.clickhouse.InsertClick(session.Context(), event); err != nil {
				return fmt.Errorf("insertar click en ClickHouse: %w", err)
			}
			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}
