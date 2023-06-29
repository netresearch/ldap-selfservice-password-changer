package rpc

import "errors"

var ErrInvalidArgumentCount = errors.New("invalid argument count")

type JSONRPC struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

type JSONRPCResponse struct {
	Success bool     `json:"success"`
	Data    []string `json:"data"`
}
