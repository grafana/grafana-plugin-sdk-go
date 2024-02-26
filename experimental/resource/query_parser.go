package resource

import (
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data/utils/jsoniter"
)

type QueryRequest[Q any] struct {
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
type GenericQueryRequest = QueryRequest[GenericDataQuery]

// Generic query parser pattern.
type TypedQueryParser[Q any] interface {
	// Get the query parser for a query type
	// The version is split from the end of the discriminator field
	ParseQuery(
		// Properties that have been parsed off the same node
		common CommonQueryProperties,
		// An iterator with context for the full node (include common values)
		iter *jsoniter.Iterator,
		// Use this value as "now"
		now time.Time,
	) (Q, error)
}

var commonKeys = map[string]bool{
	"refId":            true,
	"resultAssertions": true,
	"timeRange":        true,
	"datasource":       true,
	"datasourceId":     true,
	"queryType":        true,
	"maxDataPoints":    true,
	"intervalMs":       true,
	"hide":             true,
}

var _ TypedQueryParser[GenericDataQuery] = (*GenericQueryParser)(nil)

type GenericQueryParser struct{}

// ParseQuery implements TypedQueryParser.
func (*GenericQueryParser) ParseQuery(common CommonQueryProperties, iter *jsoniter.Iterator, _ time.Time) (GenericDataQuery, error) {
	q := GenericDataQuery{CommonQueryProperties: common, additional: make(map[string]any)}
	field, err := iter.ReadObject()
	for field != "" && err == nil {
		if !commonKeys[field] {
			q.additional[field], err = iter.Read()
			if err != nil {
				return q, err
			}
		}
		field, err = iter.ReadObject()
	}
	return q, err
}
