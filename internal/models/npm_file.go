package models

import (
	"time"
)

// NPMFileType 定义文件类型
type NPMFileType string

const (
	NPMFileTypeJSON    NPMFileType = "JSON"    // 包元数据
	NPMFileTypeTarball NPMFileType = "TARBALL" // 包文件
)

// NPMFile 记录NPM文件下载信息
type NPMFile struct {
	ID           uint   `gorm:"primarykey"`
	MirrorID     uint   `gorm:"column:mirror_id;index"`
	PackageID    string `gorm:"index"` // 包名，例如: "react"
	Version      string // 版本号，例如: "17.0.2"
	FileName     string // 文件名
	FileType     NPMFileType
	FileSize     int64     // 文件大小（字节）
	SavePath     string    // 本地保存路径
	Integrity    string    // integrity 校验值，例如: sha512-xxx
	Shasum       string    // shasum 校验值，例如: xxx
	DownloadedAt time.Time `gorm:"column:downloaded_at;comment:从上游获取的时间"`
	LastUsedTime time.Time `gorm:"column:last_used_time"`
}
