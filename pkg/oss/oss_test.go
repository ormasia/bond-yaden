package oss

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUploadFile_Mock(t *testing.T) {
	// 创建本地测试服务器
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查请求方法和路径
		if r.Method != "POST" {
			t.Errorf("请求方法错误: %s", r.Method)
		}
		// 返回模拟响应
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ossid":"mockid","url":"http://mockurl"}`))
	}))
	defer ts.Close()

	ossInfo := &OssInfo{
		Url:     ts.URL, // 指向本地测试服务器
		Timeout: 5,
	}

	ossid, url, err := UploadFile("D:\\Users\\jingbo.yang\\Desktop\\projectCode\\bond-yaden\\export\\bond_latest_quotes_20250708_140804.xlsx", "bond_latest_quotes_20250708_140804.xlsx", "", nil, ossInfo)
	if err != nil {
		t.Fatalf("上传文件失败: %v", err)
	}
	t.Logf("模拟上传成功，OSS ID: %s, URL: %s\n", ossid, url)
}
