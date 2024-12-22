package registry

import (
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
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

	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type NpmHandler struct {
	proxy *proxy.Proxy
}

func NewNpmHandler() *NpmHandler {
	p := proxy.NewProxy()
	if p == nil {
		fmt.Println("警告: proxy.NewProxy() 返回 nil")
	}

	handler := &NpmHandler{
		proxy: p,
	}

	// 验证初始化
	if handler.proxy == nil {
		fmt.Println("错误: NpmHandler 初始化后 proxy 为 nil")
	} else {
		fmt.Println("NpmHandler 初始化成功")
	}

	return handler
}

func (h *NpmHandler) SupportedType() string {
	return "NPM"
}

// handleTarball 处理 .tgz 文件请求
func (h *NpmHandler) handleTarball(c *gin.Context, mirror *models.Mirror, path string) error {
	log := logger.GetLogger()
	log.Debug("开始处理 tarball 请求",
		zap.String("path", path),
		zap.String("package", extractPackageName(path)),
		zap.String("version", extractVersion(path)),
	)

	var npmFile models.NPMFile
	result := database.DB.Where(&models.NPMFile{
		MirrorID:  mirror.ID,
		PackageID: extractPackageName(path),
		Version:   extractVersion(path),
		FileType:  models.NPMFileTypeTarball,
	}).First(&npmFile)

	log.Debug("查询 tarball 缓存结果",
		zap.Error(result.Error),
		zap.Bool("is_not_found", errors.Is(result.Error, gorm.ErrRecordNotFound)),
	)

	if result.Error == nil {
		// tarball 文件存在，直接返回缓存
		return h.serveCachedFile(c, npmFile)
	} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		log.Info("tarball未缓存，将从上游拉取",
			zap.String("package", extractPackageName(path)),
			zap.String("version", extractVersion(path)),
		)
		// 继续处理，让上层函数进行代理请求
		return nil
	}

	// 其他错误
	return fmt.Errorf("查询缓存文件失败: %v", result.Error)
}

// handleJSONMetadata 处理 JSON 元数据请求
func (h *NpmHandler) handleJSONMetadata(c *gin.Context, mirror *models.Mirror, path string) error {
	log := logger.GetLogger()
	var npmFile models.NPMFile
	result := database.DB.Where(&models.NPMFile{
		MirrorID:  mirror.ID,
		PackageID: strings.TrimPrefix(path, "/"),
		FileType:  models.NPMFileTypeJSON,
	}).First(&npmFile)
	if result.Error == nil {
		// 检查是否过期
		cacheExpireTime := npmFile.DownloadedAt.Add(time.Duration(mirror.CacheTime) * time.Minute)
		if time.Now().Before(cacheExpireTime) {
			return h.serveCachedFile(c, npmFile)
		}
		log.Info("缓存已过期，从上游拉取",
			zap.String("package", npmFile.PackageID),
			zap.Time("expired_at", cacheExpireTime),
		)
	}
	return nil
}

// serveCachedFile 从缓存中提供文件
func (h *NpmHandler) serveCachedFile(c *gin.Context, npmFile models.NPMFile) error {
	log := logger.GetLogger()

	// 获取 mirror 信息
	var mirror models.Mirror
	if err := database.DB.First(&mirror, npmFile.MirrorID).Error; err != nil {
		log.Error("获取镜像信息失败",
			zap.Error(err),
			zap.Uint("mirror_id", npmFile.MirrorID),
		)
		return fmt.Errorf("获取镜像信息失败: %v", err)
	}

	// 更新缓存命中计数和最后使用时间
	now := time.Now()
	updates := map[string]interface{}{
		"last_used_time": now,
		"downloaded_at":  now,
	}
	if err := database.DB.Model(&npmFile).Updates(updates).Error; err != nil {
		log.Error("更新文件使用时间失败", zap.Error(err))
		// 继续处理，不返回错误
	}

	// 更新缓存命中计数
	if err := updateMirrorCounts(&mirror, true); err != nil {
		log.Error("更新缓存命中计数失败", zap.Error(err))
		// 继续处理，不返回错误
	}

	data, err := os.ReadFile(npmFile.SavePath)
	if err != nil {
		log.Error("读取缓存文件失败",
			zap.Error(err),
			zap.String("path", npmFile.SavePath),
		)
		return fmt.Errorf("读取缓存文件失败: %v", err)
	}

	contentType := "application/octet-stream"
	if npmFile.FileType == models.NPMFileTypeJSON {
		contentType = "application/json"
		// 尝试解析 JSON 以验证其有效性
		var jsonTest map[string]interface{}
		if err := json.Unmarshal(data, &jsonTest); err != nil {
			log.Error("缓存的JSON文件无效",
				zap.Error(err),
				zap.String("path", npmFile.SavePath))
			return fmt.Errorf("缓存的JSON文件无效: %v", err)
		}
	}

	c.Header("Content-Type", contentType)
	c.Status(200)

	if _, err := c.Writer.Write(data); err != nil {
		log.Error("写入响应失败",
			zap.Error(err),
			zap.String("path", npmFile.SavePath),
		)
		return fmt.Errorf("写入响应失败: %v", err)
	}

	log.Debug("返回缓存文件",
		zap.String("path", npmFile.SavePath),
		zap.String("type", string(npmFile.FileType)),
	)

	return nil
}

