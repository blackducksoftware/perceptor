package main

import (
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

/*
func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}*/

func main() {
	metrics := setup()
	handler := prometheus.Handler()
	go startDoingRandomMetrics(metrics)

	http.Handle("/", handler)
	http.Handle("/metrics", handler)
	http.ListenAndServe(":9999", nil)
	log.Info("Http server started!")
}

func startDoingRandomMetrics(m *metrics) {
	log.Info("starting random metrics")
	for i := 0; ; i++ {
		m.hist.Observe(float64(i))
		// m.histVec.WithLabelValues("first", "second").Observe(float64(i))
		// log.Infof("%+v", m.counter.Desc())
		code := "200"
		method := "GET"
		path := "team"
		if i%5 == 0 {
			code = "404"
		}
		if i%13 == 0 {
			method = "POST"
		}
		if i%7 == 0 {
			path = "user"
		}
		m.counterVec.With(prometheus.Labels{"code": code, "method": method, "path": path}).Inc()
		// m.histVec.WithLabelValues("first").Observe(float64(i / 2))
		// m.histVec.WithLabelValues("second").Observe(float64(i * 2))
		thing := "pull on something"
		if i%2 == 0 {
			thing = "create"
		}
		seconds := time.Duration(i%100) * time.Minute
		m.histVec.With(prometheus.Labels{"thing": thing}).Observe(seconds.Seconds())
		time.Sleep(100 * time.Millisecond)
	}
	log.Info("done with random metrics")
}

type metrics struct {
	hist       prometheus.Histogram
	histVec    *prometheus.HistogramVec
	counter    prometheus.Counter
	counterVec *prometheus.CounterVec
}

func setup() *metrics {
	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheus.Unregister(prometheus.NewGoCollector())

	// vec := prometheus.NewHistogramVec(
	// 	prometheus.HistogramOpts{
	// 		Namespace:   "perceptor",
	// 		Subsystem:   "perceptor_core",
	// 		Name:        "some_other_name",
	// 		Help:        "whatevs",
	// 		ConstLabels: map[string]string{},
	// 		Buckets:     prometheus.ExponentialBuckets(0.5, 2, 30),
	// 	},
	// 	[]string{"first", "second"})
	hist := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace:   "perceptor",
		Subsystem:   "perceptor_core",
		Name:        "some_name_i_guess",
		ConstLabels: map[string]string{},
		Help:        "requests, handled by perceptor core, which were successful",
		// Buckets:     prometheus.LinearBuckets(0, 1, 8),
		Buckets: prometheus.ExponentialBuckets(0.5, 2, 40),
	})
	counter1 := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace:   "dunno_namespace",
		Name:        "dunno_name",
		Subsystem:   "dunno_subsystem",
		Help:        "dunno help",
		ConstLabels: map[string]string{},
	})
	counterVec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "dunno_namespace_vec",
		Name:        "dunno_name_vec",
		Subsystem:   "dunno_subsystem_vec",
		Help:        "dunno help vec",
		ConstLabels: map[string]string{},
	},
		[]string{"path", "code", "method"})

	durations := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "testing",
			Subsystem: "some_durations",
			Name:      "whaddya_know",
			Help:      "some help",
			Buckets:   prometheus.ExponentialBuckets(0.25, 2, 20),
		},
		[]string{"thing"})

	prometheus.MustRegister(hist)
	prometheus.MustRegister(durations)
	prometheus.MustRegister(counter1)
	prometheus.MustRegister(counterVec)

	return &metrics{
		hist:       hist,
		histVec:    durations,
		counter:    counter1,
		counterVec: counterVec,
	}
}
