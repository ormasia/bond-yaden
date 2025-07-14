package JSONRPC

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/mitchellh/mapstructure"
)

type Resquest struct {
	Version   string      `json:"jsonrpc"`
	Params    interface{} `json:"params"`
	RequestId string      `json:"id"`
	Method    string      `json:"method"`
}

func ParseRPCBoby(ctx *fiber.Ctx, params interface{}) {
	req := new(Resquest)
	var er error
	if er = ctx.BodyParser(req); er != nil {
		panic(NewParamsError(er))
	}
	cfg := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   params,
		TagName:  "json",
	}
	decoder, err := mapstructure.NewDecoder(cfg)
	if err != nil {
		panic(NewParamsError(err))
	}
	er = decoder.Decode(req.Params)
	if er != nil {
		panic(NewParamsError(er))
	}
	valid := validator.New()
	er = valid.Struct(params)
	if er != nil {
		panic(NewParamsError(er))
	}
}
