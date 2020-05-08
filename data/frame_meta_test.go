package data

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONNotice(t *testing.T) {
	tests := []struct {
		name   string
		notice Notice
		json   string
	}{
		{
			name: "notice with severity and text",
			notice: Notice{
				Severity: NoticeSeverityError,
				Text:     "Some text",
			},
			json: `{"severity":"error","text":"Some text"}`,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.notice)
			require.NoError(t, err)
			require.Equal(t, tt.json, string(b))

			n := Notice{}
			err = json.Unmarshal([]byte(tt.json), &n)
			require.NoError(t, err)
			require.Equal(t, tt.notice, n)
		})
	}
}
