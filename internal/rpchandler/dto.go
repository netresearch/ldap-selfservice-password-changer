package rpchandler

import "errors"

// ErrInvalidArgumentCount indicates that the RPC method received an incorrect number of parameters.
var ErrInvalidArgumentCount = errors.New("invalid argument count")

// Request represents a JSON-RPC 2.0 request with method name and string parameters.
type Request struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

// Response represents a JSON-RPC 2.0 response with success status and data payload.
type Response struct {
	Success bool     `json:"success"`
	Data    []string `json:"data"`
}
