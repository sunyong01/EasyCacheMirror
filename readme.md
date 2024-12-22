# 内网软件源缓存服务

该项目提供了在内网环境下搭建多种软件源缓存的解决方案。

## 项目背景

在内网环境中，我们经常遇到以下问题：
- 公网软件源经常会限制短时间内的请求次数
- 公网软件源的访问速度不稳定
- 现有的解决方案（如 Nexus）过于复杂且资源占用大

本项目旨在提供一个轻量级的解决方案，专注于软件源的缓存功能。

## 特性

✨ **核心功能**
- 支持多种软件源缓存（当前支持 NPM、Maven）其他类型的软件源目前会直接转发请求到上游镜像源
- HTTP 代理支持
- 缓存容量配额管理
- 自动转发非下载请求
- 提供 Web UI 界面
  - 查看缓存使用情况
  - 快捷复制镜像源 URL

🚧 **暂不支持的功能**
- Docker 镜像源缓存
- Go 语言镜像源（建议使用 GoProxy.cn）
- 需要鉴权的上级镜像源
- 多上游镜像源支持（计划支持 Fallback 机制）

## 快速开始

### Docker 部署
1. 从 Release 或 Github Actions 下载预构建镜像
2. 运行以下命令：
```bash
docker-compose up -d
```
- 默认端口：8080
- 可通过 docker-compose.yml 修改端口映射

### 配置持久化
如需配置缓存持久化，在 docker-compose.yml 中添加：
```yaml
volumes:
   - ./data:/app/data
```
### 测试
使用test文件夹下对应的markdown中的脚本进行测试。
  - 目前可以缓存的源包括： 
  - -  NPM
  - -  Maven
  - 目前向上转发的源包括:
  - -  PyPI
  - -  Go
  - -  Ruby
  - -  R(CRAN)
  - -  Conda
  - 目前不可用的
  - -  cargo
  - -  docker


正在补充更多测试用例。

### 进一步
计划添加功能：
  - 常用软件安装包缓存。例如Node.js JDK等
  - 定时自动清理缓存 
  - 添加更多测试用例


## 贡献指南

如果你有需要，欢迎提交 Issue 或 Pull Request。某些功能我暂时没有需求，因此没有考虑到。如果你能提出一些建议我会很乐意改进。

## 许可证

本项目采用 [MIT 许可证](https://opensource.org/licenses/MIT)。

```text
MIT License

Copyright (c) 2024 Present

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.