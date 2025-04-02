package data

// MarkerFrame represents an empty data frame used as a delimiter to signal the start of a new frame during chunking.
// It helps both the sender and receiver manage frame boundaries.
var MarkerFrame = NewFrame("\xE2\x97\x86") // Diamond	◆ symbol. Small (3 bytes) and easily spotted if printed

// IsMarkerFrame returns true if the given frame is a chunking marker frame.
func IsMarkerFrame(f *Frame) bool {
	return f.Rows() == 0 && f.Name == MarkerFrame.Name
}
