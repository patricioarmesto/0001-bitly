package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ClickHouse se usa por su interfaz HTTP: la query viaja en el body y los
// parámetros como param_*, así que no hace falta un cliente oficial acá.
type ClickHouse struct {
	baseURL  string
	user     string
	password string
	client   *http.Client
}

type ClickEvent struct {
	Code      string    `json:"code"`
	Country   string    `json:"country"`
	ClickedAt time.Time `json:"clicked_at"`
}

type CountryStat struct {
	Country string `json:"country"`
	Clicks  uint64 `json:"clicks"`
}

type DayStat struct {
	Day    string `json:"day"`
	Clicks uint64 `json:"clicks"`
}

type StatsResponse struct {
	Code      string        `json:"code"`
	Total     uint64        `json:"total"`
	ByCountry []CountryStat `json:"by_country"`
	ByDay     []DayStat     `json:"by_day"`
}

func NewClickHouse(baseURL, user, password string) *ClickHouse {
	return &ClickHouse{
		baseURL:  strings.TrimRight(baseURL, "/"),
		user:     user,
		password: password,
		client:   &http.Client{Timeout: 5 * time.Second},
	}
}

func (ch *ClickHouse) EnsureSchema(ctx context.Context) error {
	statements := []string{
		"CREATE DATABASE IF NOT EXISTS analytics",
		`CREATE TABLE IF NOT EXISTS analytics.clicks (
			code       String,
			country    String,
			clicked_at DateTime64(3, 'UTC')
		) ENGINE = MergeTree()
		ORDER BY (code, clicked_at)`,
	}
	for _, statement := range statements {
		if err := ch.exec(ctx, statement, nil, nil); err != nil {
			return fmt.Errorf("crear esquema: %w", err)
		}
	}
	return nil
}

func (ch *ClickHouse) InsertClick(ctx context.Context, event ClickEvent) error {
	query := `INSERT INTO analytics.clicks (code, country, clicked_at)
		VALUES ({code:String}, {country:String}, toDateTime64({ts:String}, 3, 'UTC'))`
	params := map[string]string{
		"code":    event.Code,
		"country": event.Country,
		"ts":      event.ClickedAt.UTC().Format("2006-01-02 15:04:05.000"),
	}

	wait := func(d time.Duration) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
			return nil
		}
	}
	return retry(wait, func() error {
		return ch.exec(ctx, query, params, nil)
	})
}

func (ch *ClickHouse) Stats(ctx context.Context, code string) (StatsResponse, error) {
	result := StatsResponse{
		Code:      code,
		ByCountry: []CountryStat{},
		ByDay:     []DayStat{},
	}

	var total struct {
		Data []struct {
			Total uint64 `json:"total"`
		} `json:"data"`
	}
	query := `SELECT count() AS total FROM analytics.clicks WHERE code = {code:String} FORMAT JSON`
	if err := ch.exec(ctx, query, map[string]string{"code": code}, &total); err != nil {
		return result, fmt.Errorf("total de clicks: %w", err)
	}
	if len(total.Data) > 0 {
		result.Total = total.Data[0].Total
	}

	var countries struct {
		Data []CountryStat `json:"data"`
	}
	query = `SELECT country, count() AS clicks
		FROM analytics.clicks
		WHERE code = {code:String}
		GROUP BY country
		ORDER BY clicks DESC
		FORMAT JSON`
	if err := ch.exec(ctx, query, map[string]string{"code": code}, &countries); err != nil {
		return result, fmt.Errorf("clicks por país: %w", err)
	}
	result.ByCountry = countries.Data

	var days struct {
		Data []DayStat `json:"data"`
	}
	query = `SELECT toString(toDate(clicked_at)) AS day, count() AS clicks
		FROM analytics.clicks
		WHERE code = {code:String}
		GROUP BY day
		ORDER BY day
		FORMAT JSON`
	if err := ch.exec(ctx, query, map[string]string{"code": code}, &days); err != nil {
		return result, fmt.Errorf("clicks por día: %w", err)
	}
	result.ByDay = days.Data

	return result, nil
}

func (ch *ClickHouse) exec(ctx context.Context, query string, params map[string]string, out any) error {
	endpoint := ch.baseURL + "/"
	values := url.Values{}
	// ClickHouse cita los UInt64 como strings en JSON; los counts del panel
	// se decodifican como números en Go.
	values.Set("output_format_json_quote_64bit_integers", "0")
	for key, value := range params {
		values.Set("param_"+key, value)
	}
	endpoint += "?" + values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(query))
	if err != nil {
		return err
	}
	if ch.user != "" {
		request.SetBasicAuth(ch.user, ch.password)
	}
	response, err := ch.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("ClickHouse %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	if out == nil {
		_, err = io.Copy(io.Discard, response.Body)
		return err
	}
	return json.NewDecoder(response.Body).Decode(out)
}
