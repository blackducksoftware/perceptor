/*
Copyright (C) 2018 Black Duck Software, Inc.

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

package clustermanager

/*
newLabels := make(map[string]string)
newLabels["com.blackducksoftware.image.policy-violations"] = policy // stringed int of number of policy violations (or "None" in place of "0")
newLabels["com.blackducksoftware.image.has-policy-violations"] = hasPolicyViolations // stringed bool
newLabels["com.blackducksoftware.image.vulnerabilities"] = vulns // stringed int (or "None" in place of "0")
newLabels["com.blackducksoftware.image.has-vulnerabilities"] = hasVulns // stringed bool
inputImageInfo.Labels = mapMerge(inputImageInfo.Labels, newLabels)

newAnnotations := make(map[string]string)
newAnnotations[ScannerVersionLabel] = a.ScannerVersion
newAnnotations[ScannerHubServerLabel] = a.HubServer
newAnnotations[ScannerProjectVersionUrl] = projectVersionUrl
if len(scanId) > 0 {
  newAnnotations[ScannerScanId] = scanId
}
inputImageInfo.Annotations = mapMerge(inputImageInfo.Annotations, newAnnotations)

vulnAnnotations := a.CreateBlackduckVulnerabilityAnnotation(hasVulns == "true", projectVersionUIUrl, vulns) // ? a stringified json dict
policyAnnotations := a.CreateBlackduckPolicyAnnotation(hasPolicyViolations == "true", projectVersionUIUrl, policy) // ? a stringified json dict

inputImageInfo.Annotations["quality.images.openshift.io/vulnerability.blackduck"] = vulnAnnotations.AsString()
inputImageInfo.Annotations["quality.images.openshift.io/policy.blackduck"] = policyAnnotations.AsString()
*/

/*
// CreateOpenshiftAnnotations takes the primitive information from UpdateAnnotation and translates it to openshift.
func (a *Annotator) CreateBlackduckVulnerabilityAnnotation(hasVulns bool, humanReadableURL string, vulnCount string) *BlackduckAnnotation {
	return &BlackduckAnnotation{
		"blackducksoftware",
		"Vulnerability Info",
		time.Now(),
		humanReadableURL,
		!hasVulns, // no vunls -> compliant.
		[]map[string]string{
			{
				"label":         "high",
				"score":         fmt.Sprintf("%s", vulnCount),
				"severityIndex": fmt.Sprintf("%v", 1),
			},
		},
	}
}
func (a *Annotator) CreateBlackduckPolicyAnnotation(hasPolicyViolations bool, humanReadableURL string, policyCount string) *BlackduckAnnotation {
	return &BlackduckAnnotation{
		"blackducksoftware",
		"Policy Info",
		time.Now(),
		humanReadableURL,
		!hasPolicyViolations, // no violations -> compliant
		[]map[string]string{
			{
				"label":         "important",
				"score":         fmt.Sprintf("%s", policyCount),
				"severityIndex": fmt.Sprintf("%v", 1),
			},
		},
*/

/*
oc describe image -l "com.blackducksoftware.image.has-policy-violations=true"

Name: sha256:f2bbbf44fa502938c87702035e86e9193738398282da85ac3d763122069733de
Namespace: <none>
Created: 2 weeks ago
Labels: com.blackducksoftware.image.has-policy-violations=true
com.blackducksoftware.image.has-vulnerabilities=true
com.blackducksoftware.image.policy-violations=6
com.blackducksoftware.image.vulnerabilities=13
Annotations: blackducksoftware.com/attestation-hub-server=ec2-52-15-236-103.us-east-2.compute.amazonaws.com
blackducksoftware.com/hub-scanner-version=3.6.2
blackducksoftware.com/project-endpoint=http://ec2-52-15-236-103.us-east-2.compute.amazonaws.com/api/...
blackducksoftware.com/scan-id=8cca941a-6c9a-40da-bf7d-9357a3790078
openshift.io/image.managed=true
Docker Image: 172.30.180.219:5000/sandbox/eap-app-mogo-001@sha256:f2bbbf44fa502938c87702035e86e9193738398282da85ac3d763122069733de
*/