// processResponse 处理上游响应
func (h *NpmHandler) processResponse(c *gin.Context, mirror *models.Mirror, path string, resp *http.Response) error {
	log := logger.GetLogger()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("读取响应体失败",
			zap.Error(err),
			zap.String("path", path),
		)
		return fmt.Errorf("读取响应体失败: %v", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") && c.Request.Method == "GET" {
		modifiedJSON, err := h.processJSONResponse(mirror, path, bodyBytes)
		if err != nil {
			log.Error("处理JSON响应失败",
				zap.Error(err),
				zap.String("path", path),
			)
			return err
		}
		bodyBytes = modifiedJSON
	} else if strings.HasSuffix(path, ".tgz") {
		if err := h.processTarballResponse(mirror, path, bodyBytes); err != nil {
			log.Error("处理tarball响应失败",
				zap.Error(err),
				zap.String("path", path),
			)
			return err
		}
	}

	// 写入响应
	return h.writeResponse(c, resp, bodyBytes)
}

// writeResponse 写入响应到客户端
func (h *NpmHandler) writeResponse(c *gin.Context, resp *http.Response, bodyBytes []byte) error {
	// 复制响应头
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	c.Status(resp.StatusCode)
	if c.Writer.Written() {
		return nil
	}

	if _, err := c.Writer.Write(bodyBytes); err != nil {
		if strings.Contains(err.Error(), "broken pipe") ||
			strings.Contains(err.Error(), "connection reset by peer") {
			return nil
		}
		return fmt.Errorf("写入响应失败: %v", err)
	}
	return nil
}

