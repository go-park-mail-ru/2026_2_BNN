package grpcerr

import (
	"errors"
	"log/slog"

	"bnn/internal/models"

	"github.com/samber/lo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type knownError struct {
	err  error
	code codes.Code
}

const internalMessage = "internal error"

var knownErrors = []knownError{
	{models.ErrValidation, codes.InvalidArgument},
	{models.ErrUserExists, codes.AlreadyExists},
	{models.ErrUserNotFound, codes.NotFound},
	{models.ErrInvalidCredentials, codes.Unauthenticated},
	{models.ErrInvalidToken, codes.Unauthenticated},
	{models.ErrNoteNotFound, codes.NotFound},
	{models.ErrRefreshTokenNotFound, codes.NotFound},
	{models.ErrWrongPassword, codes.InvalidArgument},
	{models.ErrBlockNotFound, codes.NotFound},
	{models.ErrAttachmentNotFound, codes.NotFound},
	{models.ErrUnsupportedFileType, codes.InvalidArgument},
	{models.ErrTagNotFound, codes.NotFound},
	{models.ErrTagExists, codes.AlreadyExists},
	{models.ErrOAuthAccount, codes.FailedPrecondition},
	{models.ErrInvalidCode, codes.InvalidArgument},
	{models.ErrTwoFactorEnabled, codes.FailedPrecondition},
	{models.ErrForbidden, codes.PermissionDenied},
}

func ToStatus(err error) error {
	if err == nil {
		return nil
	}

	known, ok := lo.Find(knownErrors, func(k knownError) bool {
		return errors.Is(err, k.err)
	})
	if !ok {
		slog.Error("Internal error", "error", err)
		return status.Error(codes.Internal, internalMessage)
	}

	return status.Error(known.code, known.err.Error())
}

func FromStatus(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return err
	}

	known, ok := lo.Find(knownErrors, func(k knownError) bool {
		return st.Code() == k.code && st.Message() == k.err.Error()
	})
	if !ok {
		return err
	}

	return known.err
}
