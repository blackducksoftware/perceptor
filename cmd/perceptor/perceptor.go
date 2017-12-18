package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	clustermanager "bitbucket.org/bdsengineering/perceptor/clustermanager"
	core "bitbucket.org/bdsengineering/perceptor/core"
	scanner "bitbucket.org/bdsengineering/perceptor/scanner"
	"github.com/fsnotify/fsnotify"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	log.Info("start")

	var sc scanner.ScanClientInterface
	var clusterClient clustermanager.Client
	var err error

	// chose a scanner client: real or mock
	// sc = scanner.NewHubScanClient("sysadmin", "blackduck", "localhost")
	sc = scanner.NewMockHub()

	// chose a cluster client: real or mock
	// clusterClient, err := clustermanager.NewKubeClient()
	clusterClient = clustermanager.NewMockClient()

	cache := core.NewVulnerabilityCache()
	if err != nil {
		log.Fatalf("unable to instantiate kubernetes client: %s", err.Error())
		panic(err)
	}
	var lastAdd *clustermanager.Pod
	go func() {
		for {
			select {
			case addPod := <-clusterClient.PodAdd():
				cache.AddPod(addPod.New)
				fmt.Printf("add pod: %v\n", addPod.New.Annotations)
				lastAdd = &addPod.New
			case u := <-clusterClient.PodUpdate():
				fmt.Printf("update pod: %v\n", u.New.Annotations)
			case d := <-clusterClient.PodDelete():
				fmt.Printf("delete pod: %v\n", d)
			}
		}
	}()
	go func() {
		for i := 0; ; i++ {
			select {
			case image := <-cache.ImagesToBeScanned:
				log.Infof("should scan image %s", image)
				// TODO need to think about how to limit concurrent scans to <= 7
				projectName := fmt.Sprintf("my-%s-project-%d", image, i)
				sc.Scan(*scanner.NewScanJob(projectName, image))
			}
		}
	}()

	log.Info("kube client: %v", clusterClient)
	log.Info("finished starting")
	setupHTTPServer()
	// hack to prevent main from returning
	select {}
}

func pollForFinishedScans(hub *scanner.ScanClientInterface, finishedContainerImagesSink chan<- []string) {
	for {
		jobs := (*hub).GetFinishedJobs()
		if len(jobs) > 0 {
			log.Info("finished jobs", jobs)
			// TODO dump jobs into sink
			finishedContainerImagesSink <- []string{}
		}
		time.Sleep(3 * time.Second)
		log.Info("poll for finished scans")
	}
}

func writeVulnerabilitiesToOC(containerImagesSource <-chan []string) {
	for {
		select {
		case images := <-containerImagesSource:
			log.Infof("TODO: update annotations, %v", images)
		}
	}
}

// other stuff

type config struct {
	Services   map[string]int // the map of service:port that sidecar scans over
	SvcTimeout int            // how long to wait for timeout when curling endpoints.
	Buckets    int            // number of exponentials for nslookup and so on .  easier to read if keep small unless debugging.
}

var (
	curlCheck *prometheus.HistogramVec
	nsLookup  *prometheus.HistogramVec
	cfg       *config
)

func initPrometheusAndViper() {
	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheus.Unregister(prometheus.NewGoCollector())

	cfg = viperLoad()
	viper.WatchConfig()

	curlCheck = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perceptron",
			Subsystem: "metrics",
			Name:      "curl",
			Help:      "The current CURL time for a service in milliseconds.",
			Buckets:   prometheus.ExponentialBuckets(1, 2, cfg.Buckets),
		},
		[]string{"service", "port", "status"})

	nsLookup = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perceptron",
			Subsystem: "metrics",
			Name:      "ns_lookup",
			Help:      "The current NS LOOKUP time for a service. Labelled with IP to detect schizophrenic resolution scenarios.",
			Buckets:   prometheus.ExponentialBuckets(1, 2, cfg.Buckets),
		},
		[]string{"service", "numIP"})

	prometheus.MustRegister(curlCheck)
	prometheus.MustRegister(nsLookup)

	// This allows someone to go into the container and change the curl endpoints.
	// use case: realtime debugging
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Info("Config file changed:", e.Name)
		cfg = viperLoad()
	})

	go func() {
		for {
			for svc, p := range cfg.Services {
				milliseconds, status, err := cfg.Curl(svc, p)
				if err != nil {
					curlCheck.WithLabelValues(svc, fmt.Sprintf("%v", p), status).Observe(float64(9999999))
				} else {
					curlCheck.WithLabelValues(svc, fmt.Sprintf("%v", p), status).Observe(float64(milliseconds))
				}
			}
			time.Sleep(5 * time.Second)
		}
	}()
	// Separate go thread for each metric, to avoid any forced correlation that might come from
	// the n+1th test being less efficient then the nth.  i.e. maybe nslookup the first time makes
	// curl faster the second time.
	go func() {
		for {
			for svc := range cfg.Services {
				milliseconds, ips, err := cfg.nSLookupIP(svc)
				if err != nil {
					nsLookup.WithLabelValues(svc, fmt.Sprintf("%v", len(ips))).Observe(float64(9999999))
				} else {
					nsLookup.WithLabelValues(svc, fmt.Sprintf("%v", len(ips))).Observe(float64(milliseconds))
				}
			}
			time.Sleep(5 * time.Second)
		}
	}()
}

