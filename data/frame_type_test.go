package data

import "testing"

func TestFrameTypeVersionComparison(t *testing.T) {
	tests := []struct {
		name    string
		version FrameTypeVersion
		other   FrameTypeVersion
		less    bool
		greater bool
	}{
		{
			name:    "equal",
			version: FrameTypeVersion{1, 2},
			other:   FrameTypeVersion{1, 2},
		},
		{
			name:    "lower minor",
			version: FrameTypeVersion{1, 1},
			other:   FrameTypeVersion{1, 2},
			less:    true,
		},
		{
			name:    "higher minor",
			version: FrameTypeVersion{1, 2},
			other:   FrameTypeVersion{1, 1},
			greater: true,
		},
		{
			name:    "lower major with higher minor",
			version: FrameTypeVersion{1, 10},
			other:   FrameTypeVersion{2, 0},
			less:    true,
		},
		{
			name:    "higher major with lower minor",
			version: FrameTypeVersion{2, 0},
			other:   FrameTypeVersion{1, 10},
			greater: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.version.Less(tt.other); got != tt.less {
				t.Errorf("Less() = %v, want %v", got, tt.less)
			}
			if got := tt.version.Greater(tt.other); got != tt.greater {
				t.Errorf("Greater() = %v, want %v", got, tt.greater)
			}
		})
	}
}
