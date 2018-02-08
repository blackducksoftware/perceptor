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

type BomComponentList struct {
	TotalCount uint32         `json:"totalCount"`
	Items      []BomComponent `json:"items"`
	Meta       Meta           `json:"_meta"`
}

type BomComponent struct {
	ComponentName          string               `json:"componentName"`
	ComponentVersionName   string               `json:"componentVersionName"`
	Component              string               `json:"component"`
	ComponentVersion       string               `json:"componentVersion"`
	ReleasedOn             string               `json:"releasedOn"`
	ReviewStatus           string               `json:"reviewStatus"`
	PolicyStatus           string               `json:"approvalStatus"`
	Licenses               []ComplexLicense     `json:"licenses"`
	Usages                 []string             `json:"usages"`
	Origins                []BomComponentOrigin `json:"origins"`
	LicenseRiskProfile     BomRiskProfile       `json:"licenseRiskProfile"`
	VersionRiskProfile     BomRiskProfile       `json:"versionRiskProfile"`
	SecurityRiskProfile    BomRiskProfile       `json:"securityRiskProfile"`
	ActivityRiskProfile    BomRiskProfile       `json:"activityRiskProfile"`
	OperationalRiskProfile BomRiskProfile       `json:"operationalRiskProfile"`
	ActivityData           BomActivityData      `json:"activityData"`
	Meta                   Meta                 `json:"_meta"`
}

type BomVulnerableComponentList struct {
	TotalCount uint32                   `json:"totalCount"`
	Items      []BomVulnerableComponent `json:"items"`
	Meta       Meta                     `json:"_meta"`
}

type BomVulnerableComponent struct {
	ComponentName              string                       `json:"componentName"`
	ComponentVersionName       string                       `json:"componentVersionName"`
	ComponentVersion           string                       `json:"componentVersion"`
	ComponentVersionOriginName string                       `json:"componentVersionOriginName"`
	ComponentVersionOrigingID  string                       `json:"componentVersionOriginId"`
	License                    ComplexLicense               `json:"license"`
	Vulnerability              VulnerabilityWithRemediation `json:"vulnerabilityWithRemediation"`
	Meta                       Meta                         `json:"_meta"`
}

type VulnerabilityWithRemediation struct {
	VulnerabilityName          string  `json:"vulnerabilityName"`
	Description                string  `json:"description"`
	VulnerabilityPublishedDate string  `json:"vulnerabilityPublishedDate"`
	VulnerabilityUpdatedDate   string  `json:"vulnerabilityUpdatedDate"`
	BaseScore                  float32 `json:"baseScore"`
	ExploitabilitySubscore     float32 `json:"exploitabilitySubscore"`
	ImpactSubscore             float32 `json:"impactSubscore"`
	Source                     string  `json:"source"`
	Severity                   string  `json:"severity"`
	RemediationStatus          string  `json:"remediationStatus"`
	RemediationCreatedAt       string  `json:"remediationCreatedAt"`
	RemediationUpdatedAt       string  `json:"remediationUpdatedAt"`
}

type BomRiskProfile struct {
	Counts []BomRiskProfileItem `json:"counts"`
}

type BomRiskProfileItem struct {
	CountType string `json:"countType"`
	Count     int    `json:"count"`
}

type BomComponentOrigin struct {
	Name                          string `json:"name"`
	ExternalNamespace             string `json:"externalNamespace"`
	ExternalID                    string `json:"externalId"`
	ExternalNamespaceDistribution bool   `json:"externalNamespaceDistribution"`
	Meta                          Meta   `json:"_meta"`
}

type BomActivityData struct {
	ContributorCount int    `json:"contributorCount12Month"`
	CommitCount      int    `json:"commitCount12Month"`
	LastCommitDate   string `json:"lastCommitDate"`
	Trend            string `json:"trending"`
}
