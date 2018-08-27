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

	"github.com/spf13/viper"
)

// ConfigManager handles:
//   - getting initial config
//   - reporting ongoing changes to config
type ConfigManager struct {
	ConfigPath      string
	IgnoreConfigMap bool
}

// NewConfigManager ...
func NewConfigManager(configPath string, ignoreConfigMap bool) *ConfigManager {
	return &ConfigManager{
		ConfigPath:      configPath,
		IgnoreConfigMap: ignoreConfigMap,
	}
}

// GetConfig returns a configuration object to configure Perceptor
func (cm *ConfigManager) GetConfig() (*Config, error) {
	config := new(Config)

	// Set the defaults
	for key, value := range config.GetDefaults() {
		viper.SetDefault(key, value)
	}

	// Look for environment variables (prefixed with "PCP_")
	viper.SetEnvPrefix("pcp") // uppercased automatically
	viper.AutomaticEnv()

	viper.SetConfigFile(cm.ConfigPath)

	// If IgnoreConfigMap set then all config items should be set via environment or defaults
	if cm.IgnoreConfigMap == false {
		err := viper.ReadInConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %v", err)
		}
	}

	// This Unmarshal will take values from in the following order of precedence:
	// environment, configMap, defaults
	err := viper.Unmarshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return config, nil
}

//
// // StartWatch will call `continuation` whenever the config file changes
// func (cm *ConfigManager) StartWatch(continuation func(*model.Config, error)) {
// 	viper.WatchConfig()
// 	viper.OnConfigChange(func(event fsnotify.Event) {
// 		log.Infof("config change detected: %+v", event)
// 		continuation(cm.GetConfig())
// 	})
// 	viper.WatchConfig()
// }
