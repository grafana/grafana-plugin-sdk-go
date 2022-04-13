package tabular

import (
	"errors"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

const FrameTypeHistogram = "histogram"

type Histogram struct {
	*data.Frame
}

type HistogramOptions struct {
	MinValue   float64
	MaxValue   float64
	BucketSize float64
}

func NewHistogramFrame(options HistogramOptions) (*Histogram, error) {
	type bucket struct {
		MinValue float64
		MaxValue float64
	}
	minValue := options.MinValue
	maxValue := options.MaxValue
	bucketSize := options.BucketSize
	if bucketSize <= 0 {
		return nil, errors.New("invalid bucket size")
	}
	if maxValue <= minValue {
		return nil, errors.New("max value should be greater than min value")
	}
	buckets := []bucket{}
	for i := minValue; i < maxValue; i += bucketSize {
		min := i
		max := i + bucketSize
		if i+bucketSize > maxValue {
			max = maxValue
		}
		buckets = append(buckets, bucket{
			MinValue: min,
			MaxValue: max,
		})
	}
	histogramFrame := data.NewFrame("")
	histogramFrame.SetMeta(&data.FrameMeta{Type: data.FrameTypeTable})
	bucketMinField := data.NewFieldFromFieldType(data.FieldTypeFloat64, len(buckets))
	bucketMinField.Name = "BucketMin"
	bucketMaxField := data.NewFieldFromFieldType(data.FieldTypeFloat64, len(buckets))
	bucketMaxField.Name = "BucketMax"
	for idx, bkt := range buckets {
		bucketMinField.Set(idx, bkt.MinValue)
		bucketMaxField.Set(idx, bkt.MaxValue)
	}
	histogramFrame.Fields = append(histogramFrame.Fields, bucketMinField, bucketMaxField)
	return &Histogram{histogramFrame}, nil
}

func NewHistogramFrameWithValues(options HistogramOptions, metricName string, values []float64) (*Histogram, error) {
	hg, err := NewHistogramFrame(options)
	if err != nil {
		return nil, err
	}
	err = hg.AddValue(metricName, make(data.Labels), values)
	return hg, err
}

func (hg *Histogram) AddValue(metricName string, labels data.Labels, values []float64) error {
	if len(hg.Fields) < 2 {
		return errors.New("fields BucketMin/BucketMax are not found in the histogram frame")
	}
	if hg.Fields[0].Name != "BucketMin" {
		return errors.New("first field of histogram should be BucketMin")
	}
	if hg.Fields[1].Name != "BucketMax" {
		return errors.New("second field of histogram should be BucketMax")
	}
	if len(values) != hg.Rows() {
		return fmt.Errorf("number of values not matching the buckets length %d", hg.Rows())
	}
	valueField := data.NewField(metricName, labels, values)
	fields := append(hg.Fields, valueField)
	hg.Fields = fields
	return nil
}

func (hg *Histogram) Validate() (isEmpty bool, errs []error) {
	if len(hg.Fields) < 2 {
		errs = append(errs, errors.New("histogram require BucketMin and BucketMax fields"))
		return false, errs
	}
	if len(hg.Fields) >= 2 {
		if hg.Fields[0].Name != "BucketMin" {
			errs = append(errs, errors.New("first field of histogram should be BucketMin"))
		}
		if hg.Fields[1].Name != "BucketMax" {
			errs = append(errs, errors.New("first field of histogram should be BucketMax"))
		}
		if len(hg.Fields) == 2 {
			errs = append(errs, errors.New("no value fields found"))
			// TODO: Can histogram ever be empty ???
			return true, errs
		}
		for idx, field := range hg.Fields {
			if field.Type() != data.FieldTypeFloat64 {
				// only FieldTypeFloat64 should present in histogram
				errs = append(errs, fmt.Errorf("type of field %d is not float64", idx))
				return
			}
		}
	}
	return false, errs
}

func FrameToHistogram(frame data.Frame, options HistogramOptions) (*Histogram, error) {
	return nil, errors.New("not implemented")
}

func LongFrameToHistogram(frame data.Frame, options HistogramOptions) (*Histogram, error) {
	return nil, errors.New("not implemented")
}

func WideFrameToHistogram(frame data.Frame, options HistogramOptions) (*Histogram, error) {
	return nil, errors.New("not implemented")
}

func MultiFrameToHistogram(frame data.Frame, options HistogramOptions) (*Histogram, error) {
	return nil, errors.New("not implemented")
}
