package backend_test

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func ExampleNewError() {
	err := backend.NewError(backend.ErrorStatusBadRequest, "Invalid query syntax")
	fmt.Printf("An error occurred:%v", err)
}
