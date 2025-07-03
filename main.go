package main

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/go-stomp/stomp/v3"
	"github.com/gorilla/websocket"
)

// 测试环境配置
const (
	Base_URL   = "https://adenapi.cstm.adenfin.com"
	WSS_URL    = "wss://adenapi.cstm.adenfin.com/message-gateway/message/atsapi/ws"
	USERNAME   = "ATSTEST10001"
	PASSWORD   = "Abc12345"
	SMS_CODE   = "1234"
	CLIENT_ID  = "30021"
	PUBLIC_KEY = "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCTCY4u102mtUlVEyUMlXOkflPLdWN+ez5IDcLiNzw2ZkiEY17U4lk8iMx7yTEO/ZWCKIEdQV+U6tplJ98X3I/Py/DzWd1L7IPE6mZgclfcXg+P4ocaHPsKgAodc4G1W9jTu2d6obL3d33USCD0soGYE6fkf8hk7EPKhgNf4iUPCwIDAQAB"
)

// 登录请求结构
type LoginRequest struct {
	Username string `json:"username"` // 登录用户名
	Password string `json:"password"` // 登录密码
	SmsCode  string `json:"code"`     // 短信验证码
}

// 登录响应结构
type LoginResponse struct {
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	Token string `json:"data"`
}

// 登录加密请求结构，所有请求都是这个格式，Msg中需要有不同的结构
type EncryptedRequest struct {
	ReqMsg   string `json:"reqMsg"`
	ReqKey   string `json:"reqKey"`
	ClientId string `json:"clientId"`
}

type EncryptedNoLoginRequest struct {
	ReqMsg   string `json:"reqMsg"`
	ReqKey   string `json:"reqKey"`
	ClientId string `json:"clientId"`
}

type EncryptedResponse struct {
	ResMsg string `json:"resMsg"`
	ResKey string `json:"resKey"`
}

type StompClient struct {
	conn      *websocket.Conn
	stompConn *stomp.Conn
	token     string
}

func main() {
	fmt.Println("开始亚丁ATS系统测试...")

	client := &StompClient{}
	// client.token = "Lh8ksvjj7LAUYjCBUGDSQIVDx8LF707N"
	// 1. 登录获取token
	fmt.Println("第一步：登录获取Token...")
	if err := client.login(); err != nil {
		log.Fatal("登录失败:", err)
	}

	fmt.Printf("登录成功，获取到Token: %s\n", client.token[:20]+"...")

	// 2. 建立WebSocket连接
	fmt.Println("第二步：建立WebSocket连接...")
	if err := client.connectWebSocket(); err != nil {
		log.Fatal("WebSocket连接失败:", err)
	}
	defer client.conn.Close()

	// 3. 建立STOMP连接
	fmt.Println("第三步：建立STOMP连接...")
	if err := client.connectStomp(); err != nil {
		log.Fatal("STOMP连接失败:", err)
	}
	defer client.stompConn.Disconnect()

	// 4. 订阅消息
	fmt.Println("第四步：订阅行情消息...")
	if err := client.subscribe(); err != nil {
		log.Fatal("订阅失败:", err)
	}

	// 等待中断信号
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	fmt.Println("连接成功，等待消息推送...")
	fmt.Println("按 Ctrl+C 退出")

	<-interrupt
	fmt.Println("正在断开连接...")
}

