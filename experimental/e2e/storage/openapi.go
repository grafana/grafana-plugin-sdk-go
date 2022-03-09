package storage

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"

	"github.com/getkin/kin-openapi/routers/gorillamux"
)

// OpenAPI is a storage implementation that stores requests and responses in OpenAPI format on disk.
type OpenAPI struct {
	path   string
	spec   *openapi3.T
	router routers.Router
}

// Add is a no-op for OpenAPI storage.
func (o *OpenAPI) Add(*http.Request, *http.Response) {}

// Delete is a no-op for OpenAPI storage.
func (o *OpenAPI) Delete(string) bool {
	return true
}

// Load loads the OpenAPI specification from disk.
func (o *OpenAPI) Load() error {
	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromFile(o.path)
	if err != nil {
		return err
	}
	router, err := gorillamux.NewRouter(spec)
	if err != nil {
		return err
	}
	o.spec = spec
	o.router = router
	return nil
}

// Save is a no-op for OpenAPI storage.
func (o *OpenAPI) Save() error {
	return nil
}

// Entries is a no-op for OpenAPI storage.
func (o *OpenAPI) Entries() []*Entry {
	return []*Entry{}
}

// Match returns an example response for the given request.
func (o *OpenAPI) Match(req *http.Request) *http.Response {
	operation := o.getRoute(req)
	if operation == nil {
		return nil
	}

	status, response := o.getResponse(operation)
	if response == nil {
		return nil
	}

	res := &http.Response{
		StatusCode: status,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
	}

	for _, v := range response.Headers {
		header := v.Value
		res.Header.Set(header.Name, fmt.Sprintf("%v", header.Example))
	}

	if response.Content == nil {
		return res
	}

	content := response.Content.Get(strings.Split(res.Header.Get("Accept"), ",")[0])
	if content == nil {
		return res
	}

	return res
}

func (o *OpenAPI) getRoute(req *http.Request) *openapi3.Operation {
	route, pathParams, err := o.router.FindRoute(req)
	if err != nil {
		return nil
	}

	requestValidationInput := &openapi3filter.RequestValidationInput{
		Request:    req,
		PathParams: pathParams,
		Route:      route,
	}
	if err := openapi3filter.ValidateRequest(req.Context(), requestValidationInput); err != nil {
		return nil
	}

	return route.Operation
}

func (o *OpenAPI) getResponse(op *openapi3.Operation) (int, *openapi3.Response) {
	defaultRef := op.Responses.Default()
	fallbackStatus := http.StatusOK
	fallbackRef := defaultRef

	for k, r := range op.Responses {
		s, err := strconv.Atoi(k)
		if err != nil {
			continue
		}

		if defaultRef == r {
			return s, r.Value
		}

		if fallbackRef == nil && s < 300 {
			fallbackRef = r
			fallbackStatus = s
			break
		}
	}

	if fallbackRef != nil {
		return fallbackStatus, fallbackRef.Value
	}

	return 0, nil
}
