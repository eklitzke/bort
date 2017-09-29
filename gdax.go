package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	gdaxBase    = "https://api.gdax.com"
	targetQuote = "USD"
)

// Product represents a product on GDAX
type Product struct {
	Id            string `json:"id"`
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
	DisplayName   string `json:"display_name"`
}

type byProduct []Product

func (p byProduct) Len() int           { return len(p) }
func (p byProduct) Less(i, j int) bool { return p[i].Id < p[j].Id }
func (p byProduct) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type RawTicker struct {
	Price  string `json:"price"`
	Volume string `json:"volume"`
	Time   string `json:"time"`
}

type Ticker struct {
	Price  float64
	Volume float64
	Time   time.Time
}

type RawStats struct {
	Open   string `json:"open"`
	High   string `json:"high"`
	Low    string `json:"low"`
	Volume string `json:"volume"`
}

type Stats struct {
	Ticker    Ticker
	Open      float64
	High      float64
	Low       float64
	ChangePct float64
}

func (t *Ticker) vintage() time.Duration {
	return time.Now().UTC().Sub(t.Time)
}

func get(path string) ([]byte, error) {
	resp, err := http.Get(gdaxBase + path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Expected status %d, instead got %d; msg was %s",
			http.StatusOK, resp.StatusCode, resp.Status)
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func getProducts() ([]Product, error) {
	body, err := get("/products")
	if err != nil {
		return nil, err
	}
	var res, ret []Product
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	for _, product := range res {
		if product.QuoteCurrency == targetQuote {
			ret = append(ret, product)
		}
	}
	return ret, nil
}

func getTicker(productId string) (Ticker, error) {
	body, err := get(fmt.Sprintf("/products/%s/ticker", productId))
	if err != nil {
		return Ticker{}, err
	}

	var raw RawTicker
	if err := json.Unmarshal(body, &raw); err != nil {
		return Ticker{}, err
	}

	var tick Ticker
	if price, err := strconv.ParseFloat(raw.Price, 64); err == nil {
		tick.Price = price
	}
	if vol, err := strconv.ParseFloat(raw.Volume, 64); err == nil {
		tick.Volume = vol
	}
	timeStr := raw.Time[:strings.IndexByte(raw.Time, '.')]
	if tv, err := time.Parse("2006-01-02T15:04:05", timeStr); err == nil {
		tick.Time = tv
	}
	return tick, nil
}

func getStats(productId string) (Stats, error) {
	ticker, err := getTicker(productId)
	if err != nil {
		return Stats{}, err
	}
	stats := Stats{Ticker: ticker}

	body, err := get(fmt.Sprintf("/products/%s/stats", productId))
	if err != nil {
		return stats, err
	}

	var raw RawStats
	if err := json.Unmarshal(body, &raw); err != nil {
		return stats, err
	}
	if open, err := strconv.ParseFloat(raw.Open, 64); err == nil {
		stats.Open = open
	}
	if low, err := strconv.ParseFloat(raw.Low, 64); err == nil {
		stats.Low = low
	}
	if high, err := strconv.ParseFloat(raw.High, 64); err == nil {
		stats.High = high
	}
	stats.ChangePct = 100 * (stats.Ticker.Price/stats.Open - 1)
	return stats, nil
}
