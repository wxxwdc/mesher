package protocol

import "errors"

var (
	ErrNilResult                      = errors.New("result is nil")
	ErrUnknown                        = ProxyError{"Unknown Error,instance is not selected, error is nil"}
	ErrUnExpectedHandlerChainResponse = ProxyError{"Response from Handler chain is nil,better to check if handler chain is empty, or some handler just return a nil response"}
)

type ProxyError struct {
	Message string
}

func (e ProxyError) Error() string {
	return e.Message
}
