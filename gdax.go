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

func (t *Ticker) vintage() time.Duration {
	return time.Now().UTC().Sub(t.Time)
}

func getProducts() ([]Product, error) {
	resp, err := http.Get(gdaxBase + "/products")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Expected status %d, instead got %d; msg was %s",
			http.StatusOK, resp.StatusCode, resp.Status)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var res, ret []Product
	json.Unmarshal(body, &res)
	for _, product := range res {
		if product.QuoteCurrency == targetQuote {
			ret = append(ret, product)
		}
	}
	return ret, nil
}

func fetchTicker(productId string) (Ticker, error) {
	resp, err := http.Get(gdaxBase + fmt.Sprintf("/products/%s/ticker", productId))
	if err != nil {
		return Ticker{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Ticker{}, fmt.Errorf("Expected status %d, instead got %d; msg was %s",
			http.StatusOK, resp.StatusCode, resp.Status)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Ticker{}, err
	}

	var raw RawTicker
	json.Unmarshal(body, &raw)

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
