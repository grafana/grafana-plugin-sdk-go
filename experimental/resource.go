package experimental

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// SendPlainText returns a plain text snippit
func SendPlainText(sender backend.CallResourceResponseSender, text string) error {
	SendResourceResponse(
		sender,
		200,
		map[string][]string{
			"content-type": {"text/plain"},
		},
		[]byte(text),
	)
	return nil
}

// SendJSON returns a json object
func SendJSON(sender backend.CallResourceResponseSender, obj interface{}) error {
	body, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	SendResourceResponse(
		sender,
		200,
		map[string][]string{
			"content-type": {"application/json"},
		},
		body,
	)
	return nil
}

// SendResourceResponse returns a json object
func SendResourceResponse(
	sender backend.CallResourceResponseSender,
	status int,
	headers map[string][]string,
	body []byte,
) error {
	sender.Send(&backend.CallResourceResponse{
		Status:  status,
		Headers: headers,
		Body:    body,
	})
	return nil
}
