package hmkv

import (
	"errors"
	"testing"
)

func TestDiscError(t *testing.T) {
	tests := []struct {
		discId int
		msg    string
		want   string
	}{
		{0, "no titles found on disc", "disc 0: no titles found on disc"},
		{2, "read error", "disc 2: read error"},
		{99, "something went wrong", "disc 99: something went wrong"},
	}

	for _, tt := range tests {
		err := NewDiscError(tt.discId, tt.msg)
		if err.Error() != tt.want {
			t.Errorf("DiscError(%d, %q).Error() = %q, want %q", tt.discId, tt.msg, err.Error(), tt.want)
		}
	}
}

func TestExternalProcessError(t *testing.T) {
	inner := errors.New("exit status 1")
	e := NewExternalProcessError(inner, "some process output")

	if e.Error() != inner.Error() {
		t.Errorf("ExternalProcessError.Error() = %q, want %q", e.Error(), inner.Error())
	}

	if e.ProcessOutput != "some process output" {
		t.Errorf("ExternalProcessError.ProcessOutput = %q, want %q", e.ProcessOutput, "some process output")
	}
}
