package e2e

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
)

func readRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, errors.New("response body is nil")
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return body, nil
}
