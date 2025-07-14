package dtalk

import (
	"context"
	"fmt"

	"github.com/spf13/viper"
)

// 发送文本消息
func DTalkSendTextMsg(ctx context.Context, content string) error {
	url := fmt.Sprintf("%s/robot/send", viper.GetString("dtalk.server"))
	token := viper.GetString("dtalk.accesstoken")
	secret := viper.GetString("dtalk.secret")

	if len(url) == 0 || len(token) == 0 || len(secret) == 0 {
		return fmt.Errorf("param error")
	}
	return DTalkSendTextMsgApi(ctx, url, token, secret, content)
}

// 发送文本消息
func DTalkSendMarkdownMsg(ctx context.Context, title string, content string) error {
	url := fmt.Sprintf("%s/robot/send", viper.GetString("dtalk.server"))
	token := viper.GetString("dtalk.accesstoken")
	secret := viper.GetString("dtalk.secret")

	if len(url) == 0 || len(token) == 0 || len(secret) == 0 {
		return fmt.Errorf("param error")
	}
	return DTalkSendMarkdownMsgApi(ctx, url, token, secret, title, content)
}
