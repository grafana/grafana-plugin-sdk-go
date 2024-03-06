package tracing

import "testing"

func RegularFunction() string {
	return callerName(0)
}

type receiver struct{}

func (r receiver) ReceiverFunction() string {
	return callerName(0)
}

func Test_callerName(t *testing.T) {
	tests := []struct {
		name string
		f    func() string
		want string
	}{
		{
			name: "regular function",
			f:    RegularFunction,
			want: "RegularFunction",
		},
		{
			name: "receiver function",
			f:    receiver{}.ReceiverFunction,
			want: "receiver.ReceiverFunction",
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f(); got != tt.want {
				t.Errorf("callerName() = %v, want %v", got, tt.want)
			}
		})
	}
}
