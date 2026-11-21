package data_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func releaseTestFrames() data.Frames {
	small := func(name string) *data.Frame {
		return data.NewFrame(name, data.NewField("v", data.Labels{"job": name}, []float64{1, 2, 3}))
	}
	large := make([]float64, 100_000)
	for i := range large {
		large[i] = float64(i)
	}
	return data.Frames{small("a"), data.NewFrame("large", data.NewField("v", nil, large)), small("b")}
}

func TestFramesMarshalArrowAndRelease(t *testing.T) {
	want, err := releaseTestFrames().MarshalArrow()
	if err != nil {
		t.Fatal(err)
	}
	frames := releaseTestFrames()
	got, err := frames.MarshalArrowAndRelease()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("MarshalArrowAndRelease() returned %d frames, want %d", len(got), len(want))
	}
	for i := range want {
		if !bytes.Equal(got[i], want[i]) {
			t.Errorf("frame %d: MarshalArrowAndRelease() bytes differ from MarshalArrow()", i)
		}
		if frames[i] != nil {
			t.Errorf("frame %d was not released", i)
		}
	}
}

func TestFramesMarshalArrowAndReleaseNilFrame(t *testing.T) {
	frames := data.Frames{data.NewFrame("a"), nil}
	if _, err := frames.MarshalArrowAndRelease(); err == nil {
		t.Fatal("MarshalArrowAndRelease() with a nil frame returned no error")
	}
	if frames[0] == nil {
		t.Error("a failed call released frame 0")
	}
}

func ExampleFrames_MarshalArrowAndRelease() {
	frames := data.Frames{data.NewFrame("a", data.NewField("v", nil, []int64{1}))}
	encoded, err := frames.MarshalArrowAndRelease()
	fmt.Println(len(encoded), frames[0] == nil, err)
	// Output: 1 true <nil>
}
