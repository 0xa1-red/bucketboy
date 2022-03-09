package twitch

import (
	"strings"

	"github.com/sirupsen/logrus"
)

func banCommand(message Message) {
	channel := message.Params[0]
	msg := strings.Join(message.Params[1:], " ")
	if !strings.HasPrefix(msg, "!") {
		return
	}
	log.WithFields(logrus.Fields{
		"channel": channel,
		"message": msg,
	}).Info("Message received...")
}