// Handle 主方法重构
func (h *NpmHandler) Handle(c *gin.Context, mirror *models.Mirror, path string) error {
	log := logger.GetLogger()

	// 规范化路径：移除开头的所有斜杠，并确保路径格式正确
	path = strings.TrimLeft(path, "/")

	log.Debug("处理请求",
		zap.String("original_path", c.Request.URL.Path),
		zap.String("normalized_path", path),
		zap.Bool("is_tarball", strings.HasSuffix(path, ".tgz")),
	)

	// 更新总请求计数
	if err := updateMirrorCounts(mirror, false); err != nil {
		log.Error("更新请求计数失败", zap.Error(err))
		// 继续处理，不返回错误
	}

	// 验证上游URL和初始化检查
	if err := h.validateSetup(mirror); err != nil {
		return err
	}

	// 处理 tarball 请求
	if strings.HasSuffix(path, ".tgz") {
		log.Debug("检测到 tarball 请求",
			zap.String("path", path),
			zap.String("package", extractPackageName(path)),
			zap.String("version", extractVersion(path)),
		)
		if err := h.handleTarball(c, mirror, path); err != nil {
			return err
		}
		// 如果缓存命中，直接返回
		if c.Writer.Written() {
			return nil
		}
		log.Debug("tarball未缓存，准备从上游获取",
			zap.String("path", path),
		)
	} else {
		// 处理 JSON 元数据请求
		if err := h.handleJSONMetadata(c, mirror, path); err != nil {
			return err
		}
		// 如果缓存命中，直接返回
		if c.Writer.Written() {
			return nil
		}
	}

	// 代理请求到上游
	resp, err := h.proxy.ProxyRequest(mirror, path, c.Request.Header)
	if err != nil {
		log.Error("代理请求失败",
			zap.Error(err),
			zap.String("path", path),
		)
		return fmt.Errorf("代理请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 处理响应
	if err := h.processResponse(c, mirror, path, resp); err != nil {
		log.Error("处理响应失败",
			zap.Error(err),
			zap.String("path", path),
		)
		return err
	}
	return nil
}

// validateSetup 验证设置
func (h *NpmHandler) validateSetup(mirror *models.Mirror) error {
	if mirror.UpstreamURL == "" {
		return fmt.Errorf("上游URL未")
	}
	if h == nil {
		return fmt.Errorf("handler 未初始化")
	}
	if h.proxy == nil {
		return fmt.Errorf("proxy 未初始化 (handler: %v)", h)
	}
	return nil
}

// ParsedNpmInfo 存储解析出的信息
type ParsedNpmInfo struct {
	PackageName string
	Version     string
}

// parseNpmTarballPath 从 tarball 路径中解析包名和版本
func parseNpmTarballPath(path string) ParsedNpmInfo {
	// 移除开头的斜杠
	path = strings.TrimPrefix(path, "/")

	// 分割路径
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return ParsedNpmInfo{}
	}

	// 获取最后一个部分（文件名）
	filename := parts[len(parts)-1]
	filename = strings.TrimSuffix(filename, ".tgz")

	// 处理作用域包（以@开头的包）
	var packageName string
	if strings.HasPrefix(parts[0], "@") {
		if len(parts) < 3 {
			return ParsedNpmInfo{}
		}
		packageName = parts[0] + "/" + parts[1]
		// 对于作用域包，件名格式为: map-sources-2.0.1.tgz
		// 而不是 @gulp-sourcemaps/map-sources-2.0.1.tgz
		filename = strings.TrimPrefix(filename, parts[1]+"-")
	} else {
		packageName = parts[0]
		filename = strings.TrimPrefix(filename, packageName+"-")
	}

	// 提取版本号
	version := filename
	// 除构建元数据（如在）
	if idx := strings.Index(version, "+"); idx != -1 {
		version = version[:idx]
	}

	return ParsedNpmInfo{
		PackageName: packageName,
		Version:     version,
	}
}

// extractPackageName 从 tarball 路径中取包
func extractPackageName(path string) string {
	info := parseNpmTarballPath(path)
	return info.PackageName
}

// extractVersion 从 tarball 路径中提取版本号
func extractVersion(path string) string {
	info := parseNpmTarballPath(path)
	return info.Version
}

func verifyNpmPackage(data []byte, integrity, shasum string) error {
	log := logger.GetLogger()

	// 如果有 integrity，优先使用 integrity 校验
	if integrity != "" {
		// integrity 格式: sha512-xxxxx
		parts := strings.Split(integrity, "-")
		if len(parts) != 2 {
			return fmt.Errorf("invalid integrity format: %s", integrity)
		}

		hashType := parts[0]
		expectedHash := parts[1]

		switch hashType {
		case "sha512":
			hash := sha512.Sum512(data)
			actualHash := base64.StdEncoding.EncodeToString(hash[:])
			if actualHash != expectedHash {
				log.Error("integrity 校验失败",
					zap.String("expected", expectedHash),
					zap.String("actual", actualHash),
				)
				return fmt.Errorf("integrity check failed")
			}
		default:
			return fmt.Errorf("unsupported hash type: %s", hashType)
		}

		log.Debug("integrity 校验成功")
		return nil
	}

	// 如果有 shasum，使用 shasum 校验
	if shasum != "" {
		hash := sha1.Sum(data)
		actualHash := hex.EncodeToString(hash[:])
		if actualHash != shasum {
			log.Error("shasum 校验失败",
				zap.String("expected", shasum),
				zap.String("actual", actualHash),
			)
			return fmt.Errorf("shasum check failed")
		}

		log.Debug("shasum 校验成功")
		return nil
	}

	log.Warn("没有可用的校验值")
	return nil
}

// processJSONResponse 处理 JSON 元数据响应
func (h *NpmHandler) processJSONResponse(mirror *models.Mirror, path string, bodyBytes []byte) ([]byte, error) {
	// 解析 JSON 数据
	var jsonData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &jsonData); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %v", err)
	}

	// 检查并修改 tarball URLs
	if versions, ok := jsonData["versions"].(map[string]interface{}); ok {
		for _, versionData := range versions {
			if versionInfo, ok := versionData.(map[string]interface{}); ok {
				if dist, ok := versionInfo["dist"].(map[string]interface{}); ok {
					if tarball, ok := dist["tarball"].(string); ok {
						if strings.HasPrefix(tarball, mirror.UpstreamURL) {
							accessURL := mirror.AccessURL
							if !strings.HasSuffix(accessURL, "/") {
								accessURL = accessURL + "/"
							}
							newURL := strings.Replace(tarball, mirror.UpstreamURL, mirror.ServiceURL+accessURL, 1)
							dist["tarball"] = newURL
						}
					}
				}
			}
		}
	}

	// 序列化修改后的数据
	modifiedJSON, err := json.Marshal(jsonData)
	if err != nil {
		return nil, fmt.Errorf("序列化修改后的数据失败: %v", err)
	}

	// 保存到文件
	savePath := filepath.Join(mirror.BlobPath, path+".json")
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return nil, fmt.Errorf("创建缓存目录失败: %v", err)
	}

	prettyJSON, _ := json.MarshalIndent(jsonData, "", "  ")
	if err := os.WriteFile(savePath, prettyJSON, 0644); err != nil {
		return nil, fmt.Errorf("保存 JSON 文件失败: %v", err)
	}

	// 更新数据库记录
	packageName, _ := jsonData["name"].(string)
	if packageName == "" {
		packageName = path
	}

	if err := h.updateJSONFileRecord(mirror, packageName, path, savePath, modifiedJSON); err != nil {
		return nil, err
	}
	return modifiedJSON, nil
}

