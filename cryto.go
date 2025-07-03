// file: hybrid_api_demo.go
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

const ServerPubPEM = `
-----BEGIN RSA PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCTCY4u102mtUlVEyUMlXOkflPLdWN+ez5IDcLiNzw2ZkiEY17U4lk8iMx7yTEO/ZWCKIEdQV+U6tplJ98X3I/Py/DzWd1L7IPE6mZgclfcXg+P4ocaHPsKgAodc4G1W9jTu2d6obL3d33USCD0soGYE6fkf8hk7EPKhgNf4iUPCwIDAQAB
-----END RSA PUBLIC KEY-----`

// ─────────────────────────────────────────────────────────────
// 2) AES-ECB + PKCS5Padding 封装
// func pkcsPad(src []byte, bs int) []byte {
// 	pad := bs - len(src)%bs
// 	return append(src, bytes.Repeat([]byte{byte(pad)}, pad)...)
// }

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

//	func aesEncryptECB(plain, key []byte) (string, error) {
//		block, err := aes.NewCipher(key)
//		if err != nil {
//			return "", err
//		}
//		plain = pkcsPad(plain, block.BlockSize())
//		dst := make([]byte, len(plain))
//		for bs, be := 0, block.BlockSize(); bs < len(plain); bs, be = bs+block.BlockSize(), be+block.BlockSize() {
//			block.Encrypt(dst[bs:be], plain[bs:be])
//		}
//		return base64.StdEncoding.EncodeToString(dst), nil
//	}
func aesDecryptECB(b64 string, key []byte) ([]byte, error) {
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

// ─────────────────────────────────────────────────────────────
// 3) RSA 操作
// 3-A  公钥加密（客户端发送时用）
// func rsaEncryptWithPub(pubPEM []byte, data []byte) (string, error) {
// 	block, _ := pem.Decode(pubPEM)
// 	pub, err := x509.ParsePKCS1PublicKey(block.Bytes)
// 	if err != nil {
// 		return "", err
// 	}
// 	cipherBytes, err := rsa.EncryptPKCS1v15(rand.Reader, pub, data)
// 	if err != nil {
// 		return "", err
// 	}
// 	return base64.StdEncoding.EncodeToString(cipherBytes), nil
// }

// 3-B  公钥“解密”（客户端接收时用：对端已经用私钥加密）
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

// 生成AES密钥 - 返回Base64编码的密钥字符串
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
