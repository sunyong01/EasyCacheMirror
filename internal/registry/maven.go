package registry

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"easyCacheMirror/internal/database"
	"easyCacheMirror/internal/logger"
	"easyCacheMirror/internal/models"
	"easyCacheMirror/internal/proxy"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type MavenHandler struct {
	proxy *proxy.Proxy
}

func NewMavenHandler() *MavenHandler {
	p := proxy.NewProxy()
	if p == nil {
		fmt.Println("警告: proxy.NewProxy() 返回 nil")
	}

	handler := &MavenHandler{
		proxy: p,
	}

	if handler.proxy == nil {
		fmt.Println("错误: MavenHandler 初始化后 proxy 为 nil")
	} else {
		fmt.Println("MavenHandler 初始化成功")
	}

	return handler
}

func (h *MavenHandler) SupportedType() string {
	return "Maven"
}

// Handle 处理Maven请求
func (h *MavenHandler) Handle(c *gin.Context, mirror *models.Mirror, path string) error {
	log := logger.GetLogger()

	// 规范化路径
	path = strings.TrimLeft(path, "/")

	log.Debug("处理Maven请求",
		zap.String("path", path),
		zap.String("mirror", mirror.Name),
		zap.String("upstream", mirror.UpstreamURL),
	)

	// 更新总请求计数
	if err := updateMirrorCounts(mirror, false); err != nil {
		log.Error("更新请求计数失败", zap.Error(err))
	}

	// 对于SNAPSHOT版本或元数据文件，直接转发到上游
	if strings.Contains(path, "SNAPSHOT") || strings.Contains(path, "maven-metadata.xml") {
		return h.proxyRequest(c, mirror, path)
	}

	// 检查缓存
	var mavenFile models.MavenFile
	result := database.DB.Where(&models.MavenFile{
		MirrorID:     mirror.ID,
		RelativePath: path,
	}).First(&mavenFile)

	if result.Error == nil {
		log.Debug("找到缓存记录",
			zap.String("path", path),
			zap.String("save_path", mavenFile.SavePath),
		)
		return h.serveCachedFile(c, mirror, &mavenFile)
	}

	log.Debug("缓存未命中，从上游获取",
		zap.String("path", path),
	)

	// 从上游获取
	resp, err := h.proxy.ProxyRequest(mirror, path, c.Request.Header)
	if err != nil {
		log.Error("代理请求失败",
			zap.Error(err),
			zap.String("path", path),
		)
		return fmt.Errorf("代理请求失败: %v", err)
	}
	defer resp.Body.Close()

	log.Debug("收到上游响应",
		zap.Int("status", resp.StatusCode),
		zap.String("content_type", resp.Header.Get("Content-Type")),
		zap.String("content_encoding", resp.Header.Get("Content-Encoding")),
	)

	// 读取原始响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应体失败: %v", err)
	}

	// 对于非SNAPSHOT版本，保存到缓存
	if resp.StatusCode == 200 {
		if err := h.processResponse(mirror, path, bodyBytes, resp); err != nil {
			log.Error("保存缓存失败", zap.Error(err))
			// 继续处理，不影响响应
		}
	}

	// 写入响应
	return h.writeResponse(c, resp, bodyBytes)
}

// serveCachedFile 从缓存提供文件
func (h *MavenHandler) serveCachedFile(c *gin.Context, mirror *models.Mirror, file *models.MavenFile) error {
	log := logger.GetLogger()

	log.Debug("命中缓存",
		zap.String("path", file.RelativePath),
		zap.String("mirror", mirror.Name),
		zap.String("save_path", file.SavePath),
	)

	// 更新使用时间和缓存命中计数
	updates := map[string]interface{}{
		"last_used_time": time.Now(),
	}
	if err := database.DB.Model(file).Updates(updates).Error; err != nil {
		log.Error("更新文件使用时间失败", zap.Error(err))
	}

	if err := updateMirrorCounts(mirror, true); err != nil {
		log.Error("更新缓存命中计数失败", zap.Error(err))
	}

	// 读取文件
	data, err := os.ReadFile(file.SavePath)
	if err != nil {
		return fmt.Errorf("读取缓存文件失败: %v", err)
	}

	// 设置响应头
	c.Header("Content-Type", file.ContentType)
	if file.ContentEncoding != "" {
		c.Header("Content-Encoding", file.ContentEncoding)
	}
	c.Status(200)

	if _, err := c.Writer.Write(data); err != nil {
		return fmt.Errorf("写入响应失败: %v", err)
	}

	return nil
}

