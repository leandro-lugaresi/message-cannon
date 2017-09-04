package worker

import (
	"errors"
	"testing"
)

func TestMultiError_Error(t *testing.T) {
	tests := []struct {
		name string
		m    MultiError
		want string
	}{
		{
			"With only one error",
			MultiError{nil, errors.New("Error example 1")},
			"Error example 1",
		},
		{
			"With Many errors",
			MultiError{errors.New("Error example 1"), errors.New("Error example 3"), nil},
			"Multi errors: \n\t - Error example 1\n\t - Error example 3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.Error(); got != tt.want {
				t.Errorf("MultiError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}
