package registry

import (
	"fmt"
	"io"
	"strings"

	"easyCacheMirror/internal/logger"
	"easyCacheMirror/internal/models"
	"easyCacheMirror/internal/proxy"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PyPiHandler struct {
	BaseHandler
	proxy *proxy.Proxy
}

func NewPyPiHandler() *PyPiHandler {
	p := proxy.NewProxy()
	if p == nil {
		fmt.Println("警告: proxy.NewProxy() 返回 nil")
	}

	handler := &PyPiHandler{
		proxy: p,
	}

	// 验证初始化
	if handler.proxy == nil {
		fmt.Println("错误: PyPiHandler 初始化后 proxy 为 nil")
	} else {
		fmt.Println("PyPiHandler 初始化成功")
	}

	return handler
}

func (h *PyPiHandler) SupportedType() string {
	return "PyPI"
}

func (h *PyPiHandler) Handle(c *gin.Context, mirror *models.Mirror, path string) error {
	log := logger.GetLogger()
	path = strings.TrimPrefix(path, "/")

	// 检查请求类型
	requestType := h.getRequestType(path)
	log.Debug("处理PyPI请求",
		zap.String("path", path),
		zap.String("type", requestType),
	)

	// 代理请求到上游
	resp, err := h.proxy.ProxyRequest(mirror, path, c.Request.Header)
	if err != nil {
		return fmt.Errorf("代理请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 复制响应头
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.Status(resp.StatusCode)

	// 复制响应体
	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		return fmt.Errorf("写入响应失败: %v", err)
	}

	return nil
}

// getRequestType 判断PyPI请求的类型
func (h *PyPiHandler) getRequestType(path string) string {
	switch {
	case strings.HasSuffix(path, ".whl"):
		return "wheel"
	case strings.HasSuffix(path, ".tar.gz"):
		return "sdist"
	case strings.HasSuffix(path, "/json"):
		return "metadata"
	case strings.HasSuffix(path, ".egg"):
		return "egg"
	case strings.HasSuffix(path, ".zip"):
		return "zip"
	case strings.Contains(path, "simple/"):
		return "simple"
	default:
		return "other"
	}
}
