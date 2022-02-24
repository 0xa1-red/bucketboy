package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/alfreddobradi/twitchbot/internal/twitch"
	"github.com/davecgh/go-spew/spew"
)

var (
	username string = ""
	token    string = ""
	channel  string = ""
)

func main() {
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
	}
	client, err := twitch.New(clientConfig)
	if err != nil {
		log.Printf("Error creating client: %v", err)
	}

	shutdown := make(chan struct{})
	go func() {
		for {
			select {
			case err := <-client.Errors:
				// if errors.Is(err, )
				log.Println("ERROR: " + err.Error())
				spew.Dump(err)
			case <-interrupt:
				client.Close(shutdown)
			}
		}
	}()

	client.Connect(shutdown)

	if err := client.Send(fmt.Sprintf("JOIN #%s", channel)); err != nil {
		panic(err)
	}

	<-shutdown
}