// processTarballResponse 处理 tarball 响应
func (h *NpmHandler) processTarballResponse(mirror *models.Mirror, path string, bodyBytes []byte) error {
	// 构建保存路径
	savePath := filepath.Join(mirror.BlobPath, path)
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return fmt.Errorf("创建 tarball 目录失败: %v", err)
	}

	// 保存文件
	if err := os.WriteFile(savePath, bodyBytes, 0644); err != nil {
		return fmt.Errorf("保存 tarball 失败: %v", err)
	}

	// 获取包信息
	info := parseNpmTarballPath(path)
	integrity, shasum := h.getPackageChecksums(info.PackageName, info.Version)

	// 校验文件
	if err := verifyNpmPackage(bodyBytes, integrity, shasum); err != nil {
		// 删除校验失败的文件
		os.Remove(savePath)
		return fmt.Errorf("包校验失败: %v", err)
	}

	// 更新数据库记录
	return h.updateTarballFileRecord(mirror, info, savePath, bodyBytes, integrity, shasum)
}

// updateJSONFileRecord 更新 JSON 文件记录
func (h *NpmHandler) updateJSONFileRecord(mirror *models.Mirror, packageName, path, savePath string, bodyBytes []byte) error {
	fileSize := int64(len(bodyBytes))

	var npmFile models.NPMFile
	result := database.DB.Where(&models.NPMFile{
		MirrorID:  mirror.ID,
		PackageID: strings.TrimPrefix(packageName, "/"),
		FileType:  models.NPMFileTypeJSON,
		FileName:  path + ".json",
	}).First(&npmFile)

	if result.Error == nil {
		npmFile.FileSize = fileSize
		npmFile.DownloadedAt = time.Now()

		if err := database.DB.Save(&npmFile).Error; err != nil {
			return fmt.Errorf("更新文件记录失败: %v", err)
		}

		return nil
	}

	// 创建新记录
	npmFile = models.NPMFile{
		MirrorID:     mirror.ID,
		PackageID:    strings.TrimPrefix(packageName, "/"),
		FileName:     path + ".json",
		FileType:     models.NPMFileTypeJSON,
		FileSize:     fileSize,
		SavePath:     savePath,
		DownloadedAt: time.Now(),
		LastUsedTime: time.Now(),
	}

	if err := database.DB.Create(&npmFile).Error; err != nil {
		return fmt.Errorf("创建文件记录失败: %v", err)
	}

	return nil
}

