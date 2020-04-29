package httputil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	pkgerr "github.com/pkg/errors"
)

// robust json decoding thanks to
// https://www.alexedwards.net/blog/how-to-properly-parse-a-json-request-body

func DecodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) *HttpError {
	defer func() {
		if _, err := io.Copy(ioutil.Discard, r.Body); err != nil {
			fmt.Printf("error draining request body: %+v", err)
		}
		_ = r.Body.Close()
	}()

	mbReader := http.MaxBytesReader(w, r.Body, 1048576)
	dec := json.NewDecoder(mbReader)
	//dec.DisallowUnknownFields()
	err := dec.Decode(&dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			return NewBadRequestError(err, msg)

		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := fmt.Sprintf("request body contains badly-formed JSON")
			return NewBadRequestError(err, msg)

		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			return NewBadRequestError(err, msg)

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("request body contains unknown field %s", fieldName)
			return NewBadRequestError(err, msg)

		case errors.Is(err, io.EOF):
			msg := "request body must not be empty"
			return NewBadRequestError(err, msg)

		case err.Error() == "http: request body too large":
			msg := "request body must not be larger than 1MB"
			return NewRequestEntityTooLargeError(err, msg)

		default:
			return NewInternalError(pkgerr.Wrap(err, "error decoding JSON"))
		}
	}

	if dec.More() {
		msg := "request body must only contain a single JSON object"
		return NewBadRequestError(err, msg)
	}

	return nil
}
