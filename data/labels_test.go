package data_test

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

type Labels map[string]string

// Equals returns true if the argument has the same k=v pairs as the receiver.
func TestEquals(t *testing.T) {
	a := data.Labels{"aLabelKey": "aLabelValue"}
	b := data.Labels{"bLabelKey": "bLabelValue"}
	c := data.Labels{"aLabelKey": "aLabelValue"}

	result1 := a.Equals(b)
	result2 := a.Equals(c)
	require.Equal(t, result1, false)
	require.Equal(t, result2, true)
}

func TestCopy(t *testing.T) {
	a := data.Labels{"copyLabelKey": "copyLabelValue"}
	result := a.Copy()
	require.Equal(t, result, data.Labels{"copyLabelKey": "copyLabelValue"})
}

func TestContains(t *testing.T) {
	a := data.Labels{"containsLabelKey": "containsLabelValue", "cat": "notADog"}
	result := a.Contains(data.Labels{"cat": "notADog"})
	require.Equal(t, result, true)
}

func TestString(t *testing.T) {
	a := data.Labels{"job": "prometheus", "group": "canary"}
	result := a.String()
	require.Equal(t, result, "group=canary, job=prometheus")
	b := `{group="canary", job=prometheus}`
	res, err := data.LabelsFromString(b)
	require.NoError(t, err)
	result1 := res.String()
	require.Equal(t, result1, "group=canary, job=prometheus")
}

func TestLabelsFromString(t *testing.T) {
	target := data.Labels{"group": "canary", "job": "prometheus"}

	// Support prometheus style input
	result, err := data.LabelsFromString(`{group="canary", job="prometheus"}`)
	require.NoError(t, err)
	require.Equal(t, target, result)

	// and influx style input
	result, err = data.LabelsFromString(`group=canary, job=prometheus`)
	require.NoError(t, err)
	require.Equal(t, target, result)

	// raw string
	result, err = data.LabelsFromString(`{method="GET"}`)
	require.NoError(t, err)
	require.Equal(t, result, data.Labels{"method": "GET"})
}