// getPackageChecksums 获取包的校验值
func (h *NpmHandler) getPackageChecksums(packageName, version string) (integrity, shasum string) {
	var metadataFile models.NPMFile
	result := database.DB.Where("package_id = ? AND file_type = ?",
		packageName, models.NPMFileTypeJSON).First(&metadataFile)

	if result.Error == nil {
		if jsonData, err := os.ReadFile(metadataFile.SavePath); err == nil {
			var metadata map[string]interface{}
			if err := json.Unmarshal(jsonData, &metadata); err == nil {
				if versions, ok := metadata["versions"].(map[string]interface{}); ok {
					if versionInfo, ok := versions[version].(map[string]interface{}); ok {
						if dist, ok := versionInfo["dist"].(map[string]interface{}); ok {
							integrity, _ = dist["integrity"].(string)
							shasum, _ = dist["shasum"].(string)
						}
					}
				}
			}
		}
	}
	return
}

// updateTarballFileRecord 更新 tarball 文件记录
func (h *NpmHandler) updateTarballFileRecord(mirror *models.Mirror, info ParsedNpmInfo, savePath string, bodyBytes []byte, integrity, shasum string) error {
	npmFile := &models.NPMFile{
		MirrorID:     mirror.ID,
		PackageID:    info.PackageName,
		Version:      info.Version,
		FileName:     filepath.Base(savePath),
		FileType:     models.NPMFileTypeTarball,
		FileSize:     int64(len(bodyBytes)),
		SavePath:     savePath,
		DownloadedAt: time.Now(),
		LastUsedTime: time.Now(),
		Integrity:    integrity,
		Shasum:       shasum,
	}

	if err := database.DB.Create(npmFile).Error; err != nil {
		return fmt.Errorf("保存文件记录失败: %v", err)
	}

	return nil
}

// updateMirrorCounts 更新镜像的请求计数
func updateMirrorCounts(mirror *models.Mirror, isHit bool) error {
	updates := map[string]interface{}{
		"request_count": gorm.Expr("request_count + ?", 1),
	}
	if isHit {
		updates["hit_count"] = gorm.Expr("hit_count + ?", 1)
	}

	result := database.DB.Model(mirror).Updates(updates)
	return result.Error
}

