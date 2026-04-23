package snapchat

import (
	"context"
	"errors"
	"net/http"
)

// fetchErrKind classifies failures from FetchUserStories for HTTP mapping.
type fetchErrKind int

const (
	fetchErrKindInvalidInput fetchErrKind = iota
	fetchErrKindMisconfigured
	fetchErrKindUpstream
	fetchErrKindInternal
)

// FetchError annotates errors from FetchUserStories so callers can choose status codes.
type FetchError struct {
	kind fetchErrKind
	msg  string
	err  error
}

func (e *FetchError) Error() string {
	if e == nil {
		return ""
	}
	if e.err != nil {
		return e.msg + ": " + e.err.Error()
	}
	return e.msg
}

func (e *FetchError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func newFetchError(kind fetchErrKind, msg string, err error) *FetchError {
	return &FetchError{kind: kind, msg: msg, err: err}
}

// InvalidInputError is a client-side validation failure (HTTP 400 via HTTPStatusForFetchError).
func InvalidInputError(msg string) error {
	return newFetchError(fetchErrKindInvalidInput, msg, nil)
}

// MisconfiguredError indicates missing or invalid server configuration (HTTP 503).
func MisconfiguredError(msg string) error {
	return newFetchError(fetchErrKindMisconfigured, msg, nil)
}

// UpstreamError wraps a transport or upstream-body failure (HTTP 502/503/504 via HTTPStatusForFetchError).
func UpstreamError(msg string, err error) error {
	return newFetchError(fetchErrKindUpstream, msg, err)
}

// InternalError is an unexpected failure building the outbound request (HTTP 500).
func InternalError(msg string, err error) error {
	return newFetchError(fetchErrKindInternal, msg, err)
}

// HTTPStatusForFetchError maps a FetchUserStories error to an HTTP status.
// Unknown errors default to 500.
func HTTPStatusForFetchError(err error) int {
	var fe *FetchError
	if errors.As(err, &fe) {
		switch fe.kind {
		case fetchErrKindInvalidInput:
			return http.StatusBadRequest
		case fetchErrKindMisconfigured:
			return http.StatusServiceUnavailable
		case fetchErrKindUpstream:
			if errors.Is(fe.err, context.DeadlineExceeded) {
				return http.StatusGatewayTimeout
			}
			if errors.Is(fe.err, context.Canceled) {
				return http.StatusServiceUnavailable
			}
			return http.StatusBadGateway
		case fetchErrKindInternal:
			return http.StatusInternalServerError
		default:
			return http.StatusInternalServerError
		}
	}
	return http.StatusInternalServerError
}
