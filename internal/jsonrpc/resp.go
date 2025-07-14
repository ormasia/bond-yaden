package JSONRPC

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
)

type Response struct {
	Version   string        `json:"jsonrpc"`
	RequestId string        `json:"id"`
	Data      interface{}   `json:"result,omitempty"`
	Error     *JSONRPCError `json:"error,omitempty"`
}

func OK(ctx *fiber.Ctx, data interface{}) error {
	reqid := ctx.Context().UserValue(X_REQUEST_ID)
	if reqid == nil {
		reqid = ""
	}
	_ = ctx.JSON(Response{
		Version:   JSON_RPC_VERSION,
		Data:      data,
		RequestId: reqid.(string),
	})
	return nil
}

func Error(ctx *fiber.Ctx, err error) error {
	reqid := ctx.Context().UserValue(X_REQUEST_ID)
	if reqid == nil {
		reqid = ""
	}
	e := RPCError(err)

	ctx.Set(X_RETCODE, fmt.Sprintf("%d", e.Code))
	_ = ctx.JSON(Response{
		Version:   JSON_RPC_VERSION,
		Error:     e,
		RequestId: reqid.(string),
	})
	return nil
}
