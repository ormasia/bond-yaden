package dtalk

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	// "lcsclient/pkg/logger"
	"net/url"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v2"
)

type DingTalkSendMsgParams struct {
	Access_token string `json:"access_token"`
	Timestamp    string `json:"timestamp"`
	Sign         string `json:"sign"`
	Secret       string `json:"secret"`
}

func (p *DingTalkSendMsgParams) GetQueryParamMap() map[string]string {
	toRet := map[string]string{}
	toRet["access_token"] = p.Access_token
	toRet["timestamp"] = p.Timestamp
	toRet["sign"] = p.Sign
	toRet["secret"] = p.Secret
	return toRet
}

type DingTalkSendMsgResp struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

type DtalkTextMsg struct {
	Content string `json:"content" validate:"required"`
}

type DtalkMarkdownMsg struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type DtalkMsg struct {
	Msgtype  string            `json:"msgtype"`
	Text     *DtalkTextMsg     `json:"text,omitempty"`
	Markdown *DtalkMarkdownMsg `json:"markdown,omitempty"`
}

// 发送文本消息
func DTalkSendMarkdownMsgApi(ctx context.Context, url string, token string, secret string, title string, text string) error {
	var err error
	r := resty.New().R().EnableTrace()

	params := &DtalkMarkdownMsg{
		Title: title,
		Text:  text,
	}

	req := &DtalkMsg{
		Msgtype:  "markdown",
		Markdown: params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	var query = &DingTalkSendMsgParams{}
	query.Access_token = token
	query.Timestamp = strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
	query.Secret = secret
	query.Sign = getDingTalkRobotSign(query)
	paramMap := query.GetQueryParamMap()

	headerMap := make(map[string]string)
	headerMap["Content-Type"] = "application/json"

	result := DingTalkSendMsgResp{}
	// URL := fmt.Sprintf("%s/robot/send", url)

	resp, err := r.SetHeaders(headerMap).SetQueryParams(paramMap).SetBody(body).SetResult(&result).Post(url)
	statusCode := resp.StatusCode()

	if err != nil {
		// logger.Z().Error("DingTalkRobotSendMsg failed", zap.Error(err))
		return err
	}

	// 正常的状态
	if statusCode != fiber.StatusOK {
		// logger.Z().Error("DingTalkRobotSendMsg failed", zap.Int("status_code", statusCode))
		return fmt.Errorf("resp statusCode:%d", statusCode)
	}

	if result.Errcode != 0 {
		// logger.Z().Error("DingTalkRobotSendMsg failed", zap.Int("error_code", result.Errcode))
		return fmt.Errorf(" resp Errcode:%d,Errmsg:%s", result.Errcode, result.Errmsg)
	}

	return nil
}

// 发送文本消息
func DTalkSendTextMsgApi(ctx context.Context, url string, token string, secret string, text string) error {
	var err error
	r := resty.New().R().EnableTrace()

	params := &DtalkTextMsg{
		Content: text,
	}

	req := &DtalkMsg{
		Msgtype: "text",
		Text:    params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	var query = &DingTalkSendMsgParams{}
	query.Access_token = token
	query.Timestamp = strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
	query.Secret = secret
	query.Sign = getDingTalkRobotSign(query)
	paramMap := query.GetQueryParamMap()

	headerMap := make(map[string]string)
	headerMap["Content-Type"] = "application/json"

	result := DingTalkSendMsgResp{}
	// URL := fmt.Sprintf("%s/robot/send", url)

	resp, err := r.SetHeaders(headerMap).SetQueryParams(paramMap).SetBody(body).SetResult(&result).Post(url)
	statusCode := resp.StatusCode()

	if err != nil {
		// logger.Z().Error("DingTalkRobotSendMsg failed", zap.Error(err))
		return err
	}

	// 正常的状态
	if statusCode != fiber.StatusOK {
		// logger.Z().Error("DingTalkRobotSendMsg failed", zap.Int("status_code", statusCode))
		return fmt.Errorf("resp statusCode:%d", statusCode)
	}

	if result.Errcode != 0 {
		// logger.Z().Error("DingTalkRobotSendMsg failed", zap.Int("error_code", result.Errcode))
		return fmt.Errorf(" resp Errcode:%d,Errmsg:%s", result.Errcode, result.Errmsg)
	}

	return nil
}

func getDingTalkRobotSign(params *DingTalkSendMsgParams) string {
	// https://developers.dingtalk.com/document/robots/customize-robot-security-settings
	// 把timestamp+"\n"+密钥当做签名字符串，
	signMsg := fmt.Sprintf("%s\n%s", params.Timestamp, params.Secret)
	// 使用HmacSHA256算法计算签名
	h := hmac.New(sha256.New, []byte(params.Secret))
	h.Write([]byte(signMsg))
	// 然后进行Base64 encode
	signMsg = base64.StdEncoding.EncodeToString(h.Sum(nil))
	// 最后再把签名参数再进行urlEncode，得到最终的签名（需要使用UTF-8字符集）
	return url.QueryEscape(signMsg)
}
