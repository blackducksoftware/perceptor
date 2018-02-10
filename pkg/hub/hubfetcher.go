package hub

import (
	"fmt"

	"github.com/blackducksoftware/hub-client-go/hubapi"
	"github.com/blackducksoftware/hub-client-go/hubclient"
	"github.com/blackducksoftware/perceptor/pkg/common"
	"github.com/prometheus/common/log"
)

type HubFetcher struct {
	client     hubclient.Client
	username   string
	password   string
	baseURL    string
	isLoggedIn bool
}

func (hf *HubFetcher) login() error {
	if hf.isLoggedIn {
		return nil
	}
	// TODO figure out if the client stays logged in indefinitely,
	//   or if maybe it will need to be relogged in at some point.
	// For now, just assume it *will* stay logged in indefinitely.
	err := hf.client.Login(hf.username, hf.password)
	hf.isLoggedIn = (err == nil)
	return err
}

// NewHubFetcher returns a new, logged-in HubFetcher.
// It will instead return an error if either of the following happen:
//  - unable to instantiate an API client
//  - unable to sign in to the Hub
func NewHubFetcher(username string, password string, baseURL string) (*HubFetcher, error) {
	client, err := hubclient.NewWithSession(baseURL, hubclient.HubClientDebugTimings)
	if err != nil {
		return nil, err
	}
	hf := HubFetcher{
		client:     *client,
		username:   username,
		password:   password,
		baseURL:    baseURL,
		isLoggedIn: false}
	err = hf.login()
	if err != nil {
		return nil, err
	}
	return &hf, nil
}

func (hf *HubFetcher) fetchProject(p hubapi.Project) (*Project, error) {
	client := hf.client
	project := Project{Name: p.Name, Source: p.Source, Versions: []Version{}}

	link, err := p.GetProjectVersionsLink()
	if err != nil {
		log.Errorf("error getting project versions link: %v", err)
		return nil, err
	}
	versions, err := client.ListProjectVersions(*link, nil)
	if err != nil {
		log.Errorf("error fetching project version: %v", err)
		return nil, err
	}

	for _, v := range versions.Items {
		version := Version{
			Distribution:    v.Distribution,
			Nickname:        v.Nickname,
			Phase:           v.Phase,
			ReleaseComments: v.ReleaseComments,
			ReleasedOn:      v.ReleasedOn,
			VersionName:     v.VersionName,
			CodeLocations:   []CodeLocation{},
		}

		codeLocationsLink, err := v.GetCodeLocationsLink()
		if err != nil {
			log.Errorf("error getting code locations link: %v", err)
			return nil, err
		}
		codeLocations, err := client.ListCodeLocations(*codeLocationsLink)
		if err != nil {
			log.Errorf("error fetching code locations: %v", err)
			return nil, err
		}
		for _, cl := range codeLocations.Items {
			var codeLocation = CodeLocation{}
			codeLocation.CodeLocationType = cl.Type
			codeLocation.CreatedAt = cl.CreatedAt
			codeLocation.MappedProjectVersion = cl.MappedProjectVersion
			codeLocation.Name = cl.Name
			codeLocation.UpdatedAt = cl.UpdatedAt
			codeLocation.Url = cl.URL
			codeLocation.ScanSummaries = []ScanSummary{}

			scanSummariesLink, err := cl.GetScanSummariesLink()
			if err != nil {
				log.Errorf("error getting scan summaries link: %v", err)
				return nil, err
			}
			scanSummaries, err := client.ListScanSummaries(*scanSummariesLink)
			if err != nil {
				log.Errorf("error fetching scan summaries: %v", err)
				return nil, err
			}
			for _, scanSumy := range scanSummaries.Items {
				var scanSummary = ScanSummary{}
				scanSummary.CreatedAt = scanSumy.CreatedAt
				scanSummary.Status = scanSumy.Status
				scanSummary.UpdatedAt = scanSumy.UpdatedAt
				codeLocation.ScanSummaries = append(codeLocation.ScanSummaries, scanSummary)
			}

			version.CodeLocations = append(version.CodeLocations, codeLocation)
		}

		var riskProfile = RiskProfile{}
		riskProfileLink, err := v.GetProjectVersionRiskProfileLink()
		if err != nil {
			log.Errorf("error getting risk profile link: %v", err)
			return nil, err
		}
		rp, err := client.GetProjectVersionRiskProfile(*riskProfileLink)
		if err != nil {
			log.Errorf("error fetching project version risk profile: %v", err)
			return nil, err
		}
		riskProfile.BomLastUpdatedAt = rp.BomLastUpdatedAt
		riskProfile.Categories = rp.Categories
		version.RiskProfile = riskProfile

		policyStatusLink, err := v.GetProjectVersionPolicyStatusLink()
		if err != nil {
			log.Errorf("error getting policy status link: %v", err)
			return nil, err
		}
		ps, err := client.GetProjectVersionPolicyStatus(*policyStatusLink)
		if err != nil {
			log.Errorf("error fetching project version policy status: %v", err)
			return nil, err
		}
		statusCounts := make(map[string]int)
		for _, item := range ps.ComponentVersionStatusCounts {
			statusCounts[item.Name] = item.Value
		}
		version.PolicyStatus = PolicyStatus{
			OverallStatus:                ps.OverallStatus,
			ComponentVersionStatusCounts: statusCounts,
			UpdatedAt:                    ps.UpdatedAt,
		}

		project.Versions = append(project.Versions, version)
	}

	return &project, nil
}

