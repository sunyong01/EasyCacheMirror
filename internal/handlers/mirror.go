package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"easyCacheMirror/internal/cache"
	"easyCacheMirror/internal/database"
	"easyCacheMirror/internal/models"
	"easyCacheMirror/internal/registry"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 创建镜像目录
func createMirrorDirectory(path string) error {
	// 检查目录是否已存在
	if _, err := os.Stat(path); err == nil {
		// 目录已存在，直接返回
		return nil
	} else if !os.IsNotExist(err) {
		// 如果是其他错误（比如权限问题），返回错误
		return err
	}
	// 目录不存在，创建新目录
	return os.MkdirAll(path, 0755)
}

// 删除镜像目录
func deleteMirrorDirectory(path string) error {
	return os.RemoveAll(path)
}

// 获取镜像列表
func ListMirrors(c *gin.Context) {
	var mirrors []models.Mirror
	result := database.DB.Find(&mirrors)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取镜像列表失败",
		})
		return
	}

	// 创建响应结构
	type MirrorResponse struct {
		models.Mirror
		UsedSpace int64 `json:"usedSpace"`
	}

	var response []MirrorResponse
	for _, mirror := range mirrors {
		// 获取已用空间
		usedSpace, err := database.GetMirrorUsedSpace(mirror.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "计算镜像使用空间失败",
			})
			return
		}

		response = append(response, MirrorResponse{
			Mirror:    mirror,
			UsedSpace: usedSpace,
		})
	}

	c.JSON(http.StatusOK, response)
}

// 创建镜像
func CreateMirror(c *gin.Context) {
	var mirror models.Mirror
	if err := c.ShouldBindJSON(&mirror); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查是否已存在相同类型的镜像
	var existingMirror models.Mirror
	result := database.DB.Where("type = ?", mirror.Type).First(&existingMirror)
	if result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("已存在 %s 类型的镜像: %s", mirror.Type, existingMirror.Name),
		})
		return
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "检查镜像类型失败",
		})
		return
	}

	// 如果是 NPM 类型且没有指定上游地址，使用默认地址
	if mirror.Type == "NPM" && mirror.UpstreamURL == "" {
		mirror.UpstreamURL = models.DefaultNPMRegistry
	}

	// 创建Blob存储目录
	if err := createMirrorDirectory(mirror.BlobPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建存储目录失败",
		})
		return
	}

	if err := database.DB.Create(&mirror).Error; err != nil {
		// 如果数据库创建失败，删除已创建的目录
		os.RemoveAll(mirror.BlobPath)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建镜像失败",
		})
		return
	}

	// 添加到缓存
	cache.GetMirrorCache().Set(&mirror)

	c.JSON(http.StatusOK, mirror)
}

// 更新镜像
func UpdateMirror(c *gin.Context) {
	var mirror models.Mirror
	if err := c.ShouldBindJSON(&mirror); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求数据",
		})
		return
	}

	// 获取原有镜像信息
	var oldMirror models.Mirror
	if err := database.DB.First(&oldMirror, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "镜像不存在",
		})
		return
	}

	// 如果类型发生变化，检查新类型是否已存在
	if mirror.Type != oldMirror.Type {
		var existingMirror models.Mirror
		result := database.DB.Where("type = ? AND id != ?", mirror.Type, oldMirror.ID).First(&existingMirror)
		if result.Error == nil {
			c.JSON(http.StatusConflict, gin.H{
				"error": fmt.Sprintf("已存在 %s 类型的镜像: %s", mirror.Type, existingMirror.Name),
			})
			return
		} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "检查镜像类型失败",
			})
			return
		}
	}

	// 检查名称是否被其他镜像使用
	if mirror.Name != oldMirror.Name {
		var existingMirror models.Mirror
		result := database.DB.Where("name = ? AND id != ?", mirror.Name, oldMirror.ID).First(&existingMirror)
		if result.Error == nil {
			c.JSON(http.StatusConflict, gin.H{
				"error": "镜像名称已被使用",
			})
			return
		}
	}

	// 如果Blob路径发生变化
	if oldMirror.BlobPath != mirror.BlobPath {
		// 创建新目录
		if err := createMirrorDirectory(mirror.BlobPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "创建新存储目录失败",
			})
			return
		}

		// 移动文件到新目录
		if err := moveDirectory(oldMirror.BlobPath, mirror.BlobPath); err != nil {
			// 如果移动失败，删除新创建的目录
			os.RemoveAll(mirror.BlobPath)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "移动存储目录失败",
			})
			return
		}

		// 删除旧目录
		os.RemoveAll(oldMirror.BlobPath)
	}

	// 保持原有的统计数据不变
	mirror.ID = oldMirror.ID
	mirror.LastUsedTime = oldMirror.LastUsedTime
	mirror.CreatedAt = oldMirror.CreatedAt
	mirror.LastCleanup = oldMirror.LastCleanup
	mirror.RequestCount = oldMirror.RequestCount // 保留请求次数
	mirror.HitCount = oldMirror.HitCount         // 保留命中次数

	// 更新镜像
	if err := database.DB.Save(&mirror).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "更新镜像失败",
		})
		return
	}

	// 更新缓存
	cache.GetMirrorCache().Set(&mirror)

	c.JSON(http.StatusOK, mirror)
}

// 删除镜像
func DeleteMirror(c *gin.Context) {
	var mirror models.Mirror
	if err := database.DB.First(&mirror, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "镜像不存在",
		})
		return
	}

	// 删除Blob存储目录
	if err := deleteMirrorDirectory(mirror.BlobPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "删除存储目录失败",
		})
		return
	}

	if err := database.DB.Delete(&mirror).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "删除镜像失败",
		})
		return
	}

	// 从缓存中移除
	cache.GetMirrorCache().Remove(&mirror)

	c.JSON(http.StatusOK, gin.H{
		"message": "删除成功",
	})
}

// 移动目录内容
func moveDirectory(src, dst string) error {

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// 如果是目录，递归移动
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				return err
			}
			if err := moveDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// 如果是文件，直接移动
			if err := os.Rename(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// CleanupMirrorCache 清理指定镜像的缓存
func CleanupMirrorCache(c *gin.Context) {
	// 获取镜像ID
	mirrorID := c.Param("id")

	// 查找镜像
	var mirror models.Mirror
	if err := database.DB.First(&mirror, mirrorID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "镜像不存在",
		})
		return
	}

	// 获取对应类型的处理器
	handler := registry.GetRegistry().GetHandler(mirror.Type)
	if handler == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "不支持的镜像类型",
		})
		return
	}

	// 执行缓存清理
	if err := handler.CleanupCache(c, &mirror); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("清理缓存失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "缓存清理完成",
	})
}

// GetSimpleMirrors 获取简化的镜像列表
func GetSimpleMirrors(c *gin.Context) {
	var mirrors []models.Mirror
	if err := database.DB.Find(&mirrors).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取镜像列表失败"})
		return
	}

	type SimpleMirror struct {
		ID          uint   `json:"id"`
		Type        string `json:"type"`
		AccessPoint string `json:"access_point"`
	}

	result := make([]SimpleMirror, 0, len(mirrors))
	for _, m := range mirrors {
		// 格式化访问路径
		serviceURL := strings.TrimRight(m.ServiceURL, "/")
		accessURL := strings.TrimLeft(m.AccessURL, "/")
		accessPoint := serviceURL + "/" + accessURL

		result = append(result, SimpleMirror{
			ID:          m.ID,
			Type:        m.Type,
			AccessPoint: accessPoint,
		})
	}

	c.JSON(http.StatusOK, result)
}
