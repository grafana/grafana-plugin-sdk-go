package remotewrite

import (
	"fmt"
	"hash/fnv"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/prometheus/prometheus/prompb"
)

type metricKey uint64

func Serialize(frames ...*data.Frame) ([]byte, error) {
	ts := tsFromFrames(frames...)
	return tsToBytes(ts)
}

func tsFromFrames(frames ...*data.Frame) []*prompb.TimeSeries {
	var entries = make(map[metricKey]prompb.TimeSeries)

	for _, frame := range frames {
		var promTimeSeries prompb.TimeSeries
		for _, field := range frame.Fields {
			if !field.Type().Numeric() {
				continue
			}
			metricName := frame.Name + "_" + field.Name
			metricName, ok := sanitizeMetricName(metricName)
			if !ok {
				continue
			}

			var samples []prompb.Sample

			labels := createLabels(field.Labels)
			key := makeMetricKey(metricName, labels)

			for i := 0; i < field.Len(); i++ {
				val, ok := field.ConcreteAt(i)
				if !ok {
					continue
				}
				value, ok := sampleValue(val)
				if !ok {
					continue
				}
				tm, ok := frame.Fields[0].ConcreteAt(i)
				if !ok {
					continue
				}

				sample := prompb.Sample{
					// Timestamp is int milliseconds for remote write.
					Timestamp: toSampleTime(tm.(time.Time)),
					Value:     value,
				}
				samples = append(samples, sample)
			}

			promTimeSeries = prompb.TimeSeries{Labels: labels, Samples: samples}
			entries[key] = promTimeSeries
		}
	}

	var promTS = make([]*prompb.TimeSeries, len(entries))
	var i int
	for _, promTs := range entries {
		promTS[i] = &promTs
		i++
	}

	return promTS
}

func toSampleTime(tm time.Time) int64 {
	return tm.UnixNano() / int64(time.Millisecond)
}

func tsToBytes(ts []*prompb.TimeSeries) ([]byte, error) {
	writeRequestData, err := proto.Marshal(&prompb.WriteRequest{Timeseries: ts})
	if err != nil {
		return nil, fmt.Errorf("unable to marshal protobuf: %v", err)
	}
	return snappy.Encode(nil, writeRequestData), nil
}

func makeMetricKey(name string, labels []*prompb.Label) metricKey {
	h := fnv.New64a()
	_, _ = h.Write([]byte(name))
	for _, label := range labels {
		_, _ = h.Write([]byte(label.Name))
		_, _ = h.Write([]byte("\x00"))
		_, _ = h.Write([]byte(label.Value))
		_, _ = h.Write([]byte("\x00"))
	}
	return metricKey(h.Sum64())
}

func createLabels(fieldLabels map[string]string) []*prompb.Label {
	labels := make([]*prompb.Label, 0, len(fieldLabels))
	for k, v := range fieldLabels {
		sanitizedName, ok := sanitizeLabelName(k)
		if !ok {
			continue
		}
		labels = append(labels, &prompb.Label{Name: sanitizedName, Value: v})
	}
	return labels
}
