package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strings"

	irc "github.com/fluffle/goirc/client"
	yaml "gopkg.in/yaml.v2"
)

var credsPath = flag.String("creds", "", "Path to credentials")

var addressedToMeRe = regexp.MustCompile(`^bort[:,]`)

type BotInfo struct {
	SSL      bool     `yaml:"ssl"`
	Host     string   `yaml:"host"`
	Port     int      `yaml:"port"`
	Pass     string   `yaml:"pass"`
	Nick     string   `yaml:"nick"`
	Oper     string   `yaml:"oper"`
	AutoJoin []string `yaml:"autojoin"`
	Announce bool     `yaml:"announce"`
}

func (info BotInfo) getIRCConfig() *irc.Config {
	cfg := irc.NewConfig(info.Nick, info.Nick, info.Nick)
	cfg.SSL = info.SSL
	cfg.SSLConfig = &tls.Config{ServerName: info.Host}
	cfg.Server = fmt.Sprintf("%s:%d", info.Host, info.Port)
	cfg.Pass = info.Pass
	cfg.NewNick = func(n string) string { return n + "^" }
	return cfg
}

// handler that just logs the line sent
func logEvent(_ *irc.Conn, line *irc.Line) {
	log.Printf("%s: %v", line.Cmd, line)
}

// parse a command directed at us (w/o stripping whitespace)
func parseCommandRaw(channel, msg string) string {
	if len(channel) == 0 || len(msg) == 0 {
		return ""
	}
	if channel[0] == '#' {
		match := addressedToMeRe.FindStringIndex(msg)
		if match != nil {
			return msg[match[1]:]
		} else if msg[0] == '!' {
			return msg[1:]
		} else if msg[:2] == ";;" {
			return msg[2:]
		}
		return ""
	}
	return msg
}

// parse a command directed at us
func parseCommand(channel, msg string) (string, string) {
	if cmd := strings.TrimSpace(parseCommandRaw(channel, msg)); cmd != "" {
		ix := strings.IndexByte(cmd, ' ')
		if ix == -1 {
			return cmd, ""
		}
		return cmd[:ix], cmd[ix+1:]
	}
	return "", ""
}

func main() {
	flag.Parse()

	if credsPath == nil || *credsPath == "" {
		log.Fatalf("I require a credentials file via -creds")
	}

	dat, err := ioutil.ReadFile(*credsPath)
	if err != nil {
		log.Fatal(err)
	}
	info := BotInfo{Nick: "bort", SSL: true}
	if err := yaml.Unmarshal(dat, &info); err != nil {
		log.Fatal(err)
	}

	gdax := makeGDAXInfo()
	quit := make(chan bool)

	c := irc.Client(info.getIRCConfig())
	for _, event := range []string{irc.ERROR, irc.KICK, irc.NOTICE} {
		c.HandleFunc(event, logEvent)
	}
	c.HandleFunc(irc.CONNECTED, func(conn *irc.Conn, line *irc.Line) {
		for _, channel := range info.AutoJoin {
			conn.Join(channel)
		}
	})
	c.HandleFunc(irc.DISCONNECTED, func(conn *irc.Conn, line *irc.Line) {
		log.Printf("DISCONNECTED: %v", line)
		quit <- true
	})
	c.HandleFunc(irc.JOIN, func(conn *irc.Conn, line *irc.Line) {
		if info.Announce {
			channel := line.Args[0]
			conn.Privmsg(channel, "bort ready to serve!")
		}
	})
	c.HandleFunc(irc.PRIVMSG, func(conn *irc.Conn, line *irc.Line) {
		if len(line.Args) != 2 {
			log.Fatalf("got unexepcted len(line.Args) = %d, line.Args = %v, line = %v\n", len(line.Args), line.Args, line)
		}
		channel := line.Args[0]
		cmd, args := parseCommand(channel, line.Args[1])
		if cmd == "" {
			return
		}

		reply := func(response string) {
			conn.Privmsg(channel, response)
		}

		getProduct := func() string {
			if len(args) > 0 {
				product := strings.ToUpper(args)
				if strings.IndexByte(product, '-') == -1 {
					return product + "-USD"
				}
				return product
			}
			return "BTC-USD"
		}

		switch cmd {
		case "price":
			fallthrough
		case "tlast":
			if priceStr, err := gdax.getPrice(getProduct()); err == nil {
				reply(priceStr)
			} else {
				log.Print(err)
			}
		case "products":
			reply(gdax.listProducts())

		case "say":
			if line.Nick == info.Oper {
				ix := strings.IndexByte(args, ' ')
				if ix == -1 {
					log.Printf("say command didn't have a message!")
					return
				}
				conn.Privmsg(args[:ix], args[ix+1:])
			}

		case "all":
			fallthrough
		case "tall":
			reply(gdax.getAllPrices())

		case "vol":
			fallthrough
		case "volume":
			if volStr, err := gdax.getVolume(getProduct()); err == nil {
				reply(volStr)
			} else {
				log.Print(err)
			}
		}
	})
	if err := c.Connect(); err != nil {
		log.Fatalf("Connection error: %v", err)
	}

	// Wait for disconnect
	<-quit
}
