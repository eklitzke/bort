package main

import (
	"fmt"
	"log"
	"strings"
	"time"
)

const (
	maxVintage = 60 * time.Second
)

// GDAXInfo represents info from GDAX
type GDAXInfo struct {
	products        []Product
	productToTicker map[string]Ticker
}

func makeGDAXInfo() *GDAXInfo {
	return &GDAXInfo{}
}

func (info *GDAXInfo) listProducts() string {
	if info.products == nil {
		products, err := getProducts()
		if err != nil {
			log.Printf("err %v when getting products", err)
			return ""
		}
		info.products = products
	}
	var productIds []string
	for _, p := range info.products {
		productIds = append(productIds, p.DisplayName)
	}
	return strings.Join(productIds, ", ")
}

func (info *GDAXInfo) getTicker(productId string) (*Ticker, error) {
	if info.productToTicker == nil {
		info.productToTicker = make(map[string]Ticker)
	}
	if tick, ok := info.productToTicker[productId]; ok {
		// use the cached value if it isn't too old
		if tick.vintage() < maxVintage {
			return &tick, nil
		}
	}
	tick, err := fetchTicker(productId)
	if err != nil {
		log.Printf("error %v when getting ticker for product %s", err, productId)
		return nil, err
	}
	info.productToTicker[productId] = tick
	return &tick, nil
}

func (info *GDAXInfo) getPrice(product string) (string, error) {
	tick, err := info.getTicker(product)
	if err != nil {
		log.Printf("err %v in getPrice()", err)
		return "", err
	}
	return fmt.Sprintf("$%1.2f", tick.Price), nil
}

func (info *GDAXInfo) getVolume(product string) (string, error) {
	tick, err := info.getTicker(product)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%1.2f", tick.Volume), nil
}