// FetchProjectByName searches for a project with the matching name,
//   returning a populated Project model
func (hf *HubFetcher) FetchProjectByName(projectName string) (*Project, error) {
	queryString := fmt.Sprintf("name:%s", projectName)
	projectList, err := hf.client.ListProjects(&hubapi.GetListOptions{Q: &queryString})
	if err != nil {
		log.Errorf("error fetching project list: %v", err)
		return nil, err
	}
	for _, p := range projectList.Items {
		if p.Name == projectName {
			return hf.fetchProject(p)
		}
	}
	return nil, nil
}

func (hf *HubFetcher) fetchImageScanUsingProject(project hubapi.Project, image common.Image) (*ImageScan, error) {
	client := hf.client

	link, err := project.GetProjectVersionsLink()
	if err != nil {
		log.Errorf("error getting project versions link: %v", err)
		return nil, err
	}
	q := fmt.Sprintf("versionName:%s", image.HubVersionName())
	options := hubapi.GetListOptions{Q: &q}
	versionList, err := client.ListProjectVersions(*link, &options)
	if err != nil {
		log.Errorf("error fetching project versions: %v", err)
		return nil, err
	}

	versions := []hubapi.ProjectVersion{}
	for _, v := range versionList.Items {
		if v.VersionName == image.Sha {
			versions = append(versions, v)
		}
	}

	switch len(versions) {
	case 0:
		return nil, nil
	case 1:
		break // good to go, continue
	default:
		return nil, fmt.Errorf("expected to find one project version of name %s, found %d", image.ShaName(), len(versions))
	}

	version := versions[0]

	riskProfileLink, err := version.GetProjectVersionRiskProfileLink()
	if err != nil {
		log.Errorf("error getting risk profile link: %v", err)
		return nil, err
	}
	riskProfile, err := client.GetProjectVersionRiskProfile(*riskProfileLink)
	if err != nil {
		log.Errorf("error fetching project version risk profile: %v", err)
		return nil, err
	}

	policyStatusLink, err := version.GetProjectVersionPolicyStatusLink()
	if err != nil {
		log.Errorf("error getting policy status link: %v", err)
		return nil, err
	}
	policyStatus, err := client.GetProjectVersionPolicyStatus(*policyStatusLink)
	if err != nil {
		log.Errorf("error fetching project version policy status: %v", err)
		return nil, err
	}
	policyStatusComponentVersionStatusCounts := make(map[string]int)
	for _, item := range policyStatus.ComponentVersionStatusCounts {
		policyStatusComponentVersionStatusCounts[item.Name] = item.Value
	}

	codeLocationsLink, err := version.GetCodeLocationsLink()
	if err != nil {
		log.Errorf("error getting code locations link: %v", err)
		return nil, err
	}
	codeLocationsList, err := client.ListCodeLocations(*codeLocationsLink)
	if err != nil {
		log.Errorf("error fetching code locations: %v", err)
		return nil, err
	}

	codeLocations := []hubapi.CodeLocation{}
	for _, cl := range codeLocationsList.Items {
		if cl.Name == image.HubScanName() {
			codeLocations = append(codeLocations, cl)
		}
	}

	switch len(codeLocations) {
	case 0:
		return nil, nil
	case 1:
		break // good to go, continue
	default:
		return nil, fmt.Errorf("expected to find one code location of name %s, found %d", image.HubScanName(), len(codeLocations))
	}

	codeLocation := codeLocations[0]

	scanSummariesLink, err := codeLocation.GetScanSummariesLink()
	if err != nil {
		log.Errorf("error getting scan summaries link: %v", err)
		return nil, err
	}
	scanSummariesList, err := client.ListScanSummaries(*scanSummariesLink)
	if err != nil {
		log.Errorf("error fetching scan summaries: %v", err)
		return nil, err
	}

	scanSummaries := []hubapi.ScanSummary{}
	for _, scanSummary := range scanSummariesList.Items {
		if isScanSummaryStatusDone(scanSummary.Status) {
			scanSummaries = append(scanSummaries, scanSummary)
		}
	}

	switch len(scanSummaries) {
	case 0:
		return nil, nil
	case 1:
		break // good to go, continue
	default:
		return nil, fmt.Errorf("expected to find one scan summary for code location %s, found %d", image.HubScanName(), len(scanSummariesList.Items))
	}

	scanSummary := scanSummaries[0]

	scan := ImageScan{
		RiskProfile: RiskProfile{
			BomLastUpdatedAt: riskProfile.BomLastUpdatedAt,
			Categories:       riskProfile.Categories,
		},
		PolicyStatus: PolicyStatus{
			ComponentVersionStatusCounts: policyStatusComponentVersionStatusCounts,
			OverallStatus:                policyStatus.OverallStatus,
			UpdatedAt:                    policyStatus.UpdatedAt,
		},
		ScanSummary: ScanSummary{
			CreatedAt: scanSummary.CreatedAt,
			Status:    scanSummary.Status,
			UpdatedAt: scanSummary.UpdatedAt,
		},
		CodeLocationCreatedAt: codeLocation.CreatedAt,
		//CodeLocationMappedProjectVersion string
		CodeLocationName:      codeLocation.Name,
		CodeLocationType:      codeLocation.Type,
		CodeLocationURL:       codeLocation.URL,
		CodeLocationUpdatedAt: codeLocation.UpdatedAt,
	}

	return &scan, nil
}

// FetchScanFromImage returns an ImageScan only if:
// - it can find a project with the matching name, with
// - a project version with the matching name, with
// - one code location, with
// - one scan summary, with
// - a completed status
func (hf *HubFetcher) FetchScanFromImage(image common.Image) (*ImageScan, error) {
	queryString := fmt.Sprintf("name:%s", image.HubProjectName())
	projectList, err := hf.client.ListProjects(&hubapi.GetListOptions{Q: &queryString})
	if err != nil {
		log.Errorf("error fetching project list: %v", err)
		return nil, err
	}
	projects := []hubapi.Project{}
	for _, proj := range projectList.Items {
		if proj.Name == image.HubProjectName() {
			projects = append(projects, proj)
		}
	}
	if len(projects) == 0 {
		return nil, nil
	}
	if len(projects) > 1 {
		return nil, fmt.Errorf("expected 1 project of name %s, found %d", image.HubProjectName(), len(projects))
	}
	project := projects[0]
	return hf.fetchImageScanUsingProject(project, image)
}
