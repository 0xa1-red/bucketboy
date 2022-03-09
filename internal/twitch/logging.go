package twitch

import (
	"github.com/alfreddobradi/twitchbot/internal/logging"
	"github.com/sirupsen/logrus"
)

var log *logrus.Entry

func init() {
	logger := logging.New()
	log = logger.WithField("module", "twitch")
}
