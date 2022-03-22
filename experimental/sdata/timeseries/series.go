package timeseries

import (
	"fmt"
	"sort"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata"
)

type CollectionReader interface {
	// Validate will error if the data is invalid according to its type rules.
	// validateData will check for duplicate metrics and sorting and costs more resources.
	// If the data is valid, then any data that was not part of the time series data
	// is also returned as ignoredFieldIndices.
	Validate(validateData bool) (ignoredFieldIndices []sdata.FrameFieldIndex, err error)

	// GetMetricRefs runs validate without validateData. If the data is valid, then
	// []TimeSeriesMetricRef is returned from reading as well as any ignored data. If invalid,
	// then an error is returned, and no refs or ignoredFieldIndices are returned.
	GetMetricRefs() (refs []MetricRef, ignoredFieldIndices []sdata.FrameFieldIndex, err error)

	Frames() []*data.Frame // returns underlying frames
}

// MetricRef is for reading and contains the data for an individual
// time series. In the cases of the Multi and Wide formats, the Fields are pointers
// to the data in the original frame. In the case of Long new fields are constructed.
type MetricRef struct {
	TimeField  *data.Field
	ValueField *data.Field
	// TODO: RefID string
	// TODO: Pointer to frame meta?
}

func CollectionReaderFromFrames(frames []*data.Frame) (CollectionReader, error) {
	if len(frames) == 0 {
		return nil, fmt.Errorf("must be at least one frame")
	}

	firstFrame := frames[0]
	if firstFrame == nil {
		return nil, fmt.Errorf("nil frames are invalid")
	}
	if firstFrame.Meta == nil {
		return nil, fmt.Errorf("metadata missing from first frame, can not determine type")
	}

	mt := firstFrame.Meta.Type
	var tcr CollectionReader

	switch {
	case mt == data.FrameTypeTimeSeriesMany: // aka multi
		mfs := MultiFrame(frames)
		tcr = &mfs
	case mt == data.FrameTypeTimeSeriesLong:
		ls := LongFrame(frames)
		tcr = &ls // TODO change to Frames for extra/ignored data?
	case mt == data.FrameTypeTimeSeriesWide:
		wfs := WideFrame(frames)
		tcr = &wfs
	default:
		return nil, fmt.Errorf("unsupported time series type %q", mt)
	}
	return tcr, nil
}

func (m MetricRef) GetMetricName() string {
	if m.ValueField != nil {
		return m.ValueField.Name
	}
	return ""
}

// TODO GetFQMetric (or something, Names + Labels)

func (m MetricRef) GetLabels() data.Labels {
	if m.ValueField != nil {
		return m.ValueField.Labels
	}
	return nil
}

func sortTimeSeriesMetricRef(refs []MetricRef) {
	sort.SliceStable(refs, func(i, j int) bool {
		iRef := refs[i]
		jRef := refs[j]

		if iRef.GetMetricName() < jRef.GetMetricName() {
			return true
		}
		if iRef.GetMetricName() > jRef.GetMetricName() {
			return false
		}

		// If here Names are equal, next sort based on if there are labels.

		if iRef.GetLabels() == nil && jRef.GetLabels() == nil {
			return true // no labels first
		}
		if iRef.GetLabels() == nil && jRef.GetLabels() != nil {
			return true
		}
		if iRef.GetLabels() != nil && jRef.GetLabels() == nil {
			return false
		}

		return iRef.GetLabels().String() < jRef.GetLabels().String()
	})
}