// CleanupCache 清理缓存
func (h *NpmHandler) CleanupCache(c *gin.Context, mirror *models.Mirror) error {
	log := logger.GetLogger()

	// 获取当前使用空间
	usedSpace, err := database.GetMirrorUsedSpace(mirror.ID)
	if err != nil {
		return fmt.Errorf("计算使用空间失败: %v", err)
	}

	// 计算使用率
	usageRatio := float64(usedSpace) / float64(mirror.MaxSize)
	if usageRatio < 0.95 {
		return nil // 使用率低于95%，不需要清理
	}

	log.Info("开始清理缓存",
		zap.String("mirror", mirror.Name),
		zap.Float64("usage_ratio", usageRatio),
		zap.Int64("used_space", usedSpace),
	)

	// 1. 删除过期的JSON文件
	var expiredJSONFiles []models.NPMFile
	expireTime := time.Now().Add(-time.Duration(mirror.CacheTime) * time.Minute)
	if err := database.DB.Where("mirror_id = ? AND file_type = ? AND downloaded_at < ?",
		mirror.ID, models.NPMFileTypeJSON, expireTime).Find(&expiredJSONFiles).Error; err != nil {
		return fmt.Errorf("查询过期JSON文件失败: %v", err)
	}

	// 删除过期的JSON文件
	for _, file := range expiredJSONFiles {
		if err := os.Remove(file.SavePath); err != nil && !os.IsNotExist(err) {
			log.Error("删除过期JSON文件失败",
				zap.Error(err),
				zap.String("path", file.SavePath),
			)
			continue
		}
		if err := database.DB.Delete(&file).Error; err != nil {
			log.Error("删除JSON文件记录失败", zap.Error(err))
			continue
		}
	}

	// 2. 循环删除最老的tarball文件直到使用率低于80%
	for {
		// 重新计算使用空间
		usedSpace, err = database.GetMirrorUsedSpace(mirror.ID)
		if err != nil {
			return fmt.Errorf("计算使用空间失败: %v", err)
		}

		usageRatio = float64(usedSpace) / float64(mirror.MaxSize)
		if usageRatio <= 0.8 {
			break // 已达到目标使用率，退出循环
		}

		var oldestTarball models.NPMFile
		if err := database.DB.Where("mirror_id = ? AND file_type = ?",
			mirror.ID, models.NPMFileTypeTarball).
			Order("COALESCE(last_used_time, downloaded_at) asc").
			First(&oldestTarball).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Info("没有可删除的tarball文件，开始删除旧的JSON文件",
					zap.String("mirror", mirror.Name),
					zap.Uint("mirror_id", mirror.ID),
				)
				// 3. 如果没有 tarball 可删除且空间仍然不足，开始删除最老的 JSON 文件
				for usageRatio > 0.8 {
					var oldestJSON models.NPMFile
					if err := database.DB.Where("mirror_id = ? AND file_type = ?",
						mirror.ID, models.NPMFileTypeJSON).
						Order("COALESCE(last_used_time, downloaded_at) asc").
						First(&oldestJSON).Error; err != nil {
						if errors.Is(err, gorm.ErrRecordNotFound) {
							log.Info("没有可删除的JSON文件",
								zap.String("mirror", mirror.Name),
							)
							break
						}
						return fmt.Errorf("查询最老的JSON文件失败: %v", err)
					}

					log.Info("准备删除JSON文件",
						zap.String("package", oldestJSON.PackageID),
						zap.Time("last_used", oldestJSON.LastUsedTime),
						zap.Time("downloaded", oldestJSON.DownloadedAt),
					)

					// 删除文件
					if err := os.Remove(oldestJSON.SavePath); err != nil && !os.IsNotExist(err) {
						log.Error("删除JSON文件失败",
							zap.Error(err),
							zap.String("path", oldestJSON.SavePath),
						)
						continue
					}

					// 删除数据库记录
					if err := database.DB.Delete(&oldestJSON).Error; err != nil {
						log.Error("删除JSON记录失败", zap.Error(err))
						continue
					}

					// 重新计算使用空间
					usedSpace, err = database.GetMirrorUsedSpace(mirror.ID)
					if err != nil {
						return fmt.Errorf("计算使用空间失败: %v", err)
					}
					usageRatio = float64(usedSpace) / float64(mirror.MaxSize)
				}
				break
			}
			return fmt.Errorf("查询最老的tarball失败: %v", err)
		}

		log.Info("准备删除tarball文件",
			zap.String("package", oldestTarball.PackageID),
			zap.String("version", oldestTarball.Version),
			zap.Time("last_used", oldestTarball.LastUsedTime),
			zap.Time("downloaded", oldestTarball.DownloadedAt),
		)

		// 删除文件
		if err := os.Remove(oldestTarball.SavePath); err != nil && !os.IsNotExist(err) {
			log.Error("删除tarball文件失败",
				zap.Error(err),
				zap.String("path", oldestTarball.SavePath),
			)
			continue
		}

		// 删除数据库记录
		if err := database.DB.Delete(&oldestTarball).Error; err != nil {
			log.Error("删除tarball记录失败", zap.Error(err))
			continue
		}

		log.Debug("删除旧tarball文件",
			zap.String("package", oldestTarball.PackageID),
			zap.String("version", oldestTarball.Version),
		)

		// 检查是否已达到目标使用率
		if usageRatio <= 0.8 {
			break
		}
	}

	log.Info("缓存清理完成",
		zap.String("mirror", mirror.Name),
		zap.Int64("used_space", usedSpace),
		zap.Float64("new_usage_ratio", usageRatio),
	)

	return nil
}
