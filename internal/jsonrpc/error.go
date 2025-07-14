package JSONRPC

import (
	jsoniter "github.com/json-iterator/go"
)

var jsonSnap = jsoniter.ConfigCompatibleWithStandardLibrary

type ErrData struct {
	Msg string `json:"msg,omitempty"`
}

type JSONRPCError struct {
	Code    int      `json:"code,omitempty"`
	Message string   `json:"message,omitempty"`
	Data    *ErrData `json:"data,omitempty"`
}

func (e *JSONRPCError) AddErrMsg(er error) error {
	e.Data.Msg = er.Error()
	return e
}

func (e JSONRPCError) Error() string {
	b, _ := jsonSnap.Marshal(&e)
	return string(b)
}

func RPCError(e error) *JSONRPCError {
	if e == nil {
		return &JSONRPCError{Code: 303500, Message: "unknown error"} //ErrUnknown
	}
	er, ok := e.(JSONRPCError)
	if ok {
		return &er
	}
	if err, ok := e.(*JSONRPCError); ok{
		return err
	}
	return &JSONRPCError{Code: 303000, Message: "service error", Data: &ErrData{Msg: e.Error()}}
}

var (
	NO_DATA_ERROR = JSONRPCError{Code: 302000, Message: "no data"}
	PARAMS_ERROR  = JSONRPCError{Code: 301000, Message: "params error"}
)

func NewRPCError(code int, msg string, errData string) *JSONRPCError {
	return &JSONRPCError{Code: code, Message: msg, Data: &ErrData{Msg: errData}}
}

func NewUnknownError() JSONRPCError {
	return JSONRPCError{Code: 303500, Message: "unknown error"}
}

func NewParamsError(er error) JSONRPCError {
	return JSONRPCError{Code: 301000, Message: "params error", Data: &ErrData{Msg: er.Error()}}
}

func NewServiceError(er error) JSONRPCError {
	return JSONRPCError{Code: 303000, Message: "service error", Data: &ErrData{Msg: er.Error()}}
}

func NewNoDataError(er error) JSONRPCError {
	return JSONRPCError{Code: 302000, Message: "no data", Data: &ErrData{Msg: er.Error()}}
}

func NewErrEmptyReqId() JSONRPCError {
	return JSONRPCError{Code: 301001, Message: "no request id"}
}
