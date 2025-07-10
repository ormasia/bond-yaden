// Package pkgNacos 提供 Nacos 配置中心客户端的封装
// 支持多命名空间的配置管理和连接池化
package pkgNacos

import (
	"fmt"
	"sync"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// NacosCli Nacos 配置客户端结构体
// 管理多个命名空间的 Nacos 配置客户端连接
type NacosCli struct {
	NacosCliMap map[string]config_client.IConfigClient // 命名空间到客户端的映射，实现连接复用
	CacheLogDir string                                 // 缓存日志目录路径
	LogDir      string                                 // 日志文件目录路径
	LogLvl      string                                 // 日志级别（debug/info/warn/error）
	lock        sync.Mutex                             // 互斥锁，保护并发访问客户端映射
	addr        string                                 // Nacos 服务器地址
	port        int32                                  // Nacos 服务器端口
	accessKey   string                                 // KMS 访问密钥（用于配置加密）
	secretKey   string                                 // KMS 秘密密钥（用于配置加密）
	region      string                                 // 区域标识符
}

// NacosCliOption 配置选项函数类型
// 使用函数式选项模式配置 NacosCli 实例
type NacosCliOption func(*NacosCli)

// NewNacosOpts 创建新的 Nacos 客户端实例
// 使用函数式选项模式，支持灵活的配置参数
// opts: 可变参数，用于配置客户端的各种选项
// 返回: 配置好的 NacosCli 实例指针
func NewNacosOpts(opts ...NacosCliOption) *NacosCli {
	c := &NacosCli{
		NacosCliMap: make(map[string]config_client.IConfigClient), // 初始化客户端映射
		LogLvl:      "warn",                                       // 默认日志级别为警告
	}
	// 应用所有配置选项
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithCacheDir 设置缓存日志目录
// cacheLogDir: 缓存日志文件存储目录路径
// 返回: 配置选项函数
func WithCacheDir(cacheLogDir string) NacosCliOption {
	return func(config *NacosCli) {
		config.CacheLogDir = cacheLogDir
	}
}

// WithLogDir 设置日志文件目录
// dir: 日志文件存储目录路径
// 返回: 配置选项函数
func WithLogDir(dir string) NacosCliOption {
	return func(config *NacosCli) {
		config.LogDir = dir
	}
}

// WithLogLevel 设置日志级别
// lvl: 日志级别字符串（debug/info/warn/error）
// 返回: 配置选项函数
func WithLogLevel(lvl string) NacosCliOption {
	return func(config *NacosCli) {
		config.LogLvl = lvl
	}
}

// WithAddr 设置 Nacos 服务器地址
// addr: Nacos 服务器的 IP 地址或域名
// 返回: 配置选项函数
func WithAddr(addr string) NacosCliOption {
	return func(config *NacosCli) {
		config.addr = addr
	}
}

// WithPort 设置 Nacos 服务器端口
// port: Nacos 服务器端口号
// 返回: 配置选项函数
func WithPort(port int32) NacosCliOption {
	return func(config *NacosCli) {
		config.port = port
	}
}

// WithRegionId 设置区域标识符
// region: 区域 ID，用于多区域部署场景
// 返回: 配置选项函数
func WithRegionId(region string) NacosCliOption {
	return func(config *NacosCli) {
		config.region = region
	}
}

// WithKmsAK 设置 KMS 访问密钥
// ak: KMS Access Key，用于配置数据的加密解密
// 返回: 配置选项函数
func WithKmsAK(ak string) NacosCliOption {
	return func(config *NacosCli) {
		config.accessKey = ak
	}
}

// WithKmsSK 设置 KMS 秘密密钥
// sk: KMS Secret Key，用于配置数据的加密解密
// 返回: 配置选项函数
func WithKmsSK(sk string) NacosCliOption {
	return func(config *NacosCli) {
		config.secretKey = sk
	}
}

// getNacosCli 获取指定命名空间的 Nacos 配置客户端
// 实现了客户端连接池，避免重复创建连接
// ns: 命名空间标识符
// 返回: 配置客户端接口和可能的错误
func (c *NacosCli) getNacosCli(ns string) (config_client.IConfigClient, error) {
	c.lock.Lock()         // 加锁保护并发访问
	defer c.lock.Unlock() // 函数结束时解锁

	// 检查是否已存在该命名空间的客户端连接
	if cli, ok := c.NacosCliMap[ns]; ok {
		return cli, nil
	}

	// 创建客户端配置
	cc := *constant.NewClientConfig(
		constant.WithTimeoutMs(5000),           // 设置超时时间为 5 秒
		constant.WithNamespaceId(ns),           // 设置命名空间 ID
		constant.WithOpenKMS(true),             // 启用 KMS 加密
		constant.WithRegionId(c.region),        // 设置区域 ID
		constant.WithSecretKey(c.secretKey),    // 设置 KMS 秘密密钥
		constant.WithAccessKey(c.accessKey),    // 设置 KMS 访问密钥
		constant.WithNotLoadCacheAtStart(true), // 启动时不加载缓存
		constant.WithLogDir(c.LogDir),          // 设置日志目录
		constant.WithCacheDir(c.CacheLogDir),   // 设置缓存目录
		constant.WithLogLevel(c.LogLvl),        // 设置日志级别
	)

	// 创建服务器配置
	sc := []constant.ServerConfig{
		{
			IpAddr: c.addr,         // Nacos 服务器地址
			Port:   uint64(c.port), // Nacos 服务器端口
		},
	}

	// 创建 Nacos 配置客户端
	nacosCli, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &cc, // 客户端配置
			ServerConfigs: sc,  // 服务器配置列表
		},
	)
	if err != nil {
		fmt.Println("init nacos client error:", err)
		return nil, err
	}

	// 缓存客户端连接以供后续使用
	c.NacosCliMap[ns] = nacosCli
	return nacosCli, nil
}

// GetCfgFromNacos 从 Nacos 配置中心获取配置内容
// 这是对外暴露的主要方法，用于获取指定的配置数据
// id: 配置数据 ID（Data ID）
// group: 配置分组名称
// ns: 命名空间标识符
// 返回: 配置内容字符串和可能的错误
func (c *NacosCli) GetCfgFromNacos(id, group, ns string) (string, error) {
	// 获取指定命名空间的客户端连接
	cli, err := c.getNacosCli(ns)
	if err != nil {
		return "", err
	}

	// 从 Nacos 获取配置内容
	return cli.GetConfig(vo.ConfigParam{
		Group:  group, // 配置分组
		DataId: id,    // 配置数据 ID
	})
}
