package main

import (
	"strings"

	log "github.com/Sirupsen/logrus"
)

func validateAndSetLogLevel(str string) (valid bool) {
	str = strings.ToLower(str)
	logLevels := []string{"debug", "info", "warning", "error", "fatal", "panic"}
	for _, level := range logLevels {
		if str == level {
			valid = true
		}
	}

	if valid {
		setLogLevel(str)
	}

	return
}

func setLogLevel(str string) {
	switch {
	case str == "debug":
		log.SetLevel(log.DebugLevel)
		break
	case str == "info":
		log.SetLevel(log.InfoLevel)
		break
	case str == "warn":
		log.SetLevel(log.WarnLevel)
		break
	case str == "error":
		log.SetLevel(log.ErrorLevel)
		break
	case str == "fatal":
		log.SetLevel(log.FatalLevel)
		break
	case str == "panic":
		log.SetLevel(log.PanicLevel)
		break
	}
}
