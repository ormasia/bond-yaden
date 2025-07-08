// 加密解密工具包
// 实现亚丁ATS系统的加密通信协议
// 包含AES对称加密和RSA非对称加密的实现
package main

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"time"
)

// pkcsUnpad PKCS填充去除函数
// 移除PKCS#5/PKCS#7填充，恢复原始数据长度
func pkcsUnpad(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, errors.New("padding error")
	}
	// 获取填充长度（最后一个字节的值）
	pad := int(src[len(src)-1])
	if pad <= 0 || pad > len(src) {
		return nil, errors.New("padding error")
	}
	// 返回去除填充后的数据
	return src[:len(src)-pad], nil
}

// aesDecryptECB AES-ECB模式解密函数
// 用于解密服务器返回的加密响应内容
// 参数：
//   - b64: Base64编码的密文
//   - key: AES解密密钥（16字节）
//
// 返回：解密后的明文数据
func aesDecryptECB(b64 string, key []byte) ([]byte, error) {
	// Base64解码密文
	cipherBytes, _ := base64.StdEncoding.DecodeString(b64)

	// 创建AES解密器
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 检查密文长度是否为块大小的整数倍
	if len(cipherBytes)%block.BlockSize() != 0 {
		return nil, errors.New("bad cipher length")
	}

	// ECB模式逐块解密
	dst := make([]byte, len(cipherBytes))
	for bs, be := 0, block.BlockSize(); bs < len(cipherBytes); bs, be = bs+block.BlockSize(), be+block.BlockSize() {
		block.Decrypt(dst[bs:be], cipherBytes[bs:be])
	}

	// 去除PKCS填充
	return pkcsUnpad(dst)
}

// 公钥“解密”
func rsaDecryptWithPub(pubPEM []byte, cipherB64 string) ([]byte, error) {
	var pub *rsa.PublicKey
	var err error

	// 尝试解析PEM格式
	if block, _ := pem.Decode(pubPEM); block != nil {
		// 先尝试PKCS1格式
		pub, err = x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			// 如果PKCS1失败，尝试PKIX格式
			pubKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				return nil, err
			}
			var ok bool
			pub, ok = pubKeyInterface.(*rsa.PublicKey)
			if !ok {
				return nil, errors.New("not a valid RSA public key")
			}
		}
	} else {
		// 如果不是PEM格式，尝试直接解析Base64编码的PKIX格式
		pubKeyInterface, err := x509.ParsePKIXPublicKey(pubPEM)
		if err != nil {
			return nil, err
		}
		var ok bool
		pub, ok = pubKeyInterface.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("not a valid RSA public key")
		}
	}

	cBytes, _ := base64.StdEncoding.DecodeString(cipherB64)

	// RSA公钥运算：m = c^e mod n
	// 这里执行的是RSA验证操作，用公钥"解密"私钥"加密"的数据
	c := new(big.Int).SetBytes(cBytes)
	m := new(big.Int).Exp(c, big.NewInt(int64(pub.E)), pub.N)
	out := m.Bytes()

	// 补齐前导零到密钥长度
	k := pub.Size()
	if len(out) < k {
		pad := make([]byte, k-len(out))
		out = append(pad, out...)
	}

	// 去除PKCS#1 v1.5填充格式：0x00 0x01 FF...FF 0x00 DATA
	if out[0] != 0x00 || out[1] != 0x01 {
		return nil, errors.New("padding error")
	}
	// 查找填充结束标记0x00
	idx := bytes.IndexByte(out[2:], 0x00)
	if idx < 8 { // 填充至少8个0xFF字节
		return nil, errors.New("padding error")
	}
	// 返回实际数据部分
	return out[2+idx+1:], nil
}

// generateAESKey 生成AES密钥函数
// 生成用于加密请求内容的随机AES密钥
// 返回：Base64编码的AES密钥字符串
func generateAESKey() string {
	// 生成128位(16字节)的随机AES密钥
	// AES-128提供足够的安全性，且性能较好
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		// 如果随机数生成失败，使用当前时间作为种子生成密钥
		// 这是一个备用方案，安全性较低，仅用于异常情况
		timeBytes := fmt.Appendf(nil, "%d", time.Now().UnixNano())
		if len(timeBytes) >= 16 {
			copy(key, timeBytes[:16])
		} else {
			copy(key, timeBytes)
		}
	}
	// 返回Base64编码的密钥，便于传输和存储
	return base64.StdEncoding.EncodeToString(key)
}

