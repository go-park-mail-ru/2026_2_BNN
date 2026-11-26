package grpcerr

import (
	"errors"
	"fmt"
	"testing"

	"bnn/internal/models"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRoundTrip(t *testing.T) {
	for _, k := range knownErrors {
		t.Run(k.err.Error(), func(t *testing.T) {
			st := ToStatus(k.err)

			assert.Equal(t, k.code, status.Code(st))
			assert.ErrorIs(t, FromStatus(st), k.err)
		})
	}
}

func TestToStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected codes.Code
	}{
		{name: "nil", err: nil, expected: codes.OK},
		{name: "wrapped", err: fmt.Errorf("%w: title too long", models.ErrValidation), expected: codes.InvalidArgument},
		{name: "unknown", err: errors.New("db is down"), expected: codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, status.Code(ToStatus(tt.err)))
		})
	}
}

func TestToStatusHidesInternalError(t *testing.T) {
	st := ToStatus(errors.New(`ERROR: relation "notes" does not exist (SQLSTATE 42P01)`))

	assert.Equal(t, codes.Internal, status.Code(st))
	assert.Equal(t, "internal error", status.Convert(st).Message())
}

func TestFromStatus(t *testing.T) {
	plain := errors.New("plain")
	internal := status.Error(codes.Internal, "internal error")

	assert.NoError(t, FromStatus(nil))
	assert.Equal(t, plain, FromStatus(plain))
	assert.Equal(t, internal, FromStatus(internal))
	assert.ErrorIs(t, FromStatus(ToStatus(fmt.Errorf("%w: title too long", models.ErrValidation))), models.ErrValidation)
}
