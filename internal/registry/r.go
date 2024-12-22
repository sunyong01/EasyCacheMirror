package registry

import (
	"fmt"
	"io"
	"strings"

	"easyCacheMirror/internal/models"
	"easyCacheMirror/internal/proxy"

	"github.com/gin-gonic/gin"
)

type RHandler struct {
	BaseHandler
	proxy *proxy.Proxy
}

func NewRHandler() *RHandler {
	p := proxy.NewProxy()
	if p == nil {
		fmt.Println("警告: proxy.NewProxy() 返回 nil")
	}

	handler := &RHandler{
		proxy: p,
	}

	// 验证初始化
	if handler.proxy == nil {
		fmt.Println("错误: RHandler 初始化后 proxy 为 nil")
	} else {
		fmt.Println("RHandler 初始化成功")
	}

	return handler
}

func (h *RHandler) SupportedType() string {
	return "R"
}

func (h *RHandler) Handle(c *gin.Context, mirror *models.Mirror, path string) error {
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
	fmt.Printf("[INFO] 处理CRAN请求: %s (类型: %s)\n", path, requestType)

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

// getRequestType 判断CRAN请求的类型
func (h *RHandler) getRequestType(path string) string {
	switch {
	case strings.HasSuffix(path, ".tar.gz"):
		return "source"
	case strings.HasSuffix(path, "PACKAGES"):
		return "packages"
	case strings.HasSuffix(path, "PACKAGES.gz"):
		return "packages-gz"
	case strings.HasSuffix(path, "PACKAGES.rds"):
		return "packages-rds"
	case strings.HasSuffix(path, ".tgz"):
		return "mac-binary"
	case strings.HasSuffix(path, ".zip"):
		return "win-binary"
	case strings.Contains(path, "src/contrib"):
		return "source-contrib"
	case strings.Contains(path, "bin/windows/contrib"):
		return "win-contrib"
	case strings.Contains(path, "bin/macosx/contrib"):
		return "mac-contrib"
	default:
		return "other"
	}
}
