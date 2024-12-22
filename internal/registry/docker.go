package registry

import (
	"fmt"
	"io"
	"strings"

	"easyCacheMirror/internal/models"
	"easyCacheMirror/internal/proxy"

	"github.com/gin-gonic/gin"
)

type DockerHandler struct {
	BaseHandler
	proxy *proxy.Proxy
}

func NewDockerHandler() *DockerHandler {
	p := proxy.NewProxy()
	if p == nil {
		fmt.Println("警告: proxy.NewProxy() 返回 nil")
	}

	handler := &DockerHandler{
		proxy: p,
	}

	// 验证初始化
	if handler.proxy == nil {
		fmt.Println("错误: DockerHandler 初始化后 proxy 为 nil")
	} else {
		fmt.Println("DockerHandler 初始化成功")
	}

	return handler
}

func (h *DockerHandler) SupportedType() string {
	return "Docker"
}

func (h *DockerHandler) Handle(c *gin.Context, mirror *models.Mirror, path string) error {
	// 添加更详细的初始化检查
	if h == nil {
		return fmt.Errorf("handler 未初始化")
	}

	if h.proxy == nil {
		return fmt.Errorf("proxy 未初始化 (handler: %v)", h)
	}

	// 移除路径开头的斜杠并确保正确的路径格式
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return fmt.Errorf("无效的路径")
	}

	// 检查请求类型
	requestType := h.getRequestType(path, c.Request.Method)

	// 打印请求信息
	fmt.Printf("[INFO] 处理Docker请求: %s (类型: %s, 方法: %s)\n",
		path, requestType, c.Request.Method)

	// 直接转发请求到上游
	resp, err := h.proxy.ProxyRequest(mirror, path, c.Request.Header)
	if err != nil {
		fmt.Printf("[ERROR] 代理请求失败: %v\n", err)
		return fmt.Errorf("代理请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 复制响应头和状态码
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.Status(resp.StatusCode)

	// 复制响应体
	written, err := io.Copy(c.Writer, resp.Body)
	if err != nil {
		fmt.Printf("[ERROR] 复制响应失败: %v\n", err)
		return fmt.Errorf("复制响应失败: %v", err)
	}

	fmt.Printf("[INFO] 完成请求: %s (状态码: %d, 大小: %d bytes)\n",
		path, resp.StatusCode, written)

	return nil
}

// getRequestType 判断Docker请求的类型
func (h *DockerHandler) getRequestType(path string, method string) string {
	switch {
	case strings.Contains(path, "/v2/_catalog"):
		return "catalog"
	case strings.Contains(path, "/v2/") && strings.HasSuffix(path, "/tags/list"):
		return "tags-list"
	case strings.Contains(path, "/v2/") && strings.Contains(path, "/manifests/"):
		if method == "HEAD" {
			return "manifest-check"
		}
		return "manifest"
	case strings.Contains(path, "/v2/") && strings.Contains(path, "/blobs/"):
		if method == "HEAD" {
			return "blob-check"
		}
		return "blob"
	case strings.Contains(path, "/v2/") && strings.Contains(path, "/uploads/"):
		return "upload"
	case strings.Contains(path, "/v2/_ping"):
		return "ping"
	case strings.Contains(path, "/v2/"):
		return "v2-api"
	default:
		return "other"
	}
}
