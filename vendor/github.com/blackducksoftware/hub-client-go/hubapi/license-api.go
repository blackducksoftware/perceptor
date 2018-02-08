// Copyright 2018 Synopsys, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hubapi

type ComplexLicense struct {
	Name           string           `json:"name"`
	Ownership      string           `json:"ownership"`
	CodeSharing    string           `json:"codeSharing"`
	LicenseType    string           `json:"type"`
	LicenseDisplay string           `json:"licenseDisplay"`
	Licenses       []ComplexLicense `json:"licenses"`
	License        string           `json:"license"` // License URL
}

type License struct {
	Name        string `json:"name"`
	Ownership   string `json:"ownership"`
	CodeSharing string `json:"codeSharing"`
	Meta        Meta   `json:"_meta"`
}
