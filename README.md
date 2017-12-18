# canary
Canary is a metrics and metadata exporter that runs in any cloud native environment to help you embed SLO's into your microservice application bundles.

# Building

Run the dockerfile, it will build the binary for you:

docker build -t jayunit199/canary:1.0 .

the build.sh is just a hacky convenience script for local development, use at your own risk.

# Running

Run the replication controller :)

```
kubectl create -f ./canary-rc.json
```

# Using

Go into your pod that is running, called hub-sidecar-, and run something like this:

```
kubectl exec -t -i <name of the pod> curl localhost:3000/status
```

Example output:

```
# HELP sidecar_metrics_curl The current CURL time for a service in milliseconds.
# TYPE sidecar_metrics_curl histogram
sidecar_metrics_curl_bucket{port="0",service="cfssl",status="",le="1"} 0
sidecar_metrics_curl_bucket{port="0",service="cfssl",status="",le="2"} 0
sidecar_metrics_curl_bucket{port="0",service="cfssl",status="",le="4"} 0
sidecar_metrics_curl_bucket{port="0",service="cfssl",status="",le="+Inf"} 1
sidecar_metrics_curl_sum{port="0",service="cfssl",status=""} 9.999999e+06
sidecar_metrics_curl_count{port="0",service="cfssl",status=""} 1
sidecar_metrics_curl_bucket{port="0",service="documentation",status="",le="1"} 0
sidecar_metrics_curl_bucket{port="0",service="documentation",status="",le="2"} 0
sidecar_metrics_curl_bucket{port="0",service="documentation",status="",le="4"} 0
sidecar_metrics_curl_bucket{port="0",service="documentation",status="",le="+Inf"} 1
sidecar_metrics_curl_sum{port="0",service="documentation",status=""} 9.999999e+06
sidecar_metrics_curl_count{port="0",service="documentation",status=""} 1
sidecar_metrics_curl_bucket{port="0",service="postgres",status="",le="1"} 0
sidecar_metrics_curl_bucket{port="0",service="postgres",status="",le="2"} 0
sidecar_metrics_curl_bucket{port="0",service="postgres",status="",le="4"} 0
sidecar_metrics_curl_bucket{port="0",service="postgres",status="",le="+Inf"} 1
sidecar_metrics_curl_sum{port="0",service="postgres",status=""} 9.999999e+06
sidecar_metrics_curl_count{port="0",service="postgres",status=""} 1
sidecar_metrics_curl_bucket{port="0",service="solr",status="",le="1"} 0
sidecar_metrics_curl_bucket{port="0",service="solr",status="",le="2"} 0
sidecar_metrics_curl_bucket{port="0",service="solr",status="",le="4"} 0
sidecar_metrics_curl_bucket{port="0",service="solr",status="",le="+Inf"} 1
sidecar_metrics_curl_sum{port="0",service="solr",status=""} 9.999999e+06
sidecar_metrics_curl_count{port="0",service="solr",status=""} 1
sidecar_metrics_curl_bucket{port="0",service="webapp",status="",le="1"} 0
sidecar_metrics_curl_bucket{port="0",service="webapp",status="",le="2"} 0
sidecar_metrics_curl_bucket{port="0",service="webapp",status="",le="4"} 0
sidecar_metrics_curl_bucket{port="0",service="webapp",status="",le="+Inf"} 1
sidecar_metrics_curl_sum{port="0",service="webapp",status=""} 9.999999e+06
sidecar_metrics_curl_count{port="0",service="webapp",status=""} 1
sidecar_metrics_curl_bucket{port="2181",service="zookeeper",status="",le="1"} 0
sidecar_metrics_curl_bucket{port="2181",service="zookeeper",status="",le="2"} 0
sidecar_metrics_curl_bucket{port="2181",service="zookeeper",status="",le="4"} 0
sidecar_metrics_curl_bucket{port="2181",service="zookeeper",status="",le="+Inf"} 1
sidecar_metrics_curl_sum{port="2181",service="zookeeper",status=""} 9.999999e+06
sidecar_metrics_curl_count{port="2181",service="zookeeper",status=""} 1
# HELP sidecar_metrics_ns_lookup The current NS LOOKUP time for a service. Labelled with IP to detect schizophrenic resolution scenarios.
# TYPE sidecar_metrics_ns_lookup histogram
sidecar_metrics_ns_lookup_bucket{numIP="0",service="cfssl",le="1"} 0
sidecar_metrics_ns_lookup_bucket{numIP="0",service="cfssl",le="2"} 0
sidecar_metrics_ns_lookup_bucket{numIP="0",service="cfssl",le="4"} 0
sidecar_metrics_ns_lookup_bucket{numIP="0",service="cfssl",le="+Inf"} 1
sidecar_metrics_ns_lookup_sum{numIP="0",service="cfssl"} 9.999999e+06
sidecar_metrics_ns_lookup_count{numIP="0",service="cfssl"} 1
sidecar_metrics_ns_lookup_bucket{numIP="0",service="documentation",le="1"} 0
sidecar_metrics_ns_lookup_bucket{numIP="0",service="documentation",le="2"} 0
sidecar_metrics_ns_lookup_bucket{numIP="0",service="documentation",le="4"} 0
sidecar_metrics_ns_lookup_bucket{numIP="0",service="documentation",le="+Inf"} 1
sidecar_metrics_ns_lookup_sum{numIP="0",service="documentation"} 9.999999e+06
sidecar_metrics_ns_lookup_count{numIP="0",service="documentation"} 1
sidecar_metrics_ns_lookup_bucket{numIP="0",service="postgres",le="1"} 0
sidecar_metrics_ns_lookup_bucket{numIP="0",service="postgres",le="2"} 0
sidecar_metrics_ns_lookup_bucket{numIP="0",service="postgres",le="4"} 0
sidecar_metrics_ns_lookup_bucket{numIP="0",service="postgres",le="+Inf"} 1
sidecar_metrics_ns_lookup_sum{numIP="0",service="postgres"} 9.999999e+06
sidecar_metrics_ns_lookup_count{numIP="0",service="postgres"} 1
sidecar_metrics_ns_lookup_bucket{numIP="0",service="solr",le="1"} 0
sidecar_metrics_ns_lookup_bucket{numIP="0",service="solr",le="2"} 0
sidecar_metrics_ns_lookup_bucket{numIP="0",service="solr",le="4"} 0
sidecar_metrics_ns_lookup_bucket{numIP="0",service="solr",le="+Inf"} 1
sidecar_metrics_ns_lookup_sum{numIP="0",service="solr"} 9.999999e+06
sidecar_metrics_ns_lookup_count{numIP="0",service="solr"} 1
sidecar_metrics_ns_lookup_bucket{numIP="0",service="webapp",le="1"} 0
sidecar_metrics_ns_lookup_bucket{numIP="0",service="webapp",le="2"} 0
sidecar_metrics_ns_lookup_bucket{numIP="0",service="webapp",le="4"} 0
sidecar_metrics_ns_lookup_bucket{numIP="0",service="webapp",le="+Inf"} 1
sidecar_metrics_ns_lookup_sum{numIP="0",service="webapp"} 9.999999e+06
sidecar_metrics_ns_lookup_count{numIP="0",service="webapp"} 1
sidecar_metrics_ns_lookup_bucket{numIP="0",service="zookeeper",le="1"} 0
sidecar_metrics_ns_lookup_bucket{numIP="0",service="zookeeper",le="2"} 0
sidecar_metrics_ns_lookup_bucket{numIP="0",service="zookeeper",le="4"} 0
sidecar_metrics_ns_lookup_bucket{numIP="0",service="zookeeper",le="+Inf"} 1
sidecar_metrics_ns_lookup_sum{numIP="0",service="zookeeper"} 9.999999e+06
sidecar_metrics_ns_lookup_count{numIP="0",service="zookeeper"} 1
```

# development policy
Canary embraces the traditional values of open source projects in the Apache and CNCF communities, and embraces ideas and community over the code itself.  Please create an issue or better yet, submit a pull request if you have any suggestions around metrics or checks that you think will be generically useful to organizations that ship code which is meant to run in a microservice environment.

# golang standards
We follow the same standards for golang as are followed in the moby project, the kubernetes project, and other major golang projects.  We embrace modern golang idioms including usage of viper (for config), glide (for dependencies), and aim to stay on the 'bleeding edge', since, after all, we aim to always deploy inside of containers.
