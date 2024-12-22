package handlers

import (
	"easyCacheMirror/internal/database"
	"easyCacheMirror/internal/models"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

type FileNode struct {
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	Path        string     `json:"path"`
	Size        int64      `json:"size"`
	ModTime     time.Time  `json:"modTime"`
	IsDirectory bool       `json:"isDirectory"`
	Children    []FileNode `json:"children,omitempty"`
	FileCount   int        `json:"fileCount,omitempty"`
	DirCount    int        `json:"dirCount,omitempty"`
}

type Handler struct {
	// 如果需要依赖项可以在这里添加
	// 例如: db *database.DB
}

// HandleStorage 统一处理存储相关的请求
func (h *Handler) HandleStorage(c *gin.Context) {
	switch c.Request.Method {
	case http.MethodGet:
		h.getStorageTree(c)
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error": "Method not allowed",
		})
	}
}

// getStorageTree 获取存储树结构
func (h *Handler) getStorageTree(c *gin.Context) {
	var mirrors []models.Mirror
	if err := database.DB.Find(&mirrors).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取镜像列表失败",
		})
		return
	}

	var rootNodes []FileNode
	for _, mirror := range mirrors {
		node, err := buildFileTree(mirror.BlobPath, mirror.Name)
		if err != nil {
			continue
		}
		rootNodes = append(rootNodes, node)
	}

	c.JSON(http.StatusOK, rootNodes)
}

// 构建文件树
func buildFileTree(root, name string) (FileNode, error) {
	info, err := os.Stat(root)
	if err != nil {
		return FileNode{}, err
	}

	node := FileNode{
		Key:         root,
		Name:        name,
		Path:        root,
		Size:        info.Size(),
		ModTime:     info.ModTime(),
		IsDirectory: info.IsDir(),
	}

	if !info.IsDir() {
		return node, nil
	}

	// 统计目录信息
	var fileCount, dirCount int
	var totalSize int64

	entries, err := os.ReadDir(root)
	if err != nil {
		return node, nil
	}

	for _, entry := range entries {
		childPath := filepath.Join(root, entry.Name())
		childInfo, err := entry.Info()
		if err != nil {
			continue
		}

		childNode := FileNode{
			Key:         childPath,
			Name:        entry.Name(),
			Path:        childPath,
			Size:        childInfo.Size(),
			ModTime:     childInfo.ModTime(),
			IsDirectory: entry.IsDir(),
		}

		if entry.IsDir() {
			dirCount++
			// 递归处理子目录
			if subNode, err := buildFileTree(childPath, entry.Name()); err == nil {
				childNode = subNode
				fileCount += subNode.FileCount
				dirCount += subNode.DirCount
				totalSize += subNode.Size
			}
		} else {
			fileCount++
			totalSize += childInfo.Size()
		}

		node.Children = append(node.Children, childNode)
	}

	node.Size = totalSize
	node.FileCount = fileCount
	node.DirCount = dirCount

	return node, nil
}

// GetMirrorUsedSpace 计算镜像已用空间
func GetMirrorUsedSpace(mirrorID uint) (int64, error) {
	var totalSize int64
	err := database.DB.Model(&models.NPMFile{}).
		Where("mirror_id = ?", mirrorID).
		Select("COALESCE(SUM(file_size), 0)").
		Scan(&totalSize).Error

	return totalSize, err
}
