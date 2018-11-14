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

	"github.com/blackducksoftware/perceptor/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// ConfigManager handles:
//   - getting initial config
//   - reporting ongoing changes to config
type ConfigManager struct {
	ConfigPath      string
	stop            <-chan struct{}
	didReadConfig   chan *Config
	readConfigPause time.Duration
	readConfigTimer *util.Timer
}

// NewConfigManager ...
func NewConfigManager(configPath string, stop <-chan struct{}) *ConfigManager {
	cm := &ConfigManager{
		ConfigPath:      configPath,
		stop:            stop,
		didReadConfig:   make(chan *Config),
		readConfigPause: 15 * time.Second,
	}
	cm.startReadConfigTimer()
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

		viper.BindEnv("Perceptor_Port")
		viper.BindEnv("Perceptor_UseMockMode")
		viper.BindEnv("Perceptor_Timings_CheckForStalledScansPauseHours")
		viper.BindEnv("Perceptor_Timings_ModelMetricsPauseSeconds")
		viper.BindEnv("Perceptor_Timings_StalledScanClientTimeoutHours")
		viper.BindEnv("Perceptor_Timings_UnknownImagePauseMilliseconds")
		viper.BindEnv("Perceptor_Timings_ClientTimeoutMilliseconds")

		viper.BindEnv("Hub_Hosts")
		viper.BindEnv("Hub_User")
		viper.BindEnv("Hub_TotalScanLimit")
		viper.BindEnv("Hub_Port")
		viper.BindEnv("Hub_PasswordEnvVar")
		viper.BindEnv("Hub_ConcurrentScanLimit")

		viper.BindEnv("LogLevel")

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

func (cm *ConfigManager) startReadConfigTimer() {
	cm.readConfigTimer = util.NewRunningTimer("configManager-readConfig", cm.readConfigPause, cm.stop, false, func() {
		config, err := cm.GetConfig()
		if err != nil {
			log.Errorf("unable to read config: %s", err.Error())
			return
		}
		select {
		case <-cm.stop:
			return
		case cm.didReadConfig <- config:
		}
	})
}

// DidReadConfig ...
func (cm *ConfigManager) DidReadConfig() <-chan *Config {
	return cm.didReadConfig
}
