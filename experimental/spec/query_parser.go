package spec

import (
	"github.com/grafana/grafana-plugin-sdk-go/data/utils/jsoniter"
)

// GenericDataQuery is a replacement for `dtos.MetricRequest` with more explicit typing
type GenericDataQuery struct {
	CommonQueryProperties `json:",inline"`

	// Additional Properties (that live at the root)
	Additional map[string]any `json:",inline"`
}

// Generic query parser pattern.
type TypedQueryParser[Q any] interface {
	// Get the query parser for a query type
	// The version is split from the end of the discriminator field
	ParseQuery(
		// Properties that have been parsed off the same node
		common CommonQueryProperties,
		// An iterator with context for the full node (include common values)
		iter *jsoniter.Iterator,
	) (Q, error)
}

var _ TypedQueryParser[GenericDataQuery] = (*GenericQueryParser)(nil)

type GenericQueryParser struct{}

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

// ParseQuery implements TypedQueryParser.
func (*GenericQueryParser) ParseQuery(common CommonQueryProperties, iter *jsoniter.Iterator) (GenericDataQuery, error) {
	q := GenericDataQuery{CommonQueryProperties: common, Additional: make(map[string]any)}
	field, err := iter.ReadObject()
	for field != "" && err == nil {
		if !commonKeys[field] {
			q.Additional[field], err = iter.Read()
			if err != nil {
				return q, err
			}
		}
		field, err = iter.ReadObject()
	}
	return q, err
}
