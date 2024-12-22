package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"easyCacheMirror/internal/logger"
	"easyCacheMirror/internal/models"

	"go.uber.org/zap"
)

// Proxy 处理上游请求的代理
type Proxy struct {
	defaultClient *http.Client
}

// NewProxy 创建新的代理实例
func NewProxy() *Proxy {
	return &Proxy{
		defaultClient: &http.Client{},
	}
}

// ProxyRequest 代理请求到上游服务器并返回响应
func (p *Proxy) ProxyRequest(mirror *models.Mirror, path string, headers http.Header) (*http.Response, error) {
	log := logger.GetLogger()

	// 构建上游URL
	upstreamURL := fmt.Sprintf("%s/%s",
		strings.TrimRight(mirror.UpstreamURL, "/"),
		path,
	)

	log.Debug("代理请求",
		zap.String("upstream_url", upstreamURL),
		zap.String("method", "GET"),
		zap.Any("headers", headers),
	)

	// 创建请求
	req, err := http.NewRequest("GET", upstreamURL, nil)
	if err != nil {
		log.Error("创建请求失败", zap.Error(err))
		return nil, fmt.Errorf("创建上游请求失败: %v", err)
	}

	// 复制请求头，包括缓存相关的头
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// 设置默认的 User-Agent
	if _, ok := req.Header["User-Agent"]; !ok {
		req.Header.Set("User-Agent", "EasyCacheMirror")
	}

	// 根据配置选择客户端
	var client *http.Client
	if mirror.UseProxy {
		client, err = p.getProxyClient(mirror.ProxyURL)
		if err != nil {
			log.Error("创建代理客户端失败", zap.Error(err))
			return nil, fmt.Errorf("创建代理客户端失败: %v", err)
		}
		log.Info("使用代理", zap.String("proxy_url", mirror.ProxyURL))
	} else {
		client = p.defaultClient
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		log.Error("请求失败", zap.Error(err))
		return nil, fmt.Errorf("代理请求失败: %v", err)
	}

	return resp, nil
}

// getProxyClient 获取配置了代理的HTTP客户端
func (p *Proxy) getProxyClient(proxyURL string) (*http.Client, error) {
	// 解析代理URL
	proxy, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("解析代理URL失败: %v", err)
	}

	// 创建带有代理的传输层
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxy),
	}

	// 创建新的客户端
	client := &http.Client{
		Transport: transport,
	}

	return client, nil
}
