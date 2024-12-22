package models

import (
	"time"
)

// MavenFileType 定义文件类型
type MavenFileType string

const (
	MavenFileTypeNormal   MavenFileType = "NORMAL"   // 普通文件(jar、pom等)
	MavenFileTypeMetadata MavenFileType = "METADATA" // maven-metadata.xml
)

// MavenFile 记录Maven文件下载信息
type MavenFile struct {
	ID              uint   `gorm:"primarykey"`
	MirrorID        uint   `gorm:"column:mirror_id;index"`
	RelativePath    string `gorm:"index"` // 相对路径，例如: "org/springframework/spring-core/5.3.9/spring-core-5.3.9.jar"
	FileType        MavenFileType
	FileSize        int64     // 文件大小（字节）
	SavePath        string    // 本地保存路径
	ContentType     string    // HTTP Content-Type
	ContentEncoding string    // 新增：记录压缩编码方式
	IsSnapshot      bool      // 是否为SNAPSHOT版本
	DownloadedAt    time.Time `gorm:"column:downloaded_at"`
	LastUsedTime    time.Time `gorm:"column:last_used_time"`
}
