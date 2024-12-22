package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"easyCacheMirror/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() error {
	// 确保data目录存在
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatal("创建数据目录失败:", err)
	}

	dbPath := filepath.Join(dataDir, "config.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}

	// 自动迁移数据库结构
	err = db.AutoMigrate(
		&models.Mirror{},
		&models.MavenFile{},
		&models.NPMFile{},
	)
	if err != nil {
		log.Fatal("数据库迁移失败:", err)
	}

	DB = db
	fmt.Println("数据库初始化成功")

	return nil
}

// GetMirrorUsedSpace 计算镜像已用空间
func GetMirrorUsedSpace(mirrorID uint) (int64, error) {
	var totalSize int64
	err := DB.Model(&models.NPMFile{}).
		Where("mirror_id = ?", mirrorID).
		Select("COALESCE(SUM(file_size), 0)").
		Scan(&totalSize).Error
	if err != nil {
		return 0, err
	}

	// 计算 Maven 文件大小
	var mavenSize int64
	err = DB.Model(&models.MavenFile{}).
		Where("mirror_id = ?", mirrorID).
		Select("COALESCE(SUM(file_size), 0)").
		Scan(&mavenSize).Error

	totalSize += mavenSize

	return totalSize, err
}
