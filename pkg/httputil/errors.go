package httputil

import (
	"net/http"

	"github.com/rs/zerolog/hlog"
)

type HttpError struct {
	Code    int
	Err     error
	Message string
}

func (e *HttpError) Error() string {
	return e.Err.Error()
}

func (e *HttpError) Abort(w http.ResponseWriter, r *http.Request) {
	log := hlog.FromRequest(r)

	// internal errors
	if e.Code >= http.StatusInternalServerError {
		log.Error().Stack().Err(e.Err).Msg("internal error")
		http.Error(w, http.StatusText(e.Code), e.Code)
		return
	}

	// client errors
	if e.Err == nil {
		log.Warn().Str("error", e.Message).Msg("client error")
	} else {
		log.Warn().Stack().Err(e.Err).Msg("client error")
	}
	RespondWithError(w, e.Code, e.Message)
}

func NewHttpError(code int, err error, msg string) *HttpError {
	return &HttpError{Code: code, Err: err, Message: msg}
}

func NewNotFoundError() *HttpError {
	return NewHttpError(http.StatusNotFound, nil, http.StatusText(http.StatusNotFound))
}

func NewBadRequestError(err error, msg string) *HttpError {
	return NewHttpError(http.StatusBadRequest, err, msg)
}

func NewUnauthorizedError(err error) *HttpError {
	return NewHttpError(http.StatusUnauthorized, err, http.StatusText(http.StatusUnauthorized))
}

func NewForbiddenError() *HttpError {
	return NewHttpError(http.StatusForbidden, nil, http.StatusText(http.StatusForbidden))
}

func NewRequestEntityTooLargeError(err error, msg string) *HttpError {
	return NewHttpError(http.StatusRequestEntityTooLarge, err, msg)
}

func NewInternalError(err error) *HttpError {
	return NewHttpError(http.StatusInternalServerError, err, "")
}
