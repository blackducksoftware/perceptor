/*
Copyright (C) 2018 Synopsys, Inc.

Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements. See the NOTICE file
distributed with this work for additional information
regarding copyright ownership. The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied. See the License for the
specific language governing permissions and limitations
under the License.
*/

package core

import (
	"fmt"
	"strings"
	"time"

	fsnotify "github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// ConfigManager handles:
//   - getting initial config
//   - reporting ongoing changes to config
type ConfigManager struct {
	ConfigPath string
}

// NewConfigManager ...
func NewConfigManager(configPath string) *ConfigManager {
	cm := &ConfigManager{
		ConfigPath: configPath,
	}
	cm.StartWatch(func(conf *Config, err error) {
		log.Infof("watch config: %+v, %+v", conf, err)
	})
	return cm
}

// GetConfig returns a configuration object to configure Perceptor
func (cm *ConfigManager) GetConfig() (*Config, error) {
	var config *Config

	if cm.ConfigPath != "" {
		viper.SetConfigFile(cm.ConfigPath)
	} else {
		viper.SetEnvPrefix("PCP")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

		viper.BindEnv("Timings_CheckForStalledScansPauseHours")
		viper.BindEnv("Timings_ModelMetricsPauseSeconds")
		viper.BindEnv("Timings_PruneOrphanedImagesPauseMinutes")
		viper.BindEnv("Timings_StalledScanClientTimeoutHours")
		viper.BindEnv("Timings_UnknownImagePauseMilliseconds")

		viper.BindEnv("Hub_User")
		viper.BindEnv("Hub_TotalScanLimit")
		viper.BindEnv("Hub_Port")
		viper.BindEnv("Hub_PasswordEnvVar")
		viper.BindEnv("Hub_ConcurrentScanLimit")
		viper.BindEnv("Hub_ClientTimeoutMilliseconds")

		viper.BindEnv("LogLevel")
		viper.BindEnv("Port")
		viper.BindEnv("UseMockMode")

		viper.AutomaticEnv()
	}

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

// StartWatch will call `continuation` whenever the config file changes
func (cm *ConfigManager) StartWatch(continuation func(*Config, error)) {
	viper.WatchConfig()
	viper.OnConfigChange(func(event fsnotify.Event) {
		log.Infof("config change detected: %+v", event)
		continuation(cm.GetConfig())
	})
	go func() {
		for {
			time.Sleep(15 * time.Second)
			conf, err := cm.GetConfig()
			fmt.Printf("pulled config: %+v, %+v\n", conf, err)
		}
	}()
}