// 登录获取Token
func (c *StompClient) login() error {
	// 构建登录请求
	loginReq := LoginRequest{
		Username: USERNAME,
		Password: PASSWORD,
		SmsCode:  SMS_CODE,
	}

	// 转换为JSON
	jsonData, err := json.Marshal(loginReq) // 序列化
	if err != nil {
		return fmt.Errorf("JSON序列化失败: %v", err)
	}

	// 加密请求，
	encryptedReq, err := encryptRequest(string(jsonData)) // *EncryptedRequest
	if err != nil {
		return fmt.Errorf("请求加密失败: %v", err)
	}

	// 发送HTTP请求
	reqBody, err := json.Marshal(encryptedReq)
	if err != nil {
		return fmt.Errorf("加密请求序列化失败: %v", err)
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
		return fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	fmt.Printf("响应状态: %s\n", resp.Status)
	fmt.Printf("响应内容: %s\n", string(respBody))

	var encryptedResp EncryptedResponse
	json.Unmarshal(respBody, &encryptedResp)

	pubKeyBytes, err := base64.StdEncoding.DecodeString(PUBLIC_KEY)
	if err != nil {
		return fmt.Errorf("公钥Base64解码失败: %v", err)
	}
	aesKey, err := rsaDecryptWithPub(pubKeyBytes, encryptedResp.ResKey)
	if err != nil {
		return fmt.Errorf("RSA解密AES密钥失败: %v", err)
	}

	aesKeyBase64 := string(aesKey)                                   // 转为字符串
	realAESKey, err := base64.StdEncoding.DecodeString(aesKeyBase64) // Base64解码
	if err != nil {
		return fmt.Errorf("Base64解码AES密钥失败: %v", err)
	}
	decryptedResp, err := aesDecryptECB(encryptedResp.ResMsg, realAESKey)
	if err != nil {
		return fmt.Errorf("AES解密响应失败: %v", err)
	}
	fmt.Printf("响应状态:%s", decryptedResp)

	// 解析响应
	var loginResp LoginResponse
	if err := json.Unmarshal(decryptedResp, &loginResp); err != nil {
		return fmt.Errorf("响应解析失败: %v", err)
	}

	if loginResp.Code != 200 {
		return fmt.Errorf("登录失败: %s", loginResp.Msg)
	}

	c.token = loginResp.Token
	return nil
}

// PKCS5UnPadding removes PKCS5/PKCS7 padding from decrypted data.
func PKCS5UnPadding(origData []byte) ([]byte, error) {
	length := len(origData)
	if length == 0 {
		return nil, fmt.Errorf("解密数据长度为0")
	}
	unpadding := int(origData[length-1])
	if unpadding > length || unpadding == 0 {
		return nil, fmt.Errorf("填充长度无效")
	}
	return origData[:(length - unpadding)], nil
}

// 建立WebSocket连接
func (c *StompClient) connectWebSocket() error {
	// 构建带token的URL
	u, err := url.Parse(WSS_URL)
	if err != nil {
		return fmt.Errorf("解析URL失败: %v", err)
	}

	// 添加token参数
	q := u.Query()
	q.Set("token", c.token)
	u.RawQuery = q.Encode()

	fmt.Printf("连接地址: %s\n", u.String())

	// 配置WebSocket连接
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Subprotocols:    []string{"v12.stomp", "v11.stomp", "v10.stomp"},
	}

	// 添加请求头
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+c.token)
	headers.Set("token", c.token)
	headers.Set("Origin", "https://adenapi.cstm.adenfin.com")
	headers.Set("User-Agent", "Go-WebSocket-Client/1.0")

	// 建立连接
	conn, resp, err := dialer.Dial(u.String(), headers)
	// conn, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		if resp != nil {
			fmt.Printf("HTTP响应状态: %s\n", resp.Status)
			// 打印更多响应信息
			fmt.Printf("响应头: %v\n", resp.Header)
			if resp.Body != nil {
				body, _ := io.ReadAll(resp.Body)
				fmt.Printf("响应内容: %s\n", string(body))
			}
		}
		return fmt.Errorf("WebSocket连接失败: %v", err)
	}

	fmt.Println("WebSocket连接成功!")
	c.conn = conn
	return nil
}

// 建立STOMP连接
func (c *StompClient) connectStomp() error {
	// 创建STOMP连接选项
	options := []func(*stomp.Conn) error{
		stomp.ConnOpt.Login("", ""),
		stomp.ConnOpt.Host("localhost"),
		stomp.ConnOpt.HeartBeat(30*time.Second, 10*time.Second),
		stomp.ConnOpt.Header("token", c.token),
		stomp.ConnOpt.Header("imei", "test-device-001"),
		stomp.ConnOpt.Header("appOs", "linux"),
		stomp.ConnOpt.Header("appVersion", "1.0.0"),
		stomp.ConnOpt.Header("deviceInfo", "test-client"),
	}

	// 使用WebSocket连接创建STOMP连接
	stompConn, err := stomp.Connect(NewWebSocketNetConn(c.conn), options...)
	if err != nil {
		return fmt.Errorf("STOMP连接失败: %v", err)
	}

	fmt.Println("STOMP连接成功!")
	c.stompConn = stompConn
	return nil
}

