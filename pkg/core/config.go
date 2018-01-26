package core

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// PerceptorConfig contains all configuration for Perceptor
type PerceptorConfig struct {
	HubHost          string
	HubUser         string
	HubUserPassword string
}

// GetPerceptorConfig returns a configuration object to configure Perceptor
func GetPerceptorConfig() (*PerceptorConfig, error) {
	var pcfg *PerceptorConfig

	viper.SetConfigName("perceptor_conf")
	viper.AddConfigPath("/etc/perceptor")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	err = viper.Unmarshal(&pcfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}
	return pcfg, nil
}

// StartWatch will start watching the Perceptor configuration file and
// call the passed handler function when the configuration file has changed
func (p *PerceptorConfig) StartWatch(handler func(fsnotify.Event)) {
	viper.WatchConfig()
	viper.OnConfigChange(handler)
}
