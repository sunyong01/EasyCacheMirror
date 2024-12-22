## 测试NPM

```bash
docker run -it --rm -e NPM_CONFIG_REGISTRY=http://192.168.0.124:8080/npm node:latest bash -c "npm install vue -g"
```

