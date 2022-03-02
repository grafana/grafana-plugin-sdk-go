package data_test

import (
	"fmt"
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

func TestString( t *testing.T) {
	a := data.Labels{"job":"prometheus","group":"canary"}
	result := a.String()
	require.Equal(t, result, "group=canary, job=prometheus")
	b := data.Labels{"region":"xyz","location":"us-midwest"}
	result1 := b.String()
	require.Equal(t, result1, "location=us-midwest, region=xyz")
}

func TestLabelsFromString(t *testing.T) {
	a := data.Labels{"location":"us-midwest","region":"xyz"}
	b := a.String()
	result, err := data.LabelsFromString(b)
	if (err != nil) {
		fmt.Println(err)
	}
	require.Equal(t, result, data.Labels{"location":"us-midwest","region":"xyz"})
}