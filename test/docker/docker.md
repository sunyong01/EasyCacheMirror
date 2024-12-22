## 测试 Docker

```bash
# 配置镜像
docker run -it --rm \
  --add-host registry-1.docker.io:192.168.0.124 \
  alpine:latest sh -c "\
  echo '{\"registry-mirrors\": [\"http://192.168.0.124:8080/docker\"]}' > /etc/docker/daemon.json && \
  docker pull nginx:latest"

# 拉取不同仓库的镜像
docker run -it --rm \
  --add-host registry-1.docker.io:192.168.0.124 \
  alpine:latest sh -c "\
  docker pull mysql:8 && \
  docker pull redis:alpine"
``` 