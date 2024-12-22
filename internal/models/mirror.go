package models

import (
	"time"
)

const (
	DefaultNPMRegistry = "https://registry.npmjs.org"
)

type Mirror struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name" gorm:"uniqueIndex"`
	Type         string    `json:"type"`
	UpstreamURL  string    `json:"upstreamUrl" gorm:"column:upstream_url"`
	UseProxy     bool      `json:"useProxy" gorm:"column:use_proxy"`
	ProxyURL     string    `json:"proxyUrl" gorm:"column:proxy_url"`
	MaxSize      int64     `json:"maxSize" gorm:"column:max_size;comment:最大容量(字节)"`
	BlobPath     string    `json:"blobPath" gorm:"column:blob_path"`
	AccessURL    string    `json:"accessUrl" gorm:"column:access_url"`
	LastUsedTime time.Time `json:"lastUsedTime" gorm:"column:last_used_time"`
	CacheTime    int       `json:"cacheTime" gorm:"column:cache_time;comment:缓存时间(分钟)"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	LastCleanup  time.Time `json:"lastCleanup" gorm:"column:last_cleanup"`
	ServiceURL   string    `json:"serviceUrl" gorm:"column:service_url"`
	HitCount     int64     `json:"hit_count" gorm:"default:0"`     // 缓存命中次数
	RequestCount int64     `json:"request_count" gorm:"default:0"` // 总请求次数
}
