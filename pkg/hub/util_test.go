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

package hub

import (
	"testing"
	"time"
)

// TestMinDuration .....
func TestMinDuration(t *testing.T) {
	cases := []struct {
		left     time.Duration
		right    time.Duration
		expected time.Duration
	}{
		{time.Second, time.Minute, time.Second},
		{time.Minute, time.Second, time.Second},
		{time.Second, time.Second, time.Second},
		{time.Duration(float64(2)) * time.Second, 4 * time.Second, 2 * time.Second},
	}
	for _, c := range cases {
		actual := MinDuration(c.left, c.right)
		if actual != c.expected {
			t.Errorf("for %s and %s, expected %s but got %s", c.left, c.right, c.expected, actual)
		}
	}
}
