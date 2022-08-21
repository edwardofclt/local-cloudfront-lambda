package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/edwardofclt/cloudfront-emulator/internal/cloudfront"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var viperConfig *viper.Viper

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		logrus.WithError(err).Fatal("failed to find working directory")
	}

	if len(os.Args) >= 2 {
		cwd = os.Args[1]
	}

	p := &cloudfront.CloudfrontConfig{}

	viperConfig = viper.New()
	viperConfig.AddConfigPath(cwd)
	viperConfig.SetConfigType("yml")
	viperConfig.SetConfigName("config")
	viperConfig.WatchConfig()
	viperConfig.ReadInConfig()

	if err := viperConfig.UnmarshalKey("config", p); err != nil {
		logrus.WithError(err).Fatal("failed to unmarshal config")
	}

	cf := cloudfront.New(p)

	viperConfig.OnConfigChange(func(in fsnotify.Event) {
		logrus.Info("Configuration Updated")
		if err := viperConfig.UnmarshalKey("config", p); err != nil {
			logrus.WithError(err).Fatal("failed to refresh config")
		}
		if err := viperConfig.UnmarshalKey("config", p); err != nil {
			logrus.WithError(err).Fatal("failed to unmarshal config")
		}

		cf.Refresh(p)
	})

	cancelChan := make(chan os.Signal, 1)
	signal.Notify(cancelChan, syscall.SIGTERM, syscall.SIGINT)
	<-cancelChan
}
