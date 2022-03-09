package twitch

import (
	"github.com/alfreddobradi/twitchbot/internal/logging"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func Logger() *logrus.Logger {
	if logger == nil {
		logger = logging.New()
	}

	return logger
}