// processResponse 处理响应内容，保存到缓存
func (h *MavenHandler) processResponse(mirror *models.Mirror, path string, bodyBytes []byte, resp *http.Response) error {
	log := logger.GetLogger()

	log.Debug("开始保存文件到缓存",
		zap.String("path", path),
		zap.Int("size", len(bodyBytes)),
	)

	// 构建保存路径
	savePath := filepath.Join(mirror.BlobPath, path)

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return fmt.Errorf("创建缓存目录失败: %v", err)
	}

	// 保存文件
	if err := os.WriteFile(savePath, bodyBytes, 0644); err != nil {
		log.Error("保存文件失败",
			zap.Error(err),
			zap.String("path", savePath),
		)
		return fmt.Errorf("保存文件失败: %v", err)
	}

	log.Debug("文件已保存到缓存",
		zap.String("path", savePath),
	)

	// 确定文件类型
	fileType := models.MavenFileTypeNormal
	if strings.Contains(path, "maven-metadata.xml") {
		fileType = models.MavenFileTypeMetadata
	}

	// 创建数据库记录
	mavenFile := models.MavenFile{
		MirrorID:        mirror.ID,
		RelativePath:    path,
		FileType:        fileType,
		FileSize:        int64(len(bodyBytes)),
		SavePath:        savePath,
		ContentType:     resp.Header.Get("Content-Type"),
		ContentEncoding: resp.Header.Get("Content-Encoding"),
		IsSnapshot:      strings.Contains(path, "SNAPSHOT"),
		DownloadedAt:    time.Now(),
		LastUsedTime:    time.Now(),
	}

	if err := database.DB.Create(&mavenFile).Error; err != nil {
		log.Error("保存文件记录失败", zap.Error(err))
	}

	return nil
}

// proxyRequest 直接代理请求到上游
func (h *MavenHandler) proxyRequest(c *gin.Context, mirror *models.Mirror, path string) error {
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
	_, err = io.Copy(c.Writer, resp.Body)
	return err
}

// writeResponse 写入响应到客户端
func (h *MavenHandler) writeResponse(c *gin.Context, resp *http.Response, bodyBytes []byte) error {
	// 复制响应头
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	c.Status(resp.StatusCode)
	if _, err := c.Writer.Write(bodyBytes); err != nil {
		return fmt.Errorf("写入响应失败: %v", err)
	}

	return nil
}

// CleanupCache 清理缓存
func (h *MavenHandler) CleanupCache(c *gin.Context, mirror *models.Mirror) error {
	log := logger.GetLogger()

	// 获取当前使用空间
	usedSpace, err := database.GetMirrorUsedSpace(mirror.ID)
	if err != nil {
		return fmt.Errorf("计算使用空间失败: %v", err)
	}

	// 如果使用率低于95%，不需要清理
	usageRatio := float64(usedSpace) / float64(mirror.MaxSize)
	if usageRatio < 0.95 {
		return nil
	}

	// 删除最老的文件直到使用率降到80%以下
	for usageRatio > 0.8 {
		var oldestFile models.MavenFile
		if err := database.DB.Where("mirror_id = ?", mirror.ID).
			Order("last_used_time asc").
			First(&oldestFile).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				break
			}
			return fmt.Errorf("查询最老文件失败: %v", err)
		}

		// 删除文件
		if err := os.Remove(oldestFile.SavePath); err != nil && !os.IsNotExist(err) {
			log.Error("删除文件失败",
				zap.Error(err),
				zap.String("path", oldestFile.SavePath),
			)
		}

		// 删除数据库记录
		if err := database.DB.Delete(&oldestFile).Error; err != nil {
			log.Error("删除文件记录失败", zap.Error(err))
		}

		// 重新计算使用空间
		usedSpace, err = database.GetMirrorUsedSpace(mirror.ID)
		if err != nil {
			return fmt.Errorf("计算使用空间失败: %v", err)
		}
		usageRatio = float64(usedSpace) / float64(mirror.MaxSize)
	}

	return nil
}
