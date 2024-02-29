package resource

import (
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data/utils/jsoniter"
)

func ParseQueryRequest(iter *jsoniter.Iterator) (*GenericQueryRequest, error) {
	return ParseTypedQueryRequest[*GenericDataQuery](&genericQueryReader{}, iter, time.Now())
}

type TypedQueryReader[T DataQuery] interface {
	// Called before any custom property is found
	Start(p *CommonQueryProperties, now time.Time) error
	// Called for each non-common property
	SetProperty(key string, iter *jsoniter.Iterator) error
	// Finished reading the JSON node
	Finish() (T, error)
}

func ParseTypedQueryRequest[T DataQuery](reader TypedQueryReader[T], iter *jsoniter.Iterator, now time.Time) (*QueryRequest[T], error) {
	var err error
	var root string
	var ok bool
	dqr := &QueryRequest[T]{}
	for root, err = iter.ReadObject(); root != ""; root, err = iter.ReadObject() {
		switch root {
		case "to":
			dqr.To, err = iter.ReadString()
		case "from":
			dqr.From, err = iter.ReadString()
		case "debug":
			dqr.Debug, err = iter.ReadBool()
		case "queries":
			ok, err = iter.ReadArray()
			for ok && err == nil {
				props := &CommonQueryProperties{}
				err = reader.Start(props, now)
				if err != nil {
					return dqr, err
				}
				err = props.readQuery(iter, reader.SetProperty)
				if err != nil {
					return dqr, err
				}

				q, err := reader.Finish()
				if err != nil {
					return dqr, err
				}
				dqr.Queries = append(dqr.Queries, q)

				ok, err = iter.ReadArray()
				if err != nil {
					return dqr, err
				}
			}
		default:
			// ignored? or error
		}
		if err != nil {
			return dqr, err
		}
	}
	return dqr, err
}

var _ TypedQueryReader[*GenericDataQuery] = (*genericQueryReader)(nil)

type genericQueryReader struct {
	common     *CommonQueryProperties
	additional map[string]any
}

// Called before any custom properties are passed
func (g *genericQueryReader) Start(p *CommonQueryProperties, _ time.Time) error {
	g.additional = make(map[string]any)
	g.common = p
	return nil
}

func (g *genericQueryReader) SetProperty(key string, iter *jsoniter.Iterator) error {
	v, err := iter.Read()
	if err != nil {
		return err
	}
	g.additional[key] = v
	return nil
}

// Finished the JSON node, return a query object
func (g *genericQueryReader) Finish() (*GenericDataQuery, error) {
	return &GenericDataQuery{
		CommonQueryProperties: *g.common,
		additional:            g.additional,
	}, nil
}
