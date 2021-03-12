package injest

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

var events = make(chan FrameEvent)

func RunServer() {
	live, err := Connect("http://localhost:3000")
	if err != nil {
		panic("error starting live")
	}

	http.HandleFunc("/", handler)
	fmt.Println("Dummy injester listening on port: 7777")

	// Async
	go func() {

		for evt := range events {
			ch := live.getChannel(evt.Key)

			js, _ := data.FrameToJSON(evt.Frame, !evt.Append, true)
			ch.Publish(js)

			fmt.Printf("wrote: grafana/broadcast/telegraf/%s [%d]\n", evt.Key, evt.Frame.Rows())

			//			log.Printf("%s [%d rows] %s\n", evt.Key, evt.Frame.Rows(), string(js))

			// log.Printf("%s [append=%v]\n", evt.Key, evt.Append)
		}
	}()

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

	parser := NewInfluxParser()

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

	created := make(map[uint64]bool, 5)
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
				created[id] = true // flag for append vs new schema
			}
			batch[id] = stream
		}
	}

	for _, v := range batch {
		isNew := created[v.id]
		events <- FrameEvent{
			Key:    v.Key,
			Frame:  v.Frame,
			Append: !isNew,
		}
	}
}
