package service

// 亚丁ATS系统客户端
// 实现与亚丁ATS（Automated Trading System）系统的连接和通信
//
// 主要功能：
// 1. 用户登录认证（混合加密）
// 2. WebSocket连接建立
// 3. STOMP协议通信
// 4. 实时债券行情数据接收
//
// 技术特点：
// - RSA+AES混合加密通信
// - WebSocket+STOMP双协议栈
// - 实时消息推送
// - 优雅的连接管理

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
	utils "wealth-bond-quote-service/pkg/crypto_utils"

	logger "wealth-bond-quote-service/pkg/log"

	"github.com/go-stomp/stomp/v3"
	"github.com/gorilla/websocket"
	_ "modernc.org/sqlite"

	"github.com/google/uuid"
)

// LoginRequest 登录请求结构体
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	SmsCode  string `json:"code"`
}

// LoginResponse 登录响应结构体
type LoginResponse struct {
	Code  int    `json:"code"` // 响应状态码，200表示成功
	Msg   string `json:"msg"`  // 响应消息，成功或错误描述
	Token string `json:"data"` // 登录成功后返回的访问令牌，用于后续API调用
}

// EncryptedRequest 加密请求结构体
type EncryptedRequest struct {
	ReqMsg   string `json:"reqMsg"`   // AES加密后的请求内容（Base64编码）
	ReqKey   string `json:"reqKey"`   // RSA加密后的AES密钥（Base64编码）
	ClientId string `json:"clientId"` // 客户端标识符，用于区分不同客户端
}

// EncryptedResponse 服务器返回的加密响应格式
type EncryptedResponse struct {
	ResMsg string `json:"resMsg"` // AES加密后的响应内容（Base64编码）
	ResKey string `json:"resKey"` // RSA加密后的AES密钥（Base64编码），需要用公钥"解密"
}

// StompClient STOMP客户端结构体
// 封装WebSocket连接和STOMP协议连接，用于接收实时消息推送
type StompClient struct {
	Conn      *websocket.Conn // WebSocket底层连接
	StompConn *stomp.Conn     // STOMP协议连接，基于WebSocket
	Token     string          // 访问令牌，用于身份验证
}

