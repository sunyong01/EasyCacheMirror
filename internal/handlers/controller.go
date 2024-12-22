package handlers

import (
	"easyCacheMirror/internal/cache"
	"easyCacheMirror/internal/database"
	"easyCacheMirror/internal/logger"
	"easyCacheMirror/internal/models"
	"easyCacheMirror/internal/registry"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Controller 负责请求的路由分发
type Controller struct {
	mirrorCache *cache.MirrorCache
	registry    *registry.Registry
}

// NewController 创建新的控制器
func NewController() *Controller {
	log := logger.GetLogger()
	reg := registry.GetRegistry()
	if reg == nil {
		log.Error("无法获取Registry")
		return nil
	}

	return &Controller{
		mirrorCache: cache.GetMirrorCache(),
		registry:    reg,
	}
}

// HandleRequest 处理请求
func (c *Controller) HandleRequest(ctx *gin.Context) {
	log := logger.GetLogger()
	path := ctx.Request.URL.Path

	// 从缓存中查找匹配的镜像
	mirrors := cache.GetMirrorCache().GetAll()
	var matchedMirror *models.Mirror
	var relativePath string
	var longestMatch int = -1

	// 规范化路径
	normalizedPath := strings.Trim(path, "/")

	// 优先匹配最长的路径
	for _, mirror := range mirrors {
		accessURL := strings.Trim(mirror.AccessURL, "/")
		if strings.HasPrefix(normalizedPath, accessURL) {
			// 找到更长的匹配
			if len(accessURL) > longestMatch {
				longestMatch = len(accessURL)
				matchedMirror = mirror
				// 正确计算相对路径
				relativePath = strings.TrimPrefix(normalizedPath, accessURL)
				relativePath = strings.TrimPrefix(relativePath, "/")
			}
		}
	}

	if matchedMirror == nil {
		ctx.String(http.StatusNotFound, "未找到匹配的镜像")
		return
	}

	log.Debug("解析路径",
		zap.String("full_path", path),
		zap.String("access_url", matchedMirror.AccessURL),
		zap.String("relative_path", relativePath),
		zap.String("mirror_type", matchedMirror.Type),
		zap.Int("match_length", longestMatch),
	)

	// 获取对应的处理器
	handler := registry.GetRegistry().GetHandler(matchedMirror.Type)
	if handler == nil {
		ctx.String(http.StatusBadRequest, "不支持的镜像类型")
		return
	}

	// 处理请求
	if err := handler.Handle(ctx, matchedMirror, relativePath); err != nil {
		log.Error("处理请求失败",
			zap.Error(err),
			zap.String("path", path),
			zap.String("mirror_type", matchedMirror.Type),
		)
		ctx.String(http.StatusInternalServerError, "处理请求失败")
		return
	}

	// 更新最后使用时间
	if err := c.updateLastUsedTime(matchedMirror); err != nil {
		log.Error("更新使用时间失败", zap.Error(err))
	}
}

func (c *Controller) updateLastUsedTime(mirror *models.Mirror) error {
	//log := logger.GetLogger()

	// 更新数据库中的最后使用时间
	result := database.DB.Model(mirror).Update("last_used_time", database.DB.NowFunc())
	if result.Error != nil {
		return result.Error
	}

	// 更新缓存中的镜像信息
	mirror.LastUsedTime = database.DB.NowFunc()
	c.mirrorCache.Set(mirror)

	//log.Debug("已更新镜像最后使用时间",
	//	zap.Uint("mirror_id", mirror.ID),
	//	zap.Time("last_used_time", mirror.LastUsedTime),
	//)

	return nil
}
