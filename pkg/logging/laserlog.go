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

package logging

// Lazerbeak is a logging library that intercepts everyhing
// and reports it upstream.

import (
	"github.com/prometheus/client_golang/prometheus"
	logrus "github.com/sirupsen/logrus"
)

type MetricsHook struct {
	vec *prometheus.CounterVec
}

func (hook *MetricsHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.DebugLevel,
		logrus.ErrorLevel,
		logrus.FatalLevel,
		logrus.InfoLevel,
		logrus.PanicLevel,
		logrus.WarnLevel,
	}
}

func (hook *MetricsHook) Fire(entry *logrus.Entry) error {
	hook.vec.WithLabelValues(entry.Level.String()).Inc()
	return nil
}

func init() {
	logrus.Infof("INITIALIZING LOGRUS WITH METRICS HOOK [metrics subsystem = %v]", "laserbeak")
	// formatter := &logrus.TextFormatter{
	// 	FullTimestamp:   false,
	// 	TimestampFormat: "15:04",
	// }
	// logrus.SetFormatter(formatter)

	cv := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "perceptor",
			Subsystem: "laserbeak",
			Name:      "log",
			Help:      "counts logrus calls by warning level",
		},
		[]string{"log_type"})

	prometheus.MustRegister(cv)

	logrus.AddHook(&MetricsHook{cv})
}