// 订阅消息
func (c *StompClient) subscribe() error {
	destination := "/user/queue/v1/apiatsbondquote/messages"

	fmt.Printf("订阅主题: %s\n", destination)

	sub, err := c.stompConn.Subscribe(destination, stomp.AckAuto)
	if err != nil {
		return fmt.Errorf("订阅失败: %v", err)
	}

	fmt.Println("订阅成功，开始监听消息...")

	// 启动消息监听协程
	go func() {
		for msg := range sub.C {
			if msg.Err != nil {
				log.Printf("消息错误: %v", msg.Err)
				continue
			}

			fmt.Println("\n========== 收到新消息 ==========")
			fmt.Printf("时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
			fmt.Printf("目的地: %s\n", msg.Destination)
			fmt.Printf("内容类型: %s\n", msg.ContentType)
			fmt.Printf("消息ID: %s\n", msg.Header.Get("message-id"))
			fmt.Printf("订阅ID: %s\n", msg.Header.Get("subscription"))
			fmt.Println("消息内容:")

			// 尝试格式化JSON输出
			var jsonData interface{}
			if err := json.Unmarshal(msg.Body, &jsonData); err == nil {
				formattedJSON, _ := json.MarshalIndent(jsonData, "", "  ")
				fmt.Println(string(formattedJSON))
			} else {
				fmt.Println(string(msg.Body))
			}
			fmt.Print("==================================\n")
		}
	}()

	return nil
}

// 加密请求
func encryptRequest(content string) (*EncryptedRequest, error) {
	// 生成AES密钥
	aesKey := generateAESKey()

	// 使用AES加密内容
	reqMsg, err := aesEncrypt(content, aesKey)
	if err != nil {
		return nil, fmt.Errorf("AES加密失败: %v", err)
	}

	// 使用RSA加密AES密钥
	reqKey, err := rsaEncrypt(aesKey)
	if err != nil {
		return nil, fmt.Errorf("RSA加密失败: %v", err)
	}

	return &EncryptedRequest{
		ReqMsg:   reqMsg,
		ReqKey:   reqKey,
		ClientId: CLIENT_ID,
	}, nil
}

// func encryptNoLoginRequest(content string) (*EncryptedNoLoginRequest, error) {
// 	// 生成AES密钥
// 	aesKey := generateAESKey()

// 	// 使用AES加密内容
// 	reqMsg, err := aesEncrypt(content, aesKey)
// 	if err != nil {
// 		return nil, fmt.Errorf("AES加密失败: %v", err)
// 	}

// 	// 使用RSA加密AES密钥
// 	reqKey, err := rsaEncrypt(aesKey)
// 	if err != nil {
// 		return nil, fmt.Errorf("RSA加密失败: %v", err)
// 	}

// 	return &EncryptedNoLoginRequest{
// 		ReqMsg: reqMsg,
// 		ReqKey: reqKey,
// 	}, nil
// }

// AES加密 - 使用ECB模式和PKCS5Padding
func aesEncrypt(data string, secretBase64 string) (string, error) {
	// 解码Base64密钥
	key, err := base64.StdEncoding.DecodeString(secretBase64)
	if err != nil {
		return "", fmt.Errorf("密钥Base64解码失败: %v", err)
	}

	// 确保输入数据使用UTF-8编码
	plaintext := []byte(data) // Go字符串默认是UTF-8编码

	// 创建AES加密器
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("创建AES加密器失败: %v", err)
	}

	// PKCS5Padding填充
	padding := aes.BlockSize - len(plaintext)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	plaintext = append(plaintext, padtext...)

	// ECB模式加密 - 不需要IV
	ciphertext := make([]byte, len(plaintext))
	for i := 0; i < len(plaintext); i += aes.BlockSize {
		block.Encrypt(ciphertext[i:i+aes.BlockSize], plaintext[i:i+aes.BlockSize])
	}

	// 返回Base64编码的结果
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// RSA加密 - 加密Base64编码的AES密钥
func rsaEncrypt(secretBase64 string) (string, error) {
	// 解析公钥
	pubKeyBytes, err := base64.StdEncoding.DecodeString(PUBLIC_KEY)
	if err != nil {
		return "", fmt.Errorf("公钥Base64解码失败: %v", err)
	}

	pubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return "", fmt.Errorf("公钥解析失败: %v", err)
	}

	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("不是有效的RSA公钥")
	}

	// 将Base64编码的密钥转换为UTF-8字节数组
	data := []byte(secretBase64)

	// RSA加密 - PKCS1v15填充方式
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPubKey, data)
	if err != nil {
		return "", fmt.Errorf("RSA加密失败: %v", err)
	}

	// 返回Base64编码的加密结果
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// WebSocket网络连接适配器，用于STOMP库
type WebSocketNetConn struct {
	conn *websocket.Conn
}

func NewWebSocketNetConn(conn *websocket.Conn) *WebSocketNetConn {
	return &WebSocketNetConn{conn: conn}
}

func (w *WebSocketNetConn) Read(p []byte) (n int, err error) {
	_, message, err := w.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	copy(p, message)
	return len(message), nil
}

func (w *WebSocketNetConn) Write(p []byte) (n int, err error) {
	err = w.conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *WebSocketNetConn) Close() error {
	return w.conn.Close()
}

func (w *WebSocketNetConn) SetDeadline(t time.Time) error {
	if err := w.conn.SetReadDeadline(t); err != nil {
		return err
	}
	return w.conn.SetWriteDeadline(t)
}

func (w *WebSocketNetConn) SetReadDeadline(t time.Time) error {
	return w.conn.SetReadDeadline(t)
}

func (w *WebSocketNetConn) SetWriteDeadline(t time.Time) error {
	return w.conn.SetWriteDeadline(t)
}
