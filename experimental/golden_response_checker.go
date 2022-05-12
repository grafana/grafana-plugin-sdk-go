package experimental

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// CheckGoldenFramer calls CheckGoldenDataResponse using a data.Framer instead of a backend.DataResponse.
func CheckGoldenFramer(path string, f data.Framer, updateFile bool) error {
	return CheckGoldenDataResponse(path, backend.FrameResponse(f), updateFile)
}

// CheckGoldenFrame calls CheckGoldenDataResponse using a single frame
func CheckGoldenFrame(path string, f *data.Frame, updateFile bool) error {
	dr := backend.DataResponse{}
	dr.Frames = data.Frames{f}
	return CheckGoldenDataResponse(path, &dr, updateFile)
}

// CheckGoldenDataResponse will verify that the stored file matches the given data.DataResponse
// when the updateFile flag is set, this will both add errors to the response and update the saved file
func CheckGoldenDataResponse(path string, dr *backend.DataResponse, updateFile bool) error {
	saved, err := readGoldenFile(path)
	if err != nil {
		return errorAfterUpdate(fmt.Errorf("error reading golden file:  %s\n%s", path, err.Error()), path, dr, updateFile)
	}

	if err := compareResponse(path, saved, dr); err != nil {
		return errorAfterUpdate(err, path, dr, updateFile)
	}

	return nil // OK
}

// CheckGoldenJSON will verify that the stored JSON file matches the given data.DataResponse
// when the updateFile flag is set, this will both add errors to the response and update the saved file
func CheckGoldenJSON(path string, dr *backend.DataResponse, updateFile bool) error {
	saved, err := readGoldenJSONFile(path)
	if err != nil {
		return errorAfterUpdate(fmt.Errorf("error reading golden file:  %s\n%s", path, err.Error()), path, dr, updateFile)
	}

	if err := compareResponse(path, saved, dr); err != nil {
		return errorAfterUpdate(err, path, dr, updateFile)
	}

	return nil // OK
}

func compareResponse(path string, expected *backend.DataResponse, actual *backend.DataResponse) error {
	if diff := cmp.Diff(expected.Error, actual.Error); diff != "" {
		return fmt.Errorf("errors mismatch %s (-want +got):\n%s", path, diff)
	}

	// When the frame count is different, you can check manually
	if diff := cmp.Diff(len(expected.Frames), len(actual.Frames)); diff != "" {
		return fmt.Errorf("frame count mismatch (-want +got):\n%s", diff)
	}

	errorString := ""

	// Check each frame
	for idx, frame := range actual.Frames {
		expectedFrame := expected.Frames[idx]
		if diff := cmp.Diff(expectedFrame, frame, data.FrameTestCompareOptions()...); diff != "" {
			errorString += fmt.Sprintf("frame[%d] mismatch (-want +got):\n%s\n", idx, diff)
		}
	}

	if len(errorString) > 0 {
		return fmt.Errorf(errorString)
	}

	return nil // OK
}

func errorAfterUpdate(err error, path string, dr *backend.DataResponse, updateFile bool) error {
	if !updateFile {
		return err
	}
	if filepath.Ext(path) == ".txt" {
		_ = writeGoldenFile(path, dr)
	}
	if filepath.Ext(path) == ".json" {
		_ = writeGoldenJSONFile(path, dr)
	}
	log.Printf("golden file updated: %s\n", path)
	return err
}

const binaryDataSection = "====== TEST DATA RESPONSE (arrow base64) ======"

func readGoldenFile(path string) (*backend.DataResponse, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	dr := &backend.DataResponse{}

	foundDataSection := false
	scanner := bufio.NewScanner(file)
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	fsize := fi.Size()
	buf := make([]byte, 0, bufio.MaxScanTokenSize)
	scanner.Buffer(buf, int(fsize))
	for scanner.Scan() {
		line := scanner.Text()
		if foundDataSection {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue // skip lines without KEY=VALUE
			}

			key := parts[0]
			val := parts[1]

			switch key {
			case "ERROR":
				return nil, fmt.Errorf("error matching not yet supported: %s", line)
			case "FRAME":
				bytes, err := base64.StdEncoding.DecodeString(val)
				if err != nil {
					return nil, err
				}
				frame, err := data.UnmarshalArrowFrame(bytes)
				if err != nil {
					return nil, err
				}
				dr.Frames = append(dr.Frames, frame)
			default:
				return nil, fmt.Errorf("unknown saved key: %s", key)
			}
		} else if strings.HasPrefix(line, binaryDataSection) {
			foundDataSection = true
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err // error reading file
	}

	if !foundDataSection {
		return nil, fmt.Errorf("no saved result found in: %s", path)
	}

	return dr, nil
}

// The golden file has a text description at the top and a binary response at the bottom
// The text part is not used for testing, but aims to give a legible response format
func writeGoldenFile(path string, dr *backend.DataResponse) error {
	str := "🌟 This was machine generated.  Do not edit. 🌟\n"
	if dr.Error != nil {
		str = fmt.Sprintf("\nERROR: %+v", dr.Error)
	}

	if dr.Frames != nil {
		for idx, frame := range dr.Frames {
			str += fmt.Sprintf("\nFrame[%d] ", idx)
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
		str += "\nERROR=" + dr.Error.Error()
	}
	for _, frame := range dr.Frames {
		bytes, _ := frame.MarshalArrow()
		encoded := base64.StdEncoding.EncodeToString(bytes)
		str += "\nFRAME=" + encoded
	}
	str += "\n"

	return ioutil.WriteFile(path, []byte(str), 0600)
}

func readGoldenJSONFile(fpath string) (*backend.DataResponse, error) {
	raw, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}
	var dr backend.DataResponse
	if err = json.Unmarshal(raw, &dr); err != nil {
		return nil, err
	}
	return &dr, nil
}

func writeGoldenJSONFile(fpath string, dr *backend.DataResponse) error {
	str, err := json.MarshalIndent(dr, "", "    ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fpath, []byte(str), 0600)
}
