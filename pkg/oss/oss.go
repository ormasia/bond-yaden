package oss

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	rpc "test/internal/jsonrpc"
	"time"

	"github.com/mitchellh/mapstructure"
)

type OssInfo struct {
	Url     string `yaml:"url"`     // OSS上传服务的URL
	Timeout int    `yaml:"timeout"` // 超时时间（秒）
}

// UploadFile 上传文件到OSS（支持业务分类、请求头、form-data参数）
// category: 业务分类（如 Open）
// filePath: 文件完整路径
// fileName: 文件名称
// md5: 文件MD5（可选，传空则不带）
// headers: map[string]string，包含 x-request-id、x-session、x-uin
// ossInfo: OSS服务配置
// 返回: OssUploadResp, error
func UploadFile(category, filePath, fileName, md5 string, headers map[string]string, ossInfo *OssInfo) (ossid, url string, err error) {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("文件不存在: %s", filePath)
	}

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 创建multipart/form-data表单
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 添加文件字段 CreateFormFile("file", fileName) 会把文件内容放到 "file" 字段，并设置文件名为 fileName。
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return "", "", fmt.Errorf("创建表单文件字段失败: %w", err)
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return "", "", fmt.Errorf("复制文件内容失败: %w", err)
	}

	// 可选参数 md5
	if md5 != "" {
		_ = writer.WriteField("md5", md5)
	}
	// 可选参数 filename
	if fileName != "" {
		_ = writer.WriteField("filename", fileName)
	}

	// 关闭writer
	err = writer.Close()
	if err != nil {
		return "", "", fmt.Errorf("关闭writer失败: %w", err)
	}

	// 拼接上传URL
	uploadUrl := fmt.Sprintf("%s/oss/v1/Upload/%s", ossInfo.Url, category)

	// 创建HTTP请求
	req, err := http.NewRequest("POST", uploadUrl, &buf)
	if err != nil {
		return "", "", fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// 设置自定义请求头
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 创建HTTP客户端，设置超时
	timeout := time.Duration(ossInfo.Timeout) * time.Second
	if ossInfo.Timeout <= 0 {
		timeout = 30 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("读取响应内容失败: %w", err)
	}

	jsonresp := rpc.Response{}

	if err := json.Unmarshal(responseBody, &jsonresp); err != nil {
		return "", "", fmt.Errorf("响应解析失败: %w, 内容: %s", err, string(responseBody))
	}
	// 检查响应状态
	if jsonresp.Error != nil {
		return "", "", fmt.Errorf("上传失败: %s, 错误码: %d", jsonresp.Error.Message, jsonresp.Error.Code)
	}

	// 解析响应
	type OssUploadResp struct {
		Ossid string `json:"ossid"`
		Url   string `json:"url"`
	}

	var ossUploadResp OssUploadResp
	if err := mapstructure.Decode(jsonresp.Data, &ossUploadResp); err != nil {
		return "", "", fmt.Errorf("响应结构体映射失败: %w", err)
	}

	// 返回响应结构体
	return ossUploadResp.Ossid, ossUploadResp.Url, nil
}
