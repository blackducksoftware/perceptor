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

package federation

import (
	"bytes"
	"fmt"
	"runtime"
	"runtime/pprof"
)

// See: https://play.golang.org/p/0hVB0_LMdm
func dumpStack() (string, string) {
	fmt.Printf("runtime.Stack\n\n\n")
	buf := make([]byte, 1<<16)
	runtime.Stack(buf, true)
	//	fmt.Printf("stack:\n\n%s\n", buf)
	//	fmt.Printf("\n\n\nusing pprof:")
	pprofBuffer := new(bytes.Buffer)
	pprof.Lookup("goroutine").WriteTo(pprofBuffer, 1)
	return string(bytes.Trim(buf, "\x00")), pprofBuffer.String()
}
