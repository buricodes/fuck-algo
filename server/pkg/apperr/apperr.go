package apperr

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

type Code string

const (
	BadRequest     Code = "bad_request"
	Unauthorised   Code = "unauthorised"
	NotFound       Code = "not_found"
	TrialExhausted Code = "trial_exhausted"
	LicenceInvalid Code = "licence_invalid"
	LicenceFull    Code = "licence_full"
	RateLimited    Code = "rate_limited"
	Upstream       Code = "upstream"
	Unavailable    Code = "unavailable"
	Internal       Code = "internal"
)

type Error struct {
	Code    Code
	Message string
	Status  int
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message, Status: statusFor(code)}
}

func Wrap(code Code, message string, err error) *Error {
	e := New(code, message)
	e.Err = err
	return e
}

func (e *Error) WithStatus(status int) *Error {
	e.Status = status
	return e
}

type UpstreamError struct {
	Provider string
	Status   int
	Body     string
	Err      error
}

func (e *UpstreamError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Provider, e.Err)
	}
	return fmt.Sprintf("%s returned %d: %s", e.Provider, e.Status, e.Body)
}

func (e *UpstreamError) Unwrap() error { return e.Err }

func From(err error) *Error {
	var appErr *Error
	var upstream *UpstreamError
	switch {
	case errors.As(err, &appErr):
		return appErr
	case errors.As(err, &upstream), errors.Is(err, context.DeadlineExceeded):
		return Wrap(Upstream, "A service we rely on did not answer properly. Try again in a moment.", err)
	}
	return Wrap(Internal, "Something went wrong on our side.", err)
}

func statusFor(code Code) int {
	switch code {
	case BadRequest, LicenceInvalid:
		return http.StatusBadRequest
	case Unauthorised:
		return http.StatusUnauthorized
	case NotFound:
		return http.StatusNotFound
	case TrialExhausted:
		return http.StatusPaymentRequired
	case LicenceFull:
		return http.StatusConflict
	case RateLimited:
		return http.StatusTooManyRequests
	case Upstream:
		return http.StatusBadGateway
	case Unavailable:
		return http.StatusServiceUnavailable
	}
	return http.StatusInternalServerError
}
