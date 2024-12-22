package main

import (
	"log"

	"easyCacheMirror/internal/database"
	"easyCacheMirror/internal/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化数据库
	database.InitDB()

	r := gin.Default()

	// 设置路由
	routes.SetupRoutes(r)

	// 启动服务器
	if err := r.Run(":8080"); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}
