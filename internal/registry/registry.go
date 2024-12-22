package registry

import (
	"fmt"
	"sync"

	"easyCacheMirror/internal/logger"
	"easyCacheMirror/internal/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Registry 管理所有的镜像处理器
type Registry struct {
	handlers map[string]Handler
	mu       sync.RWMutex
}

var (
	registry *Registry
	once     sync.Once
)

// GetRegistry 获取单例的Registry实例
func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{
			handlers: make(map[string]Handler),
		}
		// 初始化并注册处理器
		registry.registerHandlers()
	})
	return registry
}

// registerHandlers 注册所有支持的处理器
func (r *Registry) registerHandlers() {
	log := logger.GetLogger()

	// NPM 处理器
	npmHandler := NewNpmHandler()
	if npmHandler == nil {
		log.Error("无法创建NPM处理器")
		return
	}
	log.Info("注册处理器", zap.String("type", "NPM"))
	r.Register(npmHandler)

	// Maven 处理器
	mavenHandler := NewMavenHandler()
	if mavenHandler == nil {
		log.Error("无法创建Maven处理器")
		return
	}
	log.Info("注册处理器", zap.String("type", "Maven"))
	r.Register(mavenHandler)

	// PyPI 处理器
	pypiHandler := NewPyPiHandler()
	if pypiHandler == nil {
		log.Error("无法创建PyPI处理器")
		return
	}
	log.Info("注册处理器", zap.String("type", "PyPI"))
	r.Register(pypiHandler)

	// R 处理器
	rHandler := NewRHandler()
	if rHandler == nil {
		log.Error("无法创建R处理器")
		return
	}
	log.Info("注册处理器", zap.String("type", "R"))
	r.Register(rHandler)

	// Go 处理器
	goHandler := NewGoHandler()
	if goHandler == nil {
		log.Error("无法创建Go处理器")
		return
	}
	log.Info("注册处理器", zap.String("type", "Go"))
	r.Register(goHandler)

	// RubyGems 处理器
	rubygemsHandler := NewRubyGemsHandler()
	if rubygemsHandler == nil {
		log.Error("无法创建RubyGems处理器")
		return
	}
	log.Info("注册处理器", zap.String("type", "RubyGems"))
	r.Register(rubygemsHandler)

	// Conda 处理器
	condaHandler := NewCondaHandler()
	if condaHandler == nil {
		log.Error("无法创建Conda处理器")
		return
	}
	log.Info("注册处理器", zap.String("type", "Conda"))
	r.Register(condaHandler)

	// Cargo 处理器
	cargoHandler := NewCargoHandler()
	if cargoHandler == nil {
		log.Error("无法创建Cargo处理器")
		return
	}
	log.Info("注册处理器", zap.String("type", "Cargo"))
	r.Register(cargoHandler)

	// Docker 处理器
	dockerHandler := NewDockerHandler()
	if dockerHandler == nil {
		log.Error("无法创建Docker处理器")
		return
	}
	log.Info("注册处理器", zap.String("type", "Docker"))
	r.Register(dockerHandler)
}

// Register 注册一个新的处理器
func (r *Registry) Register(handler Handler) {
	log := logger.GetLogger()
	r.mu.Lock()
	defer r.mu.Unlock()

	handlerType := handler.SupportedType()
	log.Debug("注册处理器",
		zap.String("type", handlerType),
		zap.String("handler", "Registry.Register"),
	)
	r.handlers[handlerType] = handler
}

// GetHandler 获取指定类型的处理器
func (r *Registry) GetHandler(mirrorType string) Handler {
	log := logger.GetLogger()
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, exists := r.handlers[mirrorType]
	if !exists {
		log.Error("未找到处理器", zap.String("type", mirrorType))
		return nil
	}

	//log.Debug("获取处理器",
	//	zap.String("type", mirrorType),
	//	zap.String("handler", "Registry.GetHandler"),
	//)
	return handler
}

// Handler 定义了处理器接口
type Handler interface {
	SupportedType() string
	Handle(c *gin.Context, mirror *models.Mirror, path string) error
	CleanupCache(c *gin.Context, mirror *models.Mirror) error
}

// BaseHandler 提供基本的处理器实现
type BaseHandler struct{}

// SupportedType 基本的类型实现
func (h *BaseHandler) SupportedType() string {
	return "UNKNOWN"
}

// Handle 基本的处理实现
func (h *BaseHandler) Handle(c *gin.Context, mirror *models.Mirror, path string) error {
	return fmt.Errorf("未实现的处理方法")
}

// CleanupCache 基本的清理缓存实现
func (h *BaseHandler) CleanupCache(c *gin.Context, mirror *models.Mirror) error {
	// 默认实现，暂时返回 nil
	return nil
}
