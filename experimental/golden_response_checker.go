package experimental

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// CheckGoldenDataResponse will verify that the stored file matches the given data.DataResponse
// when the updateFile flag is set, this will both add errors to the response and update the saved file
func CheckGoldenDataResponse(path string, dr *backend.DataResponse, updateFile bool) error {
	saved, err := readGoldenFile(path)
	if err != nil {
		err = fmt.Errorf("error reading golden file:  %s\n%s", path, err.Error())
	} else {
		if diff := cmp.Diff(saved.Error, dr.Error); diff != "" {
			err = fmt.Errorf("errors mismatch %s (-want +got):\n%s", path, diff)
		}
		if diff := cmp.Diff(len(saved.Frames), len(dr.Frames)); diff != "" {
			err = fmt.Errorf("Frame count mismatch (-want +got):\n%s", diff)
		} else {
			// Check each frame
			for idx, frame := range dr.Frames {
				expectedFrame := saved.Frames[idx]
				if diff := cmp.Diff(expectedFrame, frame, data.FrameTestCompareOptions()...); diff != "" {
					err = fmt.Errorf("Frame[%d] mismatch (-want +got):\n%s", idx, diff)
				}
			}
		}
	}

	if err != nil && updateFile {
		_ = writeGoldenFile(path, dr)
		log.Printf("golden file updated: %s\n", path)
	}
	return err
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

// The golden file has a text description at the top and a binary response at the bottom
// The text part is not used for testing, but aims to give a legible response format
func writeGoldenFile(path string, dr *backend.DataResponse) error {
	str := ""
	if dr.Error != nil {
		str = fmt.Sprintf("%+v", dr.Error)
	}

	if dr.Frames != nil {
		for idx, frame := range dr.Frames {
			str += fmt.Sprintf("Frame[%d] ", idx)
			if frame.Meta != nil {
				meta, _ := json.MarshalIndent(frame.Meta, "", "    ")
				str += string(meta)
			}

			table, _ := frame.StringTable(100, 10)
			str += "\n" + table + "\n\n"
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
