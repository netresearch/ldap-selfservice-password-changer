package rpc

import "errors"

// ErrInvalidArgumentCount indicates that the RPC method received an incorrect number of parameters.
var ErrInvalidArgumentCount = errors.New("invalid argument count")

// JSONRPC represents a JSON-RPC 2.0 request with method name and string parameters.
type JSONRPC struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response with success status and data payload.
type JSONRPCResponse struct {
	Success bool     `json:"success"`
	Data    []string `json:"data"`
}
