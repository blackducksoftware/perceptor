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
	"reflect"
	"time"

	a "github.com/blackducksoftware/perceptor/pkg/core/actions"
	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	log "github.com/sirupsen/logrus"
)

type reducer struct{}

// logic

func newReducer(model *m.Model, actions <-chan a.Action) *reducer {
	stop := time.Now()
	go func() {
		for {
			select {
			case nextAction := <-actions:
				if log.GetLevel() == log.DebugLevel {
					log.Debugf("processing reducer action of type %s", reflect.TypeOf(nextAction))
				}

				// metrics: how many messages are waiting?
				recordNumberOfMessagesInQueue(len(actions))

				// metrics: log message type
				recordMessageType(fmt.Sprintf("%s", reflect.TypeOf(nextAction)))

				// metrics: how long idling since the last action finished processing?
				start := time.Now()
				recordReducerActivity(false, start.Sub(stop))

				// actually do the work
				nextAction.Apply(model)

				// metrics: how long did the work take?
				stop = time.Now()
				recordReducerActivity(true, stop.Sub(start))
			}
		}
	}()
	return &reducer{}
}
