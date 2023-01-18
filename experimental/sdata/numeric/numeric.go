package numeric

import (
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata"
)

type CollectionWriter interface {
	AddMetric(metricName string, l data.Labels, value interface{}) error
	SetMetricMD(metricName string, l data.Labels, fc data.FieldConfig)
}

type CollectionRW interface {
	CollectionWriter
	CollectionReader
}

type CollectionReader interface {
	GetCollection(validateData bool) (Collection, error)
}

type MetricRef struct {
	ValueField *data.Field
}

func (n MetricRef) GetMetricName() string {
	if n.ValueField != nil {
		return n.ValueField.Name
	}
	return ""
}

func (n MetricRef) GetLabels() data.Labels {
	if n.ValueField != nil {
		return n.ValueField.Labels
	}
	return nil
}

type Collection struct {
	Refs             []MetricRef
	RemainderIndices []sdata.FrameFieldIndex
	Warning          error
}
