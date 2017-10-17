package main

import (
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/dghubble/go-twitter/twitter"
)

var twitterStatus = regexp.MustCompile(`^https://twitter.com/(.*)/status/([0-9]+)$`)

func getTweetInfo(msg string) string {
	matches := twitterStatus.FindStringSubmatch(msg)
	if matches == nil {
		log.Printf("no matches!")
		return ""
	}
	log.Printf("matches = %v", matches)
	status := matches[2]
	statusId, err := strconv.ParseInt(status, 10, 64)
	if err != nil {
		return ""
	}

	httpClient := new(http.Client)
	client := twitter.NewClient(httpClient)
	tweet, resp, _ := client.Statuses.Show(statusId, nil)
	log.Print(tweet)
	log.Print(resp)
	return ""
}
