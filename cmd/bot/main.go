package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/alfreddobradi/twitchbot/internal/logging"
	"github.com/alfreddobradi/twitchbot/internal/twitch"
)

var (
	username string = ""
	token    string = ""
	channel  string = ""
)

func main() {
	log := logging.New()

	flag.StringVar(&username, "username", "justinfan123123", "Username to connect with.")
	flag.StringVar(&token, "token", "oauth:59301", "OAuth token to connect with.")
	flag.StringVar(&channel, "channel", "", "Channel to join on connect")
	flag.Parse()

	if envUsername := os.Getenv("TWITCH_USERNAME"); envUsername != "" {
		username = envUsername
	}
	if envToken := os.Getenv("TWITCH_TOKEN"); envToken != "" {
		token = envToken
	}
	if envChannel := os.Getenv("TWITCH_CHANNEL"); envChannel != "" {
		channel = envChannel
	}

	if channel == "" {
		fmt.Println("Can't connect without a channel to join")
		os.Exit(1)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	clientConfig := twitch.ClientConfig{
		Username:     username,
		Token:        token,
		Capabilities: []twitch.TwitchCap{twitch.Membership, twitch.Commands, twitch.Tags},
		Channels:     []string{channel},
	}
	client, err := twitch.New(clientConfig)
	if err != nil {
		log.WithError(err).Fatal("Couldn't create client")
	}

	shutdown := make(chan struct{})
	go func() {
		for {
			select {
			case err := <-client.Errors:
				log.WithError(err).Error("Client error")
			case <-interrupt:
				client.Close(shutdown)
			}
		}
	}()

	client.Connect(shutdown)

	<-shutdown
	log.Info("Shutting down")
}
