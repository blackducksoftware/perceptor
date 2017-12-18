package scanner

import (
	"encoding/json"
	"fmt"

	hubapi "bitbucket.org/bdsengineering/go-hub-client/hubapi"
	hubclient "bitbucket.org/bdsengineering/go-hub-client/hubclient"
	log "github.com/prometheus/common/log"
)

type Project struct {
	Name     string
	Source   string
	Versions []Version
}

type Version struct {
	CodeLocations   []CodeLocation
	RiskProfile     RiskProfile
	Distribution    string
	Nickname        string
	VersionName     string
	ReleasedOn      string
	ReleaseComments string
	Phase           string
}

type CodeLocation struct {
	ScanSummaries        []ScanSummary
	CreatedAt            string
	MappedProjectVersion string
	Name                 string
	CodeLocationType     string
	Url                  string
	UpdatedAt            string
}

type RiskProfile struct {
	Categories       map[string]map[string]int
	BomLastUpdatedAt string
}

type ScanSummary struct {
	CreatedAt string
	Status    string
	UpdatedAt string
}

type ProjectFetcher struct {
	client     hubclient.Client
	username   string
	password   string
	baseURL    string
	isLoggedIn bool
}

func (pf *ProjectFetcher) login() error {
	if pf.isLoggedIn {
		return nil
	}
	// TODO figure out if the client stays logged in indefinitely,
	//   or if maybe it will need to be relogged in at some point.
	// For now, just assume it *will* stay logged in indefinitely.
	err := pf.client.Login(pf.username, pf.password)
	pf.isLoggedIn = (err == nil)
	return err
}

// NewProjectFetcher returns a new, logged-in ProjectFetcher.
// It will instead return an error if either of the following happen:
//  - unable to instantiate a Hub API client
//  - unable to sign in to the Hub
func NewProjectFetcher(username string, password string, baseURL string) (*ProjectFetcher, error) {
	client, err := hubclient.NewWithSession(baseURL, hubclient.HubClientDebugTimings)
	if err != nil {
		return nil, err
	}
	pf := ProjectFetcher{
		client:     *client,
		username:   username,
		password:   password,
		baseURL:    baseURL,
		isLoggedIn: false}
	err = pf.login()
	if err != nil {
		return nil, err
	}
	return &pf, nil
}

func (pf *ProjectFetcher) fetchProject(p hubapi.Project) Project {
	client := pf.client
	project := Project{Name: p.Name, Source: p.Source, Versions: []Version{}}

	link, err := p.GetProjectVersionsLink()
	if err != nil {
		panic(fmt.Sprintf("error getting project versions link: %v", err))
	}
	versions, err := client.ListProjectVersions(*link)
	if err != nil {
		panic(fmt.Sprintf("error fetching project version: %v", err))
	}

	for _, v := range versions.Items {
		var version Version
		version.Distribution = v.Distribution
		version.Nickname = v.Nickname
		version.Phase = v.Phase
		version.ReleaseComments = v.ReleaseComments
		version.ReleasedOn = v.ReleasedOn
		version.VersionName = v.VersionName
		version.CodeLocations = []CodeLocation{}

		codeLocationsLink, err := v.GetCodeLocationsLink()
		if err != nil {
			panic(fmt.Sprintf("error getting code locations link: %v", err))
		}
		codeLocations, err := client.ListCodeLocations(*codeLocationsLink)
		if err != nil {
			panic(fmt.Sprintf("error fetching code locations: %v", err))
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
				panic(fmt.Sprintf("error getting scan summaries link: %v", err))
			}
			scanSummaries, err := client.ListScanSummaries(*scanSummariesLink)
			if err != nil {
				panic(fmt.Sprintf("error fetching scan summaries: %v", err))
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
			panic(fmt.Sprintf("error getting risk profile link: %v", err))
		}
		rp, err := client.GetProjectVersionRiskProfile(*riskProfileLink)
		if err != nil {
			panic(fmt.Sprintf("error fetching project version risk profile: %v", err))
		}
		riskProfile.BomLastUpdatedAt = rp.BomLastUpdatedAt
		riskProfile.Categories = rp.Categories
		version.RiskProfile = riskProfile

		// TODO can't get PolicyStatus for now
		// v.GetPolicyStatusLink()

		project.Versions = append(project.Versions, version)
	}

	return project
}

