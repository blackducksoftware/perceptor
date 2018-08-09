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

package util

import (
	"bytes"
	"runtime"
	"runtime/pprof"
)

// See: https://play.golang.org/p/0hVB0_LMdm

// DumpRuntimeStack uses runtime to inspect goroutines
func DumpRuntimeStack() string {
	buf := make([]byte, 1<<16)
	runtime.Stack(buf, true)
	return string(bytes.Trim(buf, "\x00"))
}

// DumpPProfStack uses pprof to inspect goroutines
func DumpPProfStack() (string, int) {
	pprofBuffer := new(bytes.Buffer)
	profile := pprof.Lookup("goroutine")
	profile.WriteTo(pprofBuffer, 1)
	return pprofBuffer.String(), profile.Count()
}

// DumpHeap uses pprof to inspect the heap
func DumpHeap() (string, int) {
	heapBuffer := new(bytes.Buffer)
	profile := pprof.Lookup("heap")
	profile.WriteTo(heapBuffer, 1)
	return heapBuffer.String(), profile.Count()
}
