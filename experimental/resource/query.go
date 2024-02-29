package resource

import (
	"embed"
	"encoding/json"
	"fmt"
	"unsafe"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/data/converters"
	"github.com/grafana/grafana-plugin-sdk-go/data/utils/jsoniter"
	j "github.com/json-iterator/go"
	openapi "k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

func init() { //nolint:gochecknoinits
	jsoniter.RegisterTypeEncoder("resource.GenericDataQuery", &genericQueryCodec{})
	jsoniter.RegisterTypeDecoder("resource.GenericDataQuery", &genericQueryCodec{})
}

type DataQuery interface {
	// The standard query properties
	CommonProperties() *CommonQueryProperties

	// For queries that depend on other queries to run first (eg, other refIds)
	Dependencies() []string
}

type QueryRequest[Q DataQuery] struct {
	// From Start time in epoch timestamps in milliseconds or relative using Grafana time units.
	// example: now-1h
	From string `json:"from,omitempty"`

	// To End time in epoch timestamps in milliseconds or relative using Grafana time units.
	// example: now
	To string `json:"to,omitempty"`

	// Each item has a
	Queries []Q `json:"queries"`

	// required: false
	Debug bool `json:"debug,omitempty"`
}

// GenericQueryRequest is a query request that supports any datasource
type GenericQueryRequest = QueryRequest[*GenericDataQuery]

var _ DataQuery = (*GenericDataQuery)(nil)

// GenericDataQuery is a replacement for `dtos.MetricRequest` with more explicit typing
type GenericDataQuery struct {
	CommonQueryProperties `json:",inline"`

	// Additional Properties (that live at the root)
	additional map[string]any `json:"-"` // note this uses custom JSON marshalling
}

// CommonProperties implements DataQuery.
func (g *GenericDataQuery) CommonProperties() *CommonQueryProperties {
	return &g.CommonQueryProperties
}

// Dependencies implements DataQuery.
func (g *GenericDataQuery) Dependencies() []string {
	return nil
}

// Produce an API definition that represents map[string]any
func (g GenericDataQuery) OpenAPIDefinition() openapi.OpenAPIDefinition {
	s, _ := GenericQuerySchema()
	if s == nil {
		s = &spec.Schema{}
	}
	s.SchemaProps.Type = []string{"object"}
	s.SchemaProps.AdditionalProperties = &spec.SchemaOrBool{Allows: true}
	return openapi.OpenAPIDefinition{Schema: *s}
}

func NewGenericDataQuery(body map[string]any) GenericDataQuery {
	g := &GenericDataQuery{
		additional: make(map[string]any),
	}
	for k, v := range body {
		_ = g.Set(k, v)
	}
	return *g
}

func (g *GenericDataQuery) DeepCopy() *GenericDataQuery {
	if g == nil {
		return nil
	}
	out := new(GenericDataQuery)
	jj, err := json.Marshal(g)
	if err == nil {
		_ = json.Unmarshal(jj, out)
	}
	return out
}

func (g *GenericDataQuery) DeepCopyInto(out *GenericDataQuery) {
	clone := g.DeepCopy()
	*out = *clone
}

// Set allows setting values using key/value pairs
func (g *GenericDataQuery) Set(key string, val any) *GenericDataQuery {
	switch key {
	case "refId":
		g.RefID, _ = val.(string)
	case "resultAssertions":
		body, err := json.Marshal(val)
		if err != nil {
			_ = json.Unmarshal(body, &g.ResultAssertions)
		}
	case "timeRange":
		body, err := json.Marshal(val)
		if err != nil {
			_ = json.Unmarshal(body, &g.TimeRange)
		}
	case "datasource":
		body, err := json.Marshal(val)
		if err != nil {
			_ = json.Unmarshal(body, &g.Datasource)
		}
	case "datasourceId":
		v, err := converters.JSONValueToInt64.Converter(val)
		if err != nil {
			g.DatasourceID, _ = v.(int64)
		}
	case "queryType":
		g.QueryType, _ = val.(string)
	case "maxDataPoints":
		v, err := converters.JSONValueToInt64.Converter(val)
		if err != nil {
			g.MaxDataPoints, _ = v.(int64)
		}
	case "intervalMs":
		v, err := converters.JSONValueToFloat64.Converter(val)
		if err != nil {
			g.IntervalMS, _ = v.(float64)
		}
	case "hide":
		g.Hide, _ = val.(bool)
	default:
		if g.additional == nil {
			g.additional = make(map[string]any)
		}
		g.additional[key] = val
	}
	return g
}

func (g *GenericDataQuery) Get(key string) (any, bool) {
	switch key {
	case "refId":
		return g.RefID, true
	case "resultAssertions":
		return g.ResultAssertions, true
	case "timeRange":
		return g.TimeRange, true
	case "datasource":
		return g.Datasource, true
	case "datasourceId":
		return g.DatasourceID, true
	case "queryType":
		return g.QueryType, true
	case "maxDataPoints":
		return g.MaxDataPoints, true
	case "intervalMs":
		return g.IntervalMS, true
	case "hide":
		return g.Hide, true
	}
	v, ok := g.additional[key]
	return v, ok
}

func (g *GenericDataQuery) MustString(key string) string {
	v, ok := g.Get(key)
	if ok {
		s, ok := v.(string)
		if ok {
			return s
		}
	}
	return ""
}

type genericQueryCodec struct{}

func (codec *genericQueryCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (codec *genericQueryCodec) Encode(ptr unsafe.Pointer, stream *j.Stream) {
	q := (*GenericDataQuery)(ptr)
	writeQuery(q, stream)
}

func (codec *genericQueryCodec) Decode(ptr unsafe.Pointer, iter *j.Iterator) {
	q := GenericDataQuery{}
	err := q.readQuery(jsoniter.NewIterator(iter))
	if err != nil {
		// keep existing iter error if it exists
		if iter.Error == nil {
			iter.Error = err
		}
		return
	}
	*((*GenericDataQuery)(ptr)) = q
}

// MarshalJSON writes JSON including the common and custom values
func (g GenericDataQuery) MarshalJSON() ([]byte, error) {
	cfg := j.ConfigCompatibleWithStandardLibrary
	stream := cfg.BorrowStream(nil)
	defer cfg.ReturnStream(stream)

	writeQuery(&g, stream)
	return append([]byte(nil), stream.Buffer()...), stream.Error
}

// UnmarshalJSON reads a query from json byte array
func (g *GenericDataQuery) UnmarshalJSON(b []byte) error {
	iter, err := jsoniter.ParseBytes(jsoniter.ConfigDefault, b)
	if err != nil {
		return err
	}
	return g.readQuery(iter)
}

func writeQuery(g *GenericDataQuery, stream *j.Stream) {
	q := g.CommonQueryProperties
	stream.WriteObjectStart()
	stream.WriteObjectField("refId")
	stream.WriteVal(g.RefID)

	if q.ResultAssertions != nil {
		stream.WriteMore()
		stream.WriteObjectField("resultAssertions")
		stream.WriteVal(g.ResultAssertions)
	}

	if q.TimeRange != nil {
		stream.WriteMore()
		stream.WriteObjectField("timeRange")
		stream.WriteVal(g.TimeRange)
	}

	if q.Datasource != nil {
		stream.WriteMore()
		stream.WriteObjectField("datasource")
		stream.WriteVal(g.Datasource)
	}

	if q.DatasourceID > 0 {
		stream.WriteMore()
		stream.WriteObjectField("datasourceId")
		stream.WriteVal(g.DatasourceID)
	}

	if q.QueryType != "" {
		stream.WriteMore()
		stream.WriteObjectField("queryType")
		stream.WriteVal(g.QueryType)
	}

	if q.MaxDataPoints > 0 {
		stream.WriteMore()
		stream.WriteObjectField("maxDataPoints")
		stream.WriteVal(g.MaxDataPoints)
	}

	if q.IntervalMS > 0 {
		stream.WriteMore()
		stream.WriteObjectField("intervalMs")
		stream.WriteVal(g.IntervalMS)
	}

	if q.Hide {
		stream.WriteMore()
		stream.WriteObjectField("hide")
		stream.WriteVal(g.Hide)
	}

	// The additional properties
	if g.additional != nil {
		for k, v := range g.additional {
			stream.WriteMore()
			stream.WriteObjectField(k)
			stream.WriteVal(v)
		}
	}
	stream.WriteObjectEnd()
}

func (g *GenericDataQuery) readQuery(iter *jsoniter.Iterator) error {
	return g.CommonQueryProperties.readQuery(iter, func(key string, iter *jsoniter.Iterator) error {
		if g.additional == nil {
			g.additional = make(map[string]any)
		}
		v, err := iter.Read()
		g.additional[key] = v
		return err
	})
}

func (g *CommonQueryProperties) readQuery(iter *jsoniter.Iterator,
	processUnknownKey func(key string, iter *jsoniter.Iterator) error,
) error {
	var err error
	var next j.ValueType
	field := ""
	for field, err = iter.ReadObject(); field != ""; field, err = iter.ReadObject() {
		switch field {
		case "refId":
			g.RefID, err = iter.ReadString()
		case "resultAssertions":
			err = iter.ReadVal(&g.ResultAssertions)
		case "timeRange":
			err = iter.ReadVal(&g.TimeRange)
		case "datasource":
			// Old datasource values may just be a string
			next, err = iter.WhatIsNext()
			if err == nil {
				switch next {
				case j.StringValue:
					g.Datasource = &DataSourceRef{}
					g.Datasource.UID, err = iter.ReadString()
				case j.ObjectValue:
					err = iter.ReadVal(&g.Datasource)
				default:
					return fmt.Errorf("expected string or object")
				}
			}

		case "datasourceId":
			g.DatasourceID, err = iter.ReadInt64()
		case "queryType":
			g.QueryType, err = iter.ReadString()
		case "maxDataPoints":
			g.MaxDataPoints, err = iter.ReadInt64()
		case "intervalMs":
			g.IntervalMS, err = iter.ReadFloat64()
		case "hide":
			g.Hide, err = iter.ReadBool()
		default:
			err = processUnknownKey(field, iter)
		}
		if err != nil {
			return err
		}
	}
	return err
}

type CommonQueryProperties struct {
	// RefID is the unique identifier of the query, set by the frontend call.
	RefID string `json:"refId,omitempty"`

	// Optionally define expected query result behavior
	ResultAssertions *ResultAssertions `json:"resultAssertions,omitempty"`

	// TimeRange represents the query range
	// NOTE: unlike generic /ds/query, we can now send explicit time values in each query
	// NOTE: the values for timeRange are not saved in a dashboard, they are constructed on the fly
	TimeRange *TimeRange `json:"timeRange,omitempty"`

	// The datasource
	Datasource *DataSourceRef `json:"datasource,omitempty"`

	// Deprecated -- use datasource ref instead
	DatasourceID int64 `json:"datasourceId,omitempty"`

	// QueryType is an optional identifier for the type of query.
	// It can be used to distinguish different types of queries.
	QueryType string `json:"queryType,omitempty"`

	// MaxDataPoints is the maximum number of data points that should be returned from a time series query.
	// NOTE: the values for maxDataPoints is not saved in the query model.  It is typically calculated
	// from the number of pixels visible in a visualization
	MaxDataPoints int64 `json:"maxDataPoints,omitempty"`

	// Interval is the suggested duration between time points in a time series query.
	// NOTE: the values for intervalMs is not saved in the query model.  It is typically calculated
	// from the interval required to fill a pixels in the visualization
	IntervalMS float64 `json:"intervalMs,omitempty"`

	// true if query is disabled (ie should not be returned to the dashboard)
	// NOTE: this does not always imply that the query should not be executed since
	// the results from a hidden query may be used as the input to other queries (SSE etc)
	Hide bool `json:"hide,omitempty"`
}

type DataSourceRef struct {
	// The datasource plugin type
	Type string `json:"type"`

	// Datasource UID
	UID string `json:"uid,omitempty"`

	// ?? the datasource API version?  (just version, not the group? type | apiVersion?)
}

// TimeRange represents a time range for a query and is a property of DataQuery.
type TimeRange struct {
	// From is the start time of the query.
	From string `json:"from" jsonschema:"example=now-1h"`

	// To is the end time of the query.
	To string `json:"to" jsonschema:"example=now"`
}

// ResultAssertions define the expected response shape and query behavior.  This is useful to
// enforce behavior over time.  The assertions are passed to the query engine and can be used
// to fail queries *before* returning them to a client (select * from bigquery!)
type ResultAssertions struct {
	// Type asserts that the frame matches a known type structure.
	Type data.FrameType `json:"type,omitempty"`

	// TypeVersion is the version of the Type property. Versions greater than 0.0 correspond to the dataplane
	// contract documentation https://grafana.github.io/dataplane/contract/.
	TypeVersion data.FrameTypeVersion `json:"typeVersion"`

	// Maximum bytes that can be read -- if the query planning expects more then this, the query may fail fast
	MaxBytes int64 `json:"maxBytes,omitempty"`

	// Maximum frame count
	MaxFrames int64 `json:"maxFrames,omitempty"`
}

//go:embed query.schema.json query.definition.schema.json
var f embed.FS

// Get the cached feature list (exposed as a k8s resource)
func GenericQuerySchema() (*spec.Schema, error) {
	body, err := f.ReadFile("query.schema.json")
	if err != nil {
		return nil, err
	}
	s := &spec.Schema{}
	err = s.UnmarshalJSON(body)
	return s, err
}
