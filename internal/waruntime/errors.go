package waruntime

import (
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

var ErrExtensionNotFound = errors.New("extension not found")

type ClientError uint32

const (
	ClientErrorUnknown ClientError = iota
	ClientErrorNotFound
	ClientErrorAlreadyExists
	ClientErrorInvalid
	ClientErrorForbidden
	ClientErrorConflict
	ClientErrorBadRequest
	ClientErrorGone
	ClientErrorInternalError
	ClientErrorMethodNotSupported
	ClientErrorNotAcceptable
	ClientErrorEntityTooLarge
	ClientErrorResourceExpired
	ClientErrorServerTimeout
	ClientErrorServiceUnavailable
	ClientErrorTimeout
	ClientErrorTooManyRequests
	ClientErrorUnauthorized
	ClientErrorUnexpectedObject
	ClientErrorUnexpectedServerError
	ClientErrorUnsupportedMediaType
)

func toClientError(err error) ClientError {
	switch {
	case apierrors.IsAlreadyExists(err):
		return ClientErrorAlreadyExists
	case apierrors.IsNotFound(err):
		return ClientErrorNotFound
	case apierrors.IsForbidden(err):
		return ClientErrorForbidden
	case apierrors.IsInvalid(err):
		return ClientErrorInvalid
	case apierrors.IsConflict(err):
		return ClientErrorConflict
	case apierrors.IsBadRequest(err):
		return ClientErrorBadRequest
	case apierrors.IsGone(err):
		return ClientErrorGone
	case apierrors.IsInternalError(err):
		return ClientErrorInternalError
	case apierrors.IsMethodNotSupported(err):
		return ClientErrorMethodNotSupported
	case apierrors.IsNotAcceptable(err):
		return ClientErrorNotAcceptable
	case apierrors.IsRequestEntityTooLargeError(err):
		return ClientErrorEntityTooLarge
	case apierrors.IsResourceExpired(err):
		return ClientErrorResourceExpired
	case apierrors.IsServerTimeout(err):
		return ClientErrorServerTimeout
	case apierrors.IsServiceUnavailable(err):
		return ClientErrorServiceUnavailable
	case apierrors.IsTimeout(err):
		return ClientErrorTimeout
	case apierrors.IsTooManyRequests(err):
		return ClientErrorTooManyRequests
	case apierrors.IsUnauthorized(err):
		return ClientErrorUnauthorized
	case apierrors.IsUnexpectedObjectError(err):
		return ClientErrorUnexpectedObject
	case apierrors.IsUnexpectedServerError(err):
		return ClientErrorUnexpectedServerError
	case apierrors.IsUnsupportedMediaType(err):
		return ClientErrorUnsupportedMediaType
	}
	return 0
}
