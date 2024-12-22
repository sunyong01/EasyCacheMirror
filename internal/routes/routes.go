package routes

import (
	"easyCacheMirror/internal/cache"
	"easyCacheMirror/internal/database"
	"easyCacheMirror/internal/handlers"
	"easyCacheMirror/internal/logger"
	"easyCacheMirror/internal/models"

	"go.uber.org/zap"

	"strings"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// 初始化镜像缓存
	if err := initMirrorCache(); err != nil {
		log := logger.GetLogger()
		log.Error("初始化镜像缓存失败", zap.Error(err))
	}

	// 静态文件服务
	r.Static("/assets", "./dist/assets")               // 服务前端资源文件
	r.StaticFile("/", "./dist/index.html")             // 服务主页
	r.StaticFile("/favicon.ico", "./dist/favicon.ico") // 服务网站图标

	controller := handlers.NewController()
	handler := &handlers.Handler{}

	// API 路由
	api := r.Group("/api")
	{
		api.GET("/mirrors", handlers.ListMirrors)
		api.POST("/mirrors", handlers.CreateMirror)
		api.PUT("/mirrors/:id", handlers.UpdateMirror)
		api.DELETE("/mirrors/:id", handlers.DeleteMirror)

		api.GET("/storage", handler.HandleStorage)
		api.DELETE("/storage", handler.HandleStorage)

		// 添加清理缓存的路由
		api.POST("/mirrors/:id/cleanup", handlers.CleanupMirrorCache)

		// 添加简化的镜像列表接口
		api.GET("/mirrors/simple", handlers.GetSimpleMirrors)
	}

	// 所有其他请求交给控制器处理，但要排除前端路由
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		log := logger.GetLogger()
		log.Debug("处理请求", zap.String("path", path))
		// 检查是否为镜像请求或 API 请求
		if isAPIRequest(path) {
			log.Debug("处理镜像或API请求", zap.String("path", path))
			controller.HandleRequest(c)
			return
		}

		// 如果不是镜像请求，返回前端页面
		log.Debug("返回前端页面", zap.String("path", path))
		c.File("./dist/index.html")
	})
}

// 判断是否为 API 请求或镜像请求
func isAPIRequest(path string) bool {
	log := logger.GetLogger()

	// 规范化路径：移除所有多余的斜杠
	path = strings.Trim(path, "/")
	path = strings.ReplaceAll(path, "//", "/")

	log.Debug("规范化路径",
		zap.String("original_path", path),
		zap.String("normalized_path", path),
	)

	// 检查是否为 API 请求
	if strings.HasPrefix(path, "api") {
		log.Debug("API请求", zap.String("path", path))
		return true
	}

	// 从缓存中获取所有镜像
	mirrors := cache.GetMirrorCache().GetAll()
	for _, mirror := range mirrors {
		// 规范化镜像路径
		accessURL := strings.TrimPrefix(mirror.AccessURL, "/")

		// 完全匹配路径前缀
		if strings.HasPrefix(path+"/", accessURL+"/") {
			log.Debug("匹配到镜像路径",
				zap.String("path", path),
				zap.String("mirror_path", accessURL),
				zap.String("mirror_type", mirror.Type),
			)
			return true
		}
	}

	log.Debug("非API/镜像请求", zap.String("path", path))
	return false
}

// 初始化镜像缓存
func initMirrorCache() error {
	var mirrors []models.Mirror
	if err := database.DB.Find(&mirrors).Error; err != nil {
		return err
	}

	mirrorCache := cache.GetMirrorCache()
	for _, mirror := range mirrors {
		mirror := mirror // 创建副本
		mirrorCache.Set(&mirror)
	}

	log := logger.GetLogger()
	log.Debug("镜像缓存初始化完成",
		zap.Int("mirror_count", len(mirrors)),
	)

	return nil
}
