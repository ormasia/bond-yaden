package conn

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"test/util/cryptos"
	"time"
)

// type StompClient struct {
// 	conn      *websocket.Conn
// 	stompConn *stomp.Conn
// 	token     string
// }

// 登录请求结构
type LoginRequest struct {
	Username string `json:"username"` // 登录用户名
	Password string `json:"password"` // 登录密码
	SmsCode  string `json:"code"`     // 短信验证码
}

type EncryptedRequest struct {
	ReqMsg   string `json:"reqMsg"`
	ReqKey   string `json:"reqKey"`
	ClientId string `json:"clientId"`
}

type EncryptedResponse struct {
	ResMsg string `json:"resMsg"`
	ResKey string `json:"resKey"`
}

type LoginResponse struct {
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	Token string `json:"data"`
}

// 登录获取Token
func Login(username, password, smsCode, PUBLIC_KEY, Base_URL string) (string, error) {
	// 构建登录请求
	loginReq := LoginRequest{
		Username: username,
		Password: password,
		SmsCode:  smsCode,
	}

	// 转换为JSON
	jsonData, err := json.Marshal(loginReq) // 序列化
	if err != nil {
		return "", fmt.Errorf("JSON序列化失败: %v", err)
	}

	// 加密请求
	reqMsg, reqKey, err := cryptos.Encrypt(string(jsonData), PUBLIC_KEY)
	if err != nil {
		return "", fmt.Errorf("请求加密失败: %v", err)
	}

	encryptedReq := EncryptedRequest{
		ReqMsg:   reqMsg,
		ReqKey:   reqKey,
		ClientId: "123456",
	}

	// 发送HTTP请求
	reqBody, err := json.Marshal(encryptedReq)
	if err != nil {
		return "", fmt.Errorf("加密请求序列化失败: %v", err)
	}

	LOGIN_URL := fmt.Sprintf("%s%s", Base_URL, "/cust-gateway/cust-auth/account/outApi/doLogin")
	fmt.Printf("发送登录请求到: %s\n", LOGIN_URL)

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // 忽略证书验证
		},
	}

	resp, err := client.Post(LOGIN_URL, "application/json", bytes.NewBuffer(reqBody)) //
	if err != nil {
		return "", fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	fmt.Printf("响应状态: %s\n", resp.Status)
	fmt.Printf("响应内容: %s\n", string(respBody))

	var encryptedResp EncryptedResponse
	json.Unmarshal(respBody, &encryptedResp)
	fmt.Printf("respBody反序列化后:%s", encryptedResp)

	pubKeyBytes, err := base64.StdEncoding.DecodeString(PUBLIC_KEY)
	if err != nil {
		return "", fmt.Errorf("公钥Base64解码失败: %v", err)
	}
	aesKey, err := cryptos.RsaDecryptWithPub(pubKeyBytes, encryptedResp.ResKey)
	if err != nil {
		return "", fmt.Errorf("RSA解密AES密钥失败: %v", err)
	}

	aesKeyBase64 := string(aesKey)                                   // 转为字符串
	realAESKey, err := base64.StdEncoding.DecodeString(aesKeyBase64) // Base64解码
	if err != nil {
		return "", fmt.Errorf("Base64解码AES密钥失败: %v", err)
	}
	decryptedResp, err := cryptos.AesDecryptECB(encryptedResp.ResMsg, realAESKey)
	if err != nil {
		return "", fmt.Errorf("AES解密响应失败: %v", err)
	}
	fmt.Printf("decryptedResp:%s", decryptedResp)

	// 解析响应
	var loginResp LoginResponse
	if err := json.Unmarshal(decryptedResp, &loginResp); err != nil {
		return "", fmt.Errorf("响应解析失败: %v", err)
	}

	if loginResp.Code != 200 {
		return "", fmt.Errorf("登录失败: %s", loginResp.Msg)
	}

	// c.token = loginResp.Token
	return loginResp.Token, nil
}
