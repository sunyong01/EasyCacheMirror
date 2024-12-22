## 测试 Go

```bash
# 设置 GOPROXY
docker run -it --rm golang:latest bash -c "\
  go env -w GOPROXY=http://192.168.0.124:8080/go,direct && \
  go env -w GOSUMDB=off && \
  go install github.com/gin-gonic/gin@latest"

# 创建新项目测试
docker run -it --rm golang:latest bash -c "\
  export GOPROXY=http://192.168.0.124:8080/go,direct && \
  export GOSUMDB=off && \
  mkdir test-project && \
  cd test-project && \
  go mod init test && \
  go get github.com/gin-gonic/gin"
``` 