func setupHTTPServer() {
	http.Handle("/metrics", prometheus.Handler())
	http.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		log.Info("shutting down")
		fmt.Fprint(w, "Shutdown now!\n")
		os.Exit(0)
	})
	log.Info("Serving")
	// TODO response to an edit-pods endpoint or something

	// TODO make this configurable - maybe even viperize it.
	http.ListenAndServe(":3000", nil)
	log.Info("Http server started!")
}

// Curl returns time to curl, status, and any error.
func (cfg *config) Curl(svc string, port int) (int64, string, error) {
	SvcTimeout := time.Duration(cfg.SvcTimeout) * time.Second
	client := http.Client{
		Timeout: SvcTimeout,
	}
	start := time.Now()
	url := strings.Join([]string{"http://" + svc, fmt.Sprintf("%v", port)}, ":")
	resp, err := client.Get(url)
	status := ""
	if resp != nil {
		status = resp.Status
	}
	log.Info(fmt.Sprintf("CURL Response from %v was %v : %v", url, status, err))
	elapsed := int64(time.Since(start) / time.Millisecond)
	return elapsed, status, err
}

func (cfg *config) nSLookupIP(svc string) (int64, []net.IP, error) {
	start := time.Now()
	ips, err := net.LookupIP(svc)
	log.Info(fmt.Sprintf("LOOKUP Response from %v was ips = %v : %v", svc, len(ips), err))
	elapsed := int64(time.Since(start) / time.Millisecond)
	return elapsed, ips, err
}

func (cfg *config) url(s string) string {
	return strings.Join(
		[]string{s, string(cfg.Services[s])},
		":")
}

func viperLoad() *config {
	// Default config: The blackducksoftware:hub services.  export ENV_CONFIG_JSON to override this.
	sidecarTargets := `{
	  "services":{
			"zookeeper":2181,
		  "cfssl":0,
		  "postgres":0,
		  "webapp":0,
		  "solr":0,
		  "documentation": 0
		},
		"svcTimeout":10,
		"buckets":3
	}`
	if v, ok := os.LookupEnv("ENV_CONFIG_JSON"); ok {
		sidecarTargets = v
	} else {
		log.Warn(`
      ENV_CONFIG_JSON services not provided as env var
		  Instead, writing default config to sidecar.json.
		  Edit it to reload the sidecar or restart w/ the right env var.
			EXAMPLE:
				export ENV_CONFIG_JSON="{\"services\":{\"zookeeper\":2181,\"cfssl\":5555,\"postgres\":5432, \"webapp\":8080, \"solr\":0, \"documentation\": 0 }, \"svcTimeout\":10}"
      `)
	}
	d1 := []byte(sidecarTargets)

	// Default config is written here.  We use file as a default config because it provides an
	// embedded self tests - users will probably always config by injecting env vars that get written to this file.
	err := ioutil.WriteFile("../../perceptron.json", d1, 0777)
	if err != nil {
		log.Fatal("Error writing default config file", err)
		panic(fmt.Sprint("Error writing default config file!", err))
	}

	viper.SetConfigName("perceptor") // name of config file (without extension)
	viper.AddConfigPath("../../")    // path to look for the config file in
	err = viper.ReadInConfig()       // Find and read the config file
	if err != nil {
		log.Errorf("Fatal error config file: %v \n", err)
	}

	var cfg *config

	// Read the viperized file input into the config struct.
	err = viper.Unmarshal(&cfg)
	if err != nil {
		log.Errorf("Error unmarshalling from Viper: %v\n", err)
	}

	return cfg
}
