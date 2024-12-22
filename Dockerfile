FROM alpine:latest

# 安装必要的依赖
RUN apk add --no-cache ca-certificates tzdata

# 设置时区为上海
ENV TZ=Asia/Shanghai

# 创建工作目录
WORKDIR /app

# 复制二进制文件和前端构建产物
COPY easyCacheMirror /app/
COPY dist/ /app/dist/

# 创建数据目录
RUN mkdir -p /app/data

# 暴露端口
EXPOSE 8080

# 设置入口点
ENTRYPOINT ["/app/easyCacheMirror"]
