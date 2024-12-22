## 测试 Cargo

```bash
# 配置 Cargo 镜像
docker run -it --rm rust:latest bash -c "\
  mkdir -p ~/.cargo && \
  echo '[source.crates-io]' > ~/.cargo/config && \
  echo 'replace-with = \"mirror\"' >> ~/.cargo/config && \
  echo 'protocol = \"sparse\"' >> ~/.cargo/config && \
  echo '[source.mirror]' >> ~/.cargo/config && \
  echo 'registry = \"http://192.168.0.124:8080/cargo\"' >> ~/.cargo/config && \
  echo '[registries.crates-io]' >> ~/.cargo/config && \
  echo 'index = \"http://192.168.0.124:8080/cargo\"' >> ~/.cargo/config && \
  echo 'protocol = \"sparse\"' >> ~/.cargo/config && \
  cargo install ripgrep"

# 创建新项目测试
docker run -it --rm rust:latest bash -c "\
  export CARGO_HOME=/root/.cargo && \
  export RUSTUP_DIST_SERVER=http://192.168.0.124:8080/cargo && \
  export RUSTUP_UPDATE_ROOT=http://192.168.0.124:8080/cargo/rustup && \
  cargo new test-project && \
  cd test-project && \
  cargo add tokio"
``` 