// 登录获取Token
func (c *StompClient) Login(username, password, smsCode, publicKey, baseURL, clientID string) error {
	// 构建登录请求
	loginReq := LoginRequest{
		Username: username,
		Password: password,
		SmsCode:  smsCode,
	}

	// 转换为JSON
	jsonData, err := json.Marshal(loginReq) // 序列化
	if err != nil {
		return fmt.Errorf("JSON序列化失败: %v", err)
	}

	// 加密请求
	var encryptedReq EncryptedRequest
	msg, key, err := utils.EncryptRequest(string(jsonData), publicKey, clientID)
	if err != nil {
		return fmt.Errorf("请求加密失败: %v", err)
	}
	encryptedReq.ReqMsg = msg
	encryptedReq.ReqKey = key
	encryptedReq.ClientId = clientID

	// 发送HTTP请求
	reqBody, err := json.Marshal(encryptedReq)
	if err != nil {
		return fmt.Errorf("加密请求序列化失败: %v", err)
	}

	// 构建登录API的完整URL
	// 路径: /cust-gateway/cust-auth/account/outApi/doLogin
	LOGIN_URL := fmt.Sprintf("%s%s", baseURL, "/cust-gateway/cust-auth/account/outApi/doLogin")
	fmt.Printf("发送登录请求到: %s\n", LOGIN_URL)

	// 创建HTTP客户端，配置超时和TLS设置
	client := &http.Client{
		Timeout: 30 * time.Second, // 请求超时时间30秒
		Transport: &http.Transport{
			// TLS配置：跳过证书验证（仅用于测试环境）
			// 生产环境应该移除InsecureSkipVerify或设置为false
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Post(LOGIN_URL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	var encryptedResp EncryptedResponse
	json.Unmarshal(respBody, &encryptedResp)

	pubKeyBytes, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return fmt.Errorf("公钥Base64解码失败: %v", err)
	}
	aesKey, err := utils.RsaDecryptWithPub(pubKeyBytes, encryptedResp.ResKey)
	if err != nil {
		return fmt.Errorf("RSA解密AES密钥失败: %v", err)
	}

	aesKeyBase64 := string(aesKey)                                   // 转为字符串
	realAESKey, err := base64.StdEncoding.DecodeString(aesKeyBase64) // Base64解码
	if err != nil {
		return fmt.Errorf("Base64解码AES密钥失败: %v", err)
	}
	decryptedResp, err := utils.AesDecryptECB(encryptedResp.ResMsg, realAESKey)
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

	c.Token = loginResp.Token
	return nil
}

// 建立WebSocket连接
func (c *StompClient) ConnectWebSocket(wssURL string) error {
	// 构建带token的URL
	u, err := url.Parse(wssURL)
	if err != nil {
		return fmt.Errorf("解析URL失败: %v", err)
	}

	// 添加token参数
	q := u.Query()
	q.Set("token", "Bearer "+c.Token)
	u.RawQuery = q.Encode()

	fmt.Printf("连接地址: %s\n", u.String())

	// 配置WebSocket拨号器
	dialer := websocket.Dialer{
		// TLS配置：跳过证书验证（测试环境）
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		// 支持的STOMP协议版本，按优先级排序
		// v12.stomp: STOMP 1.2版本（最新）
		// v11.stomp: STOMP 1.1版本
		// v10.stomp: STOMP 1.0版本（向后兼容）
		Subprotocols: []string{"v12.stomp", "v11.stomp", "v10.stomp"},
	}

	// 设置WebSocket连接的HTTP请求头
	headers := http.Header{}
	// 标准的Bearer Token认证头
	// headers.Set("Authorization", "Bearer "+c.token)
	// 自定义token头（服务器可能需要）
	headers.Set("token", c.Token)
	// 用户代理标识
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
	c.Conn = conn
	return nil
}

// 建立STOMP连接
func (c *StompClient) ConnectStomp() error {
	// 创建STOMP连接选项配置
	options := []func(*stomp.Conn) error{
		// 登录凭据（空用户名密码，使用token认证）
		stomp.ConnOpt.Login("", ""),
		// 虚拟主机名（STOMP协议要求）
		stomp.ConnOpt.Host("localhost"),
		// 心跳配置：发送心跳间隔30秒，接收心跳超时120秒
		// 用于保持连接活跃和检测连接状态
		stomp.ConnOpt.HeartBeat(30*time.Second, 120*time.Second),
		// 自定义STOMP头部信息
		stomp.ConnOpt.Header("token", c.Token),            // 访问令牌
		stomp.ConnOpt.Header("imei", "test-device-001"),   // 设备IMEI标识
		stomp.ConnOpt.Header("appOs", "windows"),          // 应用运行的操作系统
		stomp.ConnOpt.Header("appVersion", "1.0.0"),       // 应用版本号
		stomp.ConnOpt.Header("deviceInfo", "test-client"), // 设备信息描述
	}

	// 使用WebSocket连接创建STOMP连接
	stompConn, err := stomp.Connect(NewWebSocketNetConn(c.Conn), options...)
	if err != nil {
		return fmt.Errorf("STOMP连接失败: %v", err)
	}

	fmt.Println("STOMP连接成功!")
	c.StompConn = stompConn
	return nil
}

// 订阅消息
func (c *StompClient) Subscribe(ctx context.Context, rawChan chan []byte, errChan chan error) error {
	// 订阅目标地址：债券行情消息队列
	// /user/queue/v1/apiatsbondquote/messages
	// - /user: 用户专用队列前缀
	// - /queue: 队列类型（点对点消息）
	// - v1: API版本
	// - apiatsbondquote: 债券行情API标识
	// - messages: 消息主题
	destination := "/user/queue/v1/apiatsbondquote/messages"

	fmt.Printf("订阅主题: %s\n", destination)

	// 订阅消息，使用自动确认模式
	// stomp.AckAuto: 消息接收后自动确认，无需手动ACK
	subcribeId := uuid.New().String()
	// 订阅消息，并添加自定义消息头
	sub, err := c.StompConn.Subscribe(
		destination,
		stomp.AckAuto,
		stomp.SubscribeOpt.Header("uuid", subcribeId),               // 客户端标识符
		stomp.SubscribeOpt.Header("id", subcribeId),                 // 客户端标识符
		stomp.SubscribeOpt.Header("receipt", "receipt-"+subcribeId), // 请求回执
	)

	if err != nil {
		return fmt.Errorf("订阅失败: %v", err)
	}

	fmt.Println("订阅成功，开始监听消息...")

	for {
		select {
		case <-ctx.Done():
			// 上下文取消，退出监听
			fmt.Println("上下文取消，停止消息监听")
			return nil
		case msg := <-sub.C:
			if msg.Err != nil {
				logger.Error("消息错误: %v", msg.Err)
				errChan <- msg.Err
				return nil
			}

			// 将rawjson发送到RawChan通道
			if len(msg.Body) != 0 {
				rawChan <- msg.Body
			}

			// // 尝试格式化JSON输出
			// var jsonData any
			// if err := json.Unmarshal(msg.Body, &jsonData); err == nil {
			// 	formattedJSON, _ := json.MarshalIndent(jsonData, "", "  ")
			// 	fmt.Println(string(formattedJSON))
			// } else {
			// 	fmt.Println(string(msg.Body))
			// }
			// fmt.Print("==================================\n")
		}
	}
}

// WebSocketNetConn WebSocket网络连接适配器
// 将WebSocket连接包装成net.Conn接口，供STOMP库使用
// STOMP库需要net.Conn接口，而WebSocket连接需要适配
type WebSocketNetConn struct {
	conn *websocket.Conn // 底层WebSocket连接
}

// NewWebSocketNetConn 创建WebSocket网络连接适配器
// 参数：
//   - conn: WebSocket连接实例
//
// 返回：适配器实例
func NewWebSocketNetConn(conn *websocket.Conn) *WebSocketNetConn {
	return &WebSocketNetConn{conn: conn}
}

// Read 实现net.Conn接口的Read方法
// 从WebSocket连接读取数据
func (w *WebSocketNetConn) Read(p []byte) (n int, err error) {
	// 读取WebSocket消息（忽略消息类型）
	_, message, err := w.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	// if err == nil && len(message) > 0 {
	// 	fmt.Printf("[%s] ", time.Now().Format("15:04:05.000"))
	// 	fmt.Println("收到STOMP帧:")
	// 	fmt.Println(string(message))
	// 	fmt.Println("----------------------------")
	// }
	// 将消息内容复制到缓冲区
	copy(p, message)
	return len(message), nil
}

// Write 实现net.Conn接口的Write方法
// 向WebSocket连接写入数据
func (w *WebSocketNetConn) Write(p []byte) (n int, err error) {
	// if len(p) > 0 {
	// 	fmt.Printf("[%s] ", time.Now().Format("15:04:05.000"))
	// 	fmt.Println("发送STOMP帧:")
	// 	fmt.Println(string(p))
	// 	fmt.Println("----------------------------")
	// }
	// 发送文本消息到WebSocket
	err = w.conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Close 实现net.Conn接口的Close方法
// 关闭WebSocket连接
func (w *WebSocketNetConn) Close() error {
	return w.conn.Close()
}

// SetDeadline 实现net.Conn接口的SetDeadline方法
// 设置读写超时时间
func (w *WebSocketNetConn) SetDeadline(t time.Time) error {
	// 同时设置读和写的超时时间
	if err := w.conn.SetReadDeadline(t); err != nil {
		return err
	}
	return w.conn.SetWriteDeadline(t)
}

// SetReadDeadline 实现net.Conn接口的SetReadDeadline方法
// 设置读取超时时间
func (w *WebSocketNetConn) SetReadDeadline(t time.Time) error {
	return w.conn.SetReadDeadline(t)
}

// SetWriteDeadline 实现net.Conn接口的SetWriteDeadline方法
// 设置写入超时时间
func (w *WebSocketNetConn) SetWriteDeadline(t time.Time) error {
	return w.conn.SetWriteDeadline(t)
}

func (w *WebSocketNetConn) LocalAddr() net.Addr {
	return w.conn.LocalAddr()
}

func (w *WebSocketNetConn) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}
