package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
	"encoding/json"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron"
)

var addr = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
var location = flag.String("location", "KÖLN", "Location")
var every = flag.String("every", "15m", "Update time")
var myClient = &http.Client{Timeout: 10 * time.Second}

var (
	promMeasurement = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rheinpegel_measurement",
			Help: "Rheinpegel measurement",
		},
		[]string{"location"},
	)
	promTrend = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rheinpegel_trend",
			Help: "Rheinpegel trend",
		},
		[]string{"location"},
	)
)

type CurrentMeasurement struct {
	Timestamp 		time.Time
	Value     		float64
	Trend     		float64
	StateMnwMhw 	string
	StateNswHsw 	string
}

type GaugeZero struct {
	Unit 				string
	Value 			float64
	ValidFrom 	time.Time
}

type Measurement struct {
	Shortname 					string
	Longname 						string
	Unit 								string
	Equidistance 				string
	CurrentMeasurement 	CurrentMeasurement
	GaugeZero 					GaugeZero
}

func init() {
	prometheus.MustRegister(promMeasurement)
	prometheus.MustRegister(promTrend)
}

func getMeasurement(location string, target interface{}) error {
	var url = fmt.Sprintf("https://www.pegelonline.wsv.de/webservices/rest-api/v2/stations/%s/W.json?includeCurrentMeasurement=true", location)

  r, err := myClient.Get(url)
  if err != nil {
		log.Println(err)
    return err
  }
  defer r.Body.Close()

  return json.NewDecoder(r.Body).Decode(target)
}

func collectSample() {
	log.Println("Collecting sample...")
	currentMeasurement := new(Measurement)
	getMeasurement(*location, currentMeasurement)

	promMeasurement.With(prometheus.Labels{"location": *location}).Set(currentMeasurement.CurrentMeasurement.Value)
	promTrend.With(prometheus.Labels{"location": *location}).Set(currentMeasurement.CurrentMeasurement.Trend)
}

func main() {
	flag.Parse()
	http.Handle("/metrics", prometheus.Handler())

	collectSample()
	c := cron.New()
	c.AddFunc(fmt.Sprintf("@every %s", *every), collectSample)
	c.Start()

	log.Printf("Listening on %s!", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
