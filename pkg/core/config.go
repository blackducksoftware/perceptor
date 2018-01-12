package core

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/prometheus/common/log"
	"github.com/spf13/viper"
)

type config struct {
	Services   map[string]int // the map of service:port that sidecar scans over
	SvcTimeout int            // how long to wait for timeout when curling endpoints.
	Buckets    int            // number of exponentials for nslookup and so on .  easier to read if keep small unless debugging.
}

func initViper() {
	cfg := viperLoad()
	viper.WatchConfig()

	// This allows someone to go into the container and change the curl endpoints.
	// use case: realtime debugging
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Info("Config file changed:", e.Name)
		cfg = viperLoad()
	})
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
	err := ioutil.WriteFile("../../perceptor.json", d1, 0777)
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
