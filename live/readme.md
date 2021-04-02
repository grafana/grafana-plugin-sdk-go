# Grafana Live API client

Usage example:

```go
package main

import (
	"context"
	"log"

	"github.com/grafana/grafana-plugin-sdk-go/live"
)

func main() {
	client, err := live.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	_, err = client.PublishStream(context.Background(), "test", []byte(`{}`))
	if err != nil {
		log.Fatal(err)
	}
}
```
