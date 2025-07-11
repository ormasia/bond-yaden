package oss

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

type OssInfo struct {
	Url     string `yaml:"url"`     // OSS上传服务的URL
	Timeout int    `yaml:"timeout"` // 超时时间（秒）
}

// UploadFile 上传文件到OSS
// filePath: 文件完整路径
// fileName: 文件名称
// ossInfo: OSS服务配置
// 返回: OSS服务响应（通常是OSS ID或文件标识符）, error
func UploadFile(filePath, fileName string, ossInfo *OssInfo) (string, error) {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("文件不存在: %s", filePath)
	}

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 创建multipart表单
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 添加文件字段
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return "", fmt.Errorf("创建表单文件字段失败: %w", err)
	}

	// 复制文件内容到表单
	_, err = io.Copy(part, file)
	if err != nil {
		return "", fmt.Errorf("复制文件内容失败: %w", err)
	}

	// 关闭writer
	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("关闭writer失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", ossInfo.Url, &buf)
	if err != nil {
		return "", fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置Content-Type
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 创建HTTP客户端，设置超时
	timeout := time.Duration(ossInfo.Timeout) * time.Second
	if ossInfo.Timeout <= 0 {
		timeout = 30 * time.Second // 默认30秒超时
	}

	client := &http.Client{
		Timeout: timeout,
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OSS服务返回错误状态: %d", resp.StatusCode)
	}

	// 读取响应内容（通常返回的是OSS ID或其他标识符）
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应内容失败: %w", err)
	}

	// 返回OSS服务的响应（可能是OSS ID、文件标识符或URL）
	ossResponse := string(responseBody)
	return ossResponse, nil
}
