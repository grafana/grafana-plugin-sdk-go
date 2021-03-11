package injest

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/influxdata/telegraf/plugins/parsers/influx"
)

func RunServer() {
	http.HandleFunc("/", handler)
	fmt.Println("Dummy injest backend Listening on port: 7777")
	if err := http.ListenAndServe(":7777", nil); err != nil {
		panic(err)
	}
}

func setupResponse(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

var count = 0

var allstreams = make(map[uint64]MetricFrameStream, 5)

func handler(w http.ResponseWriter, r *http.Request) {
	setupResponse(&w)
	if r.Method == "OPTIONS" {
		return
	}

	count++

	handler := influx.NewMetricHandler()
	parser := influx.NewParser(handler)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	metrics, err := parser.Parse(body)
	if err != nil {
		log.Printf("Error making metrics: %v", err)
		http.Error(w, "error making metrics", http.StatusBadRequest)
		return
	}

	batch := make(map[uint64]MetricFrameStream, 5)

	for _, m := range metrics {
		id := m.HashID()
		stream, ok := batch[id]
		if ok {
			// Same batch
			stream.Append(m)
		} else {
			stream, ok = allstreams[id]
			if ok {
				stream.Clear()
				stream.Append(m)
			} else {
				stream, err = NewMetricFrameStream(m)
				if err != nil {
					log.Printf("error making frame: %v\n", err)
					continue
				}
				allstreams[id] = stream

				s, _ := data.FrameToJSON(stream.Frame, true, false)
				log.Printf("[ADDING] %s\n", string(s))
			}
			batch[id] = stream
		}
	}

	for _, v := range batch {
		s, _ := data.FrameToJSON(v.Frame, false, true)
		log.Printf("[%d] %s [%d rows] %s\n", count, v.Frame.Name, v.Frame.Rows(), string(s))
	}
}