// FetchProjectOfName searches for a project with the matching name,
//   returning a populated Project model
func (pf *ProjectFetcher) FetchProjectOfName(projectName string) *Project {
	projs, err := pf.client.ListProjects()
	if err != nil {
		panic(fmt.Sprintf("error fetching project list: %v", err))
	}
	for _, p := range projs.Items {
		if p.Name != projectName {
			// log.Info("skipping project ", p.Name, " as it doesn't match requested name ", projectName)
			continue
		}
		project := pf.fetchProject(p)
		return &project
	}
	return nil
}

// HitHubAPI is an example; don't use it in production
func HitHubAPI() {
	baseURL := "https://localhost"
	username := "sysadmin"
	password := "blackduck"
	pf, err := NewProjectFetcher(username, password, baseURL)
	if err != nil {
		panic("unable to instantiate ProjectFetcher: " + err.Error())
	}
	project := pf.FetchProjectOfName("openshift/origin-docker-registry")
	bytes, _ := json.Marshal(project)
	log.Infof("fetched project: %v \n\nwith json: %v", project, string(bytes[:]))
	log.Infof("bytes: %d", len(bytes))
}

func exampleHubAPI() {
	baseURL := "https://localhost"
	username := "sysadmin"
	password := "blackduck"
	client, err := hubclient.NewWithSession(baseURL, hubclient.HubClientDebugTimings)
	if err != nil {
		log.Fatalf("unable to create hub client %v", err)
		panic("oops, unable to create hub client " + err.Error())
	}
	err = client.Login(username, password)
	if err == nil {
		log.Info("success logging in!")
		projects, _ := client.ListProjects()
		log.Info("projects: %v", projects)
	} else {
		log.Errorf("unable to log in, %v", err)
	}

	projs, err := client.ListProjects()
	if err != nil {
		panic(fmt.Sprintf("error fetching project list: %v", err))
	}
	for _, p := range projs.Items {
		log.Info("proj: ", p)
		log.Info("proj href: ", p.Meta.Href)
		link, err := p.GetProjectVersionsLink()
		if err != nil {
			panic(fmt.Sprintf("error getting project versions link: %v", err))
		}
		versions, err := client.ListProjectVersions(*link)
		if err != nil {
			panic(fmt.Sprintf("error fetching project version: %v", err))
		}
		log.Info("project versions for url: ", link.Href, ": ", versions, "\n\n")

		for _, v := range versions.Items {
			log.Info("version: ", v)
			log.Info("version href: ", v.Meta.Href)
			codeLocationsLink, err := v.GetCodeLocationsLink()
			if err != nil {
				panic(fmt.Sprintf("error getting code locations link: %v", err))
			}
			//codeLocations, err := client.GetCodeLocation(*codeLocationsLink)
			codeLocations, err := client.ListCodeLocations(*codeLocationsLink)
			//			client.
			if err != nil {
				panic(fmt.Sprintf("error fetching code locations: %v", err))
			}
			log.Info("code locations: ", codeLocations)
			for _, codeLocation := range codeLocations.Items {
				scanSummariesLink, err := codeLocation.GetScanSummariesLink()
				if err != nil {
					panic(fmt.Sprintf("error getting scan summaries link: %v", err))
				}
				scanSummaries, err := client.ListScanSummaries(*scanSummariesLink)
				if err != nil {
					panic(fmt.Sprintf("error fetching scan summaries: %v", err))
				}
				for _, scanSummary := range scanSummaries.Items {
					log.Info("scan summary: ", scanSummary)
				}
			}

			riskProfileLink, err := v.GetProjectVersionRiskProfileLink()
			if err != nil {
				panic(fmt.Sprintf("error getting risk profile link: %v", err))
			}
			riskProfile, err := client.GetProjectVersionRiskProfile(*riskProfileLink)
			if err != nil {
				panic(fmt.Sprintf("error fetching project version risk profile: %v", err))
			}
			log.Info("project version risk profile: ", riskProfile)

			// TODO can't get PolicyStatus for now
			// v.GetPolicyStatusLink()

			//scanSummaryLink, err := v.
			log.Info("\n\n")
		}
		log.Info("\n\n\n")
	}
	//	log.Info("projs", projs)
}
