package main

import (
	"fmt"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
)

func main() {
	configPath := os.Args[1]
	handleConfig(configPath)
	viper.OnConfigChange(func(in fsnotify.Event) {
		handleConfig(configPath)
	})
	viper.WatchConfig()
	log.Info("Viper examples are running!")

	select {}
}

func handleConfig(configPath string) {
	config, err := getConfig(configPath)
	if err != nil {
		log.Errorf("unable to get new config: %s", err.Error())
	} else {
		log.Infof("new config %+v", config)
	}
}

type Config struct {
	A int
	B string
	C bool
	D []string
}

func getConfig(configPath string) (*Config, error) {
	var config *Config

	viper.SetConfigFile(configPath)

	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return config, nil
}
