package main

import (
	"fmt"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
)

func main() {
	var configPath string
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}
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

// Config ...
type Config struct {
	A int
	B string
	C bool
	D []string
}

func getConfig(configPath string) (*Config, error) {
	var config Config
	log.Infof("config path: %+v", configPath)

	if configPath != "" {
		viper.SetConfigFile(configPath)
		err := viper.ReadInConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %v", err)
		}
	} else {
		log.Infof("about to read env")
		viper.SetEnvPrefix("VPE")
		viper.BindEnv("A")
		viper.BindEnv("B")
		viper.BindEnv("C")
		viper.AutomaticEnv()
	}

	err := viper.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return &config, nil
}
