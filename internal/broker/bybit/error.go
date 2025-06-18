package bybit

import (
	"fmt"
)

const errorTitel = "BybitAPI"

type ErrorType string

const (
	RequestErrorT        ErrorType = "RequestError"
	ServerResponseErrorT ErrorType = "ServerResponseError"
	SerDeErrorT          ErrorType = "SerDeError"
	InternalErrorT       ErrorType = "InternalError"
	UnknownErrorT        ErrorType = "UnknownError"
)

type Error struct {
	Type     ErrorType
	Err      error
	Endpoint string
}

func NewError(t ErrorType, e error) *Error {
	return &Error{Type: t, Err: e}
}

func (e *Error) ServerResponseCode() int {
	err, ok := e.Err.(*serverResponseError)
	if !ok {
		return 0
	}

	return err.code
}

func (e *Error) SetEndpoint(endpoint string) *Error {
	newError := *e
	newError.Endpoint = endpoint

	return &newError
}

func (e *Error) Error() string {
	if e.Endpoint != "" {
		return fmt.Sprintf("%s: %s: %s: %s", errorTitel, e.Endpoint, e.Type, e.Err)
	}

	return fmt.Sprintf("%s: %s: %s", errorTitel, e.Type, e.Err)
}

type serverResponseError struct {
	msg  string
	code int
}

func ErrorFromServerResponse(r *ServerResponse) *Error {
	err := &serverResponseError{
		msg:  r.RetMsg,
		code: r.RetCode,
	}

	return NewError(ServerResponseErrorT, err)
}

func (r *serverResponseError) IsSuccess() bool {
	return r.code == 0
}

func UnwrapServerResponse(r *ServerResponse) (*ServerResponse, error) {
	if err := ErrorFromServerResponse(r).Err.(*serverResponseError); !err.IsSuccess() {
		return r, NewError(ServerResponseErrorT, err)
	}

	return r, nil
}

func (e *serverResponseError) Error() string {
	return fmt.Sprintf("%s (code %d)", e.msg, e.code)
}
