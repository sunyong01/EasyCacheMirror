package cache

import (
	"easyCacheMirror/internal/logger"
	"easyCacheMirror/internal/models"
	"strings"
	"sync"

	"go.uber.org/zap"
)

type MirrorCache struct {
	mirrors map[uint]*models.Mirror
	mutex   sync.RWMutex
}

var (
	mirrorCache *MirrorCache
	mirrorOnce  sync.Once
)

func GetMirrorCache() *MirrorCache {
	mirrorOnce.Do(func() {
		mirrorCache = &MirrorCache{
			mirrors: make(map[uint]*models.Mirror),
		}
	})
	return mirrorCache
}

func (mc *MirrorCache) Set(mirror *models.Mirror) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	log := logger.GetLogger()
	log.Debug("设置镜像缓存",
		zap.Uint("id", mirror.ID),
		zap.String("type", mirror.Type),
		zap.String("access_url", mirror.AccessURL),
	)

	mc.mirrors[mirror.ID] = mirror
}

func (mc *MirrorCache) Get(path string) *models.Mirror {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	// 规范化路径
	path = strings.Trim(path, "/")
	path = strings.TrimPrefix(path, "/")

	for _, mirror := range mc.mirrors {
		// 规范化镜像路径
		accessURL := strings.TrimPrefix(mirror.AccessURL, "/")
		if strings.HasPrefix(path, accessURL) {
			return mirror
		}
	}
	return nil
}

func (mc *MirrorCache) Remove(mirror *models.Mirror) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	delete(mc.mirrors, mirror.ID)
}

func (mc *MirrorCache) Clear() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.mirrors = make(map[uint]*models.Mirror)
}

// GetAll 获取所有缓存的镜像
func (c *MirrorCache) GetAll() []*models.Mirror {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	log := logger.GetLogger()
	log.Debug("获取所有镜像缓存",
		zap.Int("count", len(c.mirrors)),
	)

	mirrors := make([]*models.Mirror, 0, len(c.mirrors))
	for _, mirror := range c.mirrors {
		mirrors = append(mirrors, mirror)
	}
	return mirrors
}
