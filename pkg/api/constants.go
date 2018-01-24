package api

// Two things that should work:
// curl -X GET http://perceptor.bds-perceptor.svc.cluster.local:3001/metrics
// curl -X GET http://perceptor.bds-perceptor:3001/metrics
const (
	PerceptorBaseURL = "http://perceptor.bds-perceptor"
	// perceptor-scanner paths
	NextImagePath    = "nextimage"
	FinishedScanPath = "finishedscan"
	// perceiver paths
	PodPath         = "pod"
	ScanResultsPath = "scanresults"
	// ports (basically so that you can run these locally without them stomping on each other -- for testing)
	PerceptorPort        = "3001"
	PerceiverPort        = "3002"
	PerceptorScannerPort = "3003"
)