// aesEncrypt AES加密函数
// 使用AES-ECB模式和PKCS#5填充对数据进行加密
// 参数：
//   - data: 待加密的明文数据
//   - secretBase64: Base64编码的AES密钥
//
// 返回：Base64编码的密文
func aesEncrypt(data string, secretBase64 string) (string, error) {
	// 解码Base64编码的AES密钥
	key, err := base64.StdEncoding.DecodeString(secretBase64)
	if err != nil {
		return "", fmt.Errorf("密钥Base64解码失败: %v", err)
	}

	// 将字符串转换为UTF-8字节数组
	// Go字符串默认使用UTF-8编码
	plaintext := []byte(data)

	// 创建AES加密器（支持128/192/256位密钥）
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("创建AES加密器失败: %v", err)
	}

	// PKCS#5填充（实际上是PKCS#7填充的特例）
	// 计算需要填充的字节数
	padding := aes.BlockSize - len(plaintext)%aes.BlockSize
	// 创建填充数据，每个填充字节的值等于填充长度
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	plaintext = append(plaintext, padtext...)

	// ECB模式加密 - 逐块加密，不需要初始化向量(IV)
	// 注意：ECB模式安全性较低，但服务器要求使用此模式
	ciphertext := make([]byte, len(plaintext))
	for i := 0; i < len(plaintext); i += aes.BlockSize {
		block.Encrypt(ciphertext[i:i+aes.BlockSize], plaintext[i:i+aes.BlockSize])
	}

	// 返回Base64编码的加密结果
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// rsaEncrypt RSA加密函数
// 使用RSA公钥加密Base64编码的AES密钥
// 参数：
//   - secretBase64: Base64编码的AES密钥字符串
//
// 返回：Base64编码的RSA加密结果
func rsaEncrypt(plaintext, publicKey string) (string, error) {
	// 解码Base64编码的公钥
	pubKeyBytes, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return "", fmt.Errorf("公钥Base64解码失败: %v", err)
	}

	// 解析PKIX格式的公钥
	pubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return "", fmt.Errorf("公钥解析失败: %v", err)
	}

	// 类型断言，确保是RSA公钥
	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("不是有效的RSA公钥")
	}

	// 将Base64编码的AES密钥转换为字节数组
	// 这里加密的是AES密钥的Base64字符串，而不是原始密钥字节
	data := []byte(plaintext)

	// 使用RSA公钥加密，采用PKCS#1 v1.5填充方式
	// 这是经典的RSA加密方式，兼容性好
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPubKey, data)
	if err != nil {
		return "", fmt.Errorf("RSA加密失败: %v", err)
	}

	// 返回Base64编码的RSA加密结果
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// encryptRequest 加密请求函数
// 实现亚丁ATS系统的混合加密方案：
// 1. 生成随机AES密钥
// 2. 用AES密钥加密请求内容
// 3. 用RSA公钥加密AES密钥
// 4. 组装成加密请求结构
// 参数：
//   - content: 待加密的请求内容（JSON字符串）
//
// 返回：加密后的请求结构体
func encryptRequest(content string, publicKey, clientID string) (*EncryptedRequest, error) {
	// 第一步：生成随机AES密钥（Base64编码）
	aesKey := generateAESKey()

	// 第二步：使用AES密钥加密请求内容
	// 采用AES-ECB模式和PKCS#5填充
	reqMsg, err := aesEncrypt(content, aesKey)
	if err != nil {
		return nil, fmt.Errorf("AES加密失败: %v", err)
	}

	// 第三步：使用RSA公钥加密AES密钥
	// 确保AES密钥的安全传输
	reqKey, err := rsaEncrypt(aesKey, publicKey)
	if err != nil {
		return nil, fmt.Errorf("RSA加密失败: %v", err)
	}

	// 第四步：组装加密请求结构
	return &EncryptedRequest{
		ReqMsg:   reqMsg,   // AES加密后的请求内容
		ReqKey:   reqKey,   // RSA加密后的AES密钥
		ClientId: clientID, // 客户端标识符
	}, nil
}
