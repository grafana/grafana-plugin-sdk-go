package datasource

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

var (
	// ErrorNotCached is returned when a cached value was not found in the internal cache
	ErrorNotCached = errors.New("no cached response found")
)

// A CachedResponse is the combination of a response and an expiration timestamp
type CachedResponse struct {
	ExpiresAt time.Time
	Frames    data.Frames
}

func getCacheKey(duration time.Duration, req backend.DataQuery) (string, error) {
	m := map[string]interface{}{
		"query":    req.JSON,
		"interval": req.Interval,
		"type":     req.QueryType,
		"time_range": backend.TimeRange{
			To:   req.TimeRange.To.Round(duration),
			From: req.TimeRange.From.Round(duration),
		},
	}

	b := bytes.NewBuffer(nil)
	if err := json.NewEncoder(b).Encode(m); err != nil {
		return "", err
	}

	h := sha256.New()
	if _, err := h.Write(b.Bytes()); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil

}

// CachedQueryDatahandler is a QueryDataHandler wrapper that responds with cached values
type CachedQueryDatahandler struct {
	duration time.Duration
	handler  backend.QueryDataHandler
	cache    map[string]CachedResponse
}

func isExpired(val CachedResponse) bool {
	return time.Now().After(val.ExpiresAt)
}

func (c *CachedQueryDatahandler) getCachedResponse(query backend.DataQuery) (data.Frames, error) {
	key, err := getCacheKey(c.duration, query)
	if err != nil {
		return nil, err
	}

	if res, ok := c.cache[key]; ok {
		if !isExpired(res) {
			return nil, ErrorNotCached
		}

		return res.Frames, nil
	}

	return nil, ErrorNotCached
}

func (c *CachedQueryDatahandler) saveResponse(query backend.DataQuery, res data.Frames) error {
	key, err := getCacheKey(c.duration, query)
	if err != nil {
		return err
	}

	c.cache[key] = CachedResponse{
		ExpiresAt: time.Now().Add(c.duration),
		Frames:    res,
	}

	return nil
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
//
// The Frames' RefID property, when it is an empty string, will be automatically set to
// the RefID in QueryDataResponse.Responses map. This is done before the QueryDataResponse is
// sent to Grafana. Therefore one does not need to be set that property on frames when using this method.
func (c *CachedQueryDatahandler) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	var (
		cachedResponses = backend.Responses{}
		staleQueries    = []backend.DataQuery{}
	)

	for i, v := range req.Queries {
		frames, err := c.getCachedResponse(v)
		if err != nil {
			if err == ErrorNotCached {
				staleQueries = append(staleQueries, req.Queries[i])
			}
		}

		cachedResponses[v.RefID] = backend.DataResponse{
			Frames: frames,
			Error:  err,
		}
	}

	// Handle stale queries and save the results
	req.Queries = staleQueries

	response, err := c.handler.QueryData(ctx, req)
	if err != nil {
		return nil, err
	}

	for _, v := range staleQueries {
		res := response.Responses[v.RefID]
		// Only save cached values of non-error responses
		if res.Error == nil {
			if err := c.saveResponse(v, res.Frames); err != nil {
				return nil, err
			}
		}

		// Still forward response no matter what
		cachedResponses[v.RefID] = res
	}

	return &backend.QueryDataResponse{
		Responses: cachedResponses,
	}, nil
}

func (c *CachedQueryDatahandler) cleanupCache() {
	for k, v := range c.cache {
		if isExpired(v) {
			delete(c.cache, k)
		}
	}
}

// StartGC blocks the current thread and cleans up the cache key on every interval (where interval is the cache duration * 5)
func (c *CachedQueryDatahandler) StartGC(ctx context.Context) {
	ticker := time.NewTicker(c.duration * 5)
	for {
		select {
		case <-ticker.C:
			c.cleanupCache()
		case <-ctx.Done():
			return
		}
	}
}

// WithCaching wraps the provided QueryDataHandler with a cache layer
func WithCaching(ctx context.Context, handler backend.QueryDataHandler, duration time.Duration) *CachedQueryDatahandler {
	c := &CachedQueryDatahandler{
		duration: duration,
		handler:  handler,
	}

	go c.StartGC(ctx)

	return c
}
