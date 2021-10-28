package xapi

import (
	"errors"
	"fmt"
)

// JSONRPCError represents the JsonRPC2 Error message in a more golang
// friendly way rather than just a response type.
type JSONRPCError struct {
	Code    float64
	Data    interface{}
	Message string
}

func (e JSONRPCError) Error() string {
	return fmt.Sprintf("error code: %v, %v", e.Code, e.Message)
}

var (
	// ErrInvalidCredentials is returned when authentication fails.
	ErrInvalidCredentials = errors.New("missing login or password")
	// ErrMissingChannel is returned when a response channel is not found for
	// a response that comes in.  This should not happen unless a channel is removed
	// due to timeout and the message comes in late.
	ErrMissingChannel = errors.New("missing response channel for request")
	// ErrMissingIDField is when a json response from the Webex device does not have an ID field
	// and thus you can't correlate what request this response fufills.
	ErrMissingIDField = errors.New("missing id field in response")
	// ErrNotConnected happens when you try and perform client operations without calling
	// one of the connect methods.
	ErrNotConnected = errors.New("not connected")
	// ErrInvalidMsg is returned when an invalid json message type is received from the Webex device.
	ErrInvalidMsg = errors.New("invalid message")
	// ErrUnknownResponse is returned when a jsonrpc2 response comes in with an unknown data type.
	ErrUnknownResponse = errors.New("unknown response")
	// ErrUnsupportedMsg is returned when a unhandled jsonrpc2 occurs.  Currently this only happens
	// when a jsonrpc2 Request Message comes in from the server.
	ErrUnsupportedMsg = errors.New("unsupported jsonrpc2 message")
	// ErrMissingData is returned when we parse the response json struct for the jpath and
	// it returns nothing.
	ErrMissingData = errors.New("missing response data")
	// ErrMissingCallback is returned when we find a response in the callback tree but its missing
	// I don't think would ever happen in the real world.
	ErrMissingCallback = errors.New("missing callback")
)
