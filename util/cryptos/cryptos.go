package cryptos

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

// description: 使用aes加密内容，使用公钥加密aeskey
// return: reqMsg, reqKey, error
func Encrypt(content string, PUBLIC_KEY string) (string, string, error) {
	// 生成AES密钥
	aesKey := generateAESKey()

	// 使用AES加密内容
	reqMsg, err := aesEncrypt(content, aesKey)
	if err != nil {
		return "", "", fmt.Errorf("AES加密失败: %v", err)
	}

	// 使用RSA加密AES密钥
	reqKey, err := rsaEncrypt(aesKey, PUBLIC_KEY)
	if err != nil {
		return "", "", fmt.Errorf("RSA加密失败: %v", err)
	}
	return reqMsg, reqKey, nil

}

func generateAESKey() string {
	// 生成128位(16字节)的随机AES密钥
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		// 如果随机数生成失败，使用当前时间作为种子生成密钥
		timeBytes := []byte(fmt.Sprintf("%d", time.Now().UnixNano()))
		if len(timeBytes) >= 16 {
			copy(key, timeBytes[:16])
		} else {
			copy(key, timeBytes)
		}
	}
	// 返回Base64编码的密钥
	return base64.StdEncoding.EncodeToString(key)
}

// secretBase64是aeskey
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

func rsaEncrypt(secretBase64 string, PUBLIC_KEY string) (string, error) {
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

// 3-B  公钥“解密”（客户端接收时用：对端已经用私钥加密）
func RsaDecryptWithPub(pubPEM []byte, cipherB64 string) ([]byte, error) {
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

	// m = c^e mod n
	c := new(big.Int).SetBytes(cBytes)
	m := new(big.Int).Exp(c, big.NewInt(int64(pub.E)), pub.N)
	out := m.Bytes()

	k := pub.Size()
	if len(out) < k { // 前导 0 补齐
		pad := make([]byte, k-len(out))
		out = append(pad, out...)
	}
	// 去掉 PKCS#1 v1.5 Padding: 0x00 0x01 FF.. 0x00 DATA
	if out[0] != 0x00 || out[1] != 0x01 {
		return nil, errors.New("padding error")
	}
	idx := bytes.IndexByte(out[2:], 0x00)
	if idx < 8 {
		return nil, errors.New("padding error")
	}
	return out[2+idx+1:], nil
}

func AesDecryptECB(b64 string, key []byte) ([]byte, error) {
	cipherBytes, _ := base64.StdEncoding.DecodeString(b64)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(cipherBytes)%block.BlockSize() != 0 {
		return nil, errors.New("bad cipher length")
	}
	dst := make([]byte, len(cipherBytes))
	for bs, be := 0, block.BlockSize(); bs < len(cipherBytes); bs, be = bs+block.BlockSize(), be+block.BlockSize() {
		block.Decrypt(dst[bs:be], cipherBytes[bs:be])
	}
	return pkcsUnpad(dst)
}

func pkcsUnpad(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, errors.New("padding error")
	}
	pad := int(src[len(src)-1])
	if pad <= 0 || pad > len(src) {
		return nil, errors.New("padding error")
	}
	return src[:len(src)-pad], nil
}
