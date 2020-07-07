package experimental

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// CheckGoldenDataResponse will verify that the stored file matches the given data.DataResponse
// when the updateFile flag is set, this will both add errors to the response and update the saved file
func CheckGoldenDataResponse(path string, dr *backend.DataResponse, t *testing.T, updateFile bool) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		if updateFile {
			err = writeGoldenFile(path, dr)
			if err == nil {
				t.Errorf("golden file did not exist.  creating: %s", path)
			} else {
				t.Errorf("error creating golden file:  %s / %s", path, err.Error())
			}
		} else {
			t.Errorf("missing golden file: %s", path)
		}
		return
	}

	needsUpdate := false

	saved, err := readGoldenFile(path)
	if err != nil {
		t.Errorf("error reading golden file:  %s / %s", path, err.Error())
		needsUpdate = true
	} else {
		diff := cmp.Diff(saved.Error, dr.Error)
		if diff != "" {
			t.Errorf("errors mismatch %s (-want +got):\n%s", path, diff)
			needsUpdate = true
		} else if len(saved.Frames) != len(dr.Frames) {
			t.Errorf("the number of frames returned is different:\n%s", path)
			needsUpdate = true
		} else {
			// Check each frame
			for idx, frame := range dr.Frames {
				expectedFrame := saved.Frames[idx]
				if diff := cmp.Diff(expectedFrame, frame, data.FrameTestCompareOptions()...); diff != "" {
					t.Errorf("Frame[%d] mismatch (-want +got):\n%s", idx, diff)
					needsUpdate = true
				}
			}
		}
	}

	if needsUpdate && updateFile {
		_ = writeGoldenFile(path, dr)
		t.Errorf("golden file updated: %s", path)
	}
}

const binaryDataSection = "\n====== TEST DATA RESPONSE (arrow base64) ======\n"

func readGoldenFile(path string) (*backend.DataResponse, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := string(bytes)
	idx := strings.LastIndex(text, binaryDataSection)
	if idx < 0 {
		return nil, fmt.Errorf("missing saved binary response")
	}
	lines := strings.Split(text[idx+len(binaryDataSection):], "\n")

	dr := &backend.DataResponse{}

	for _, line := range lines {
		idx = strings.Index(line, "=")
		if idx < 2 {
			continue // skip lines without KEY=VALUE
		}
		key := line[:idx]
		val := line[idx+1:]

		if key == "ERROR" {
			return nil, fmt.Errorf("error matching not yet supported: %s", line)
		}
		if key == "FRAME" {
			bytes, err = base64.StdEncoding.DecodeString(val)
			if err != nil {
				return nil, err
			}
			frame, err := data.UnmarshalArrowFrame(bytes)
			if err != nil {
				return nil, err
			}
			dr.Frames = append(dr.Frames, frame)
		}
	}
	return dr, nil
}

// When writing the golden file, we add
func writeGoldenFile(path string, dr *backend.DataResponse) error {
	str := ""
	if dr.Error != nil {
		str = fmt.Sprintf("%+v", dr.Error)
	}

	if dr.Frames != nil {
		for idx, frame := range dr.Frames {
			metaString := ""
			if frame.Meta != nil {
				frame.Meta.Custom = nil
				if frame.Meta.Stats != nil {
					frame.Meta.Stats = make([]string, 0) // avoid timing changes
				}

				meta, _ := json.MarshalIndent(frame.Meta, "", "    ")
				metaString = string(meta)
			}

			str += fmt.Sprintf("Frame[%d] %s\n", idx, metaString)

			table, _ := frame.StringTable(100, 10)
			str += table
			str += "\n\n\n"
		}
	}

	// Add the binary section flag
	str += binaryDataSection

	if dr.Error != nil {
		str += "ERROR=" + dr.Error.Error() + "\n"
	}
	for _, frame := range dr.Frames {
		bytes, _ := frame.MarshalArrow()
		encoded := base64.StdEncoding.EncodeToString(bytes)
		str += "FRAME=" + encoded + "\n"
	}

	return ioutil.WriteFile(path, []byte(str), 0600)
}
