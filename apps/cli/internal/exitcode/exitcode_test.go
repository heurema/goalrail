package exitcode

import (
	"errors"
	"testing"
)

func TestForError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: Success},
		{name: "runtime default", err: errors.New("boom"), want: Runtime},
		{name: "usage", err: UsageError(errors.New("bad args")), want: Usage},
		{name: "validation", err: ValidationError(errors.New("invalid")), want: Validation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ForError(tt.err); got != tt.want {
				t.Fatalf("ForError() = %d, want %d", got, tt.want)
			}
		})
	}
}
