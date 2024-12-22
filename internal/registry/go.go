package registry

import (
	"fmt"
	"io"
	"strings"

	"easyCacheMirror/internal/models"
	"easyCacheMirror/internal/proxy"

	"github.com/gin-gonic/gin"
)

type GoHandler struct {
	BaseHandler
	proxy *proxy.Proxy
}

func NewGoHandler() *GoHandler {
	p := proxy.NewProxy()
	if p == nil {
		fmt.Println("警告: proxy.NewProxy() 返回 nil")
	}

	handler := &GoHandler{
		proxy: p,
	}

	// 验证初始化
	if handler.proxy == nil {
		fmt.Println("错误: GoHandler 初始化后 proxy 为 nil")
	} else {
		fmt.Println("GoHandler 初始化成功")
	}

	return handler
}

func (h *GoHandler) SupportedType() string {
	return "Go"
}

func (h *GoHandler) Handle(c *gin.Context, mirror *models.Mirror, path string) error {
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
	requestType := h.getRequestType(path)

	// 打印请求信息
	fmt.Printf("[INFO] 处理Go请求: %s (类型: %s)\n", path, requestType)

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

// getRequestType 判断Go请求的类型
func (h *GoHandler) getRequestType(path string) string {
	// Go模块代理的标准路径格式：
	// /@v/list - 列出所有版本
	// /@latest - 获取最新版本
	// /@v/{version}.info - 版本信息
	// /@v/{version}.mod - go.mod文件
	// /@v/{version}.zip - 模块源码
	switch {
	case strings.HasSuffix(path, "/@v/list"):
		return "version-list"
	case strings.HasSuffix(path, "/@latest"):
		return "latest-version"
	case strings.HasSuffix(path, ".info"):
		return "version-info"
	case strings.HasSuffix(path, ".mod"):
		return "go-mod"
	case strings.HasSuffix(path, ".zip"):
		return "source"
	case strings.Contains(path, "/@v/"):
		return "version-query"
	default:
		return "other"
	}
}
