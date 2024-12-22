## 测试 PyPI

```bash
# 使用 pip
docker run -it --rm python:3.11 bash -c "\
  pip config set global.index-url http://192.168.0.124:8080/pypi/simple && \
  pip config set global.trusted-host 192.168.0.124 && \
  pip install numpy"

# 使用 pip.conf
docker run -it --rm python:3.11 bash -c "\
  mkdir -p ~/.config/pip && \
  echo '[global]' > ~/.config/pip/pip.conf && \
  echo 'index-url = http://192.168.0.124:8080/pypi/simple' >> ~/.config/pip/pip.conf && \
  echo 'trusted-host = 192.168.0.124' >> ~/.config/pip/pip.conf && \
  pip install numpy"

# 使用 poetry
docker run -it --rm python:3.11 bash -c "\
  pip install poetry && \
  poetry config repositories.mirror http://192.168.0.124:8080/pypi/simple && \
  poetry config certificates.mirror.client-cert false && \
  poetry new test-project && \
  cd test-project && \
  poetry add numpy"
``` 