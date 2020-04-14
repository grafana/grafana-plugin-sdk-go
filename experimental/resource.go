package experimental

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// SendPlainText returns a plain text snippit
func SendPlainText(sender backend.CallResourceResponseSender, text string) error {
	return SendResourceResponse(
		sender,
		200,
		map[string][]string{
			"content-type": {"text/plain"},
		},
		[]byte(text),
	)
}

// SendJSON returns a json object
func SendJSON(sender backend.CallResourceResponseSender, obj interface{}) error {
	body, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return SendResourceResponse(
		sender,
		200,
		map[string][]string{
			"content-type": {"application/json"},
		},
		body,
	)
}

// SendResourceResponse returns a json object
func SendResourceResponse(
	sender backend.CallResourceResponseSender,
	status int,
	headers map[string][]string,
	body []byte,
) error {
	return sender.Send(&backend.CallResourceResponse{
		Status:  status,
		Headers: headers,
		Body:    body,
	})
}
