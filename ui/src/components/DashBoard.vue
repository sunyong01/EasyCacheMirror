<template>
  <n-config-provider :hljs="hljs">
    <div class="dashboard-container">
      <n-space vertical>
        <n-card v-for="mirror in mirrors" :key="mirror.id" :title="mirror.type + ' 配置指南'" class="mirror-guide">
          <n-tabs type="segment">
            <n-tab-pane
              v-for="guide in getConfigGuides(mirror)"
              :key="guide.tool"
              :name="guide.tool"
              :tab="guide.tool"
            >
              <n-card size="small" class="command-card">
                <template #header-extra>
                  <n-button text type="primary" @click="copyCommand(guide.command)">
                    复制
                  </n-button>
                </template>
                <n-code
                  :code="guide.command"
                  :language="getLanguage(guide.tool)"
                  :word-wrap="true"
                  :show-line-numbers="true"
                />
              </n-card>
            </n-tab-pane>
          </n-tabs>
        </n-card>
      </n-space>
    </div>
  </n-config-provider>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NCard, NTabs, NTabPane, NButton, NSpace, NCode, NConfigProvider, useMessage } from 'naive-ui'
import { mirrorApi, type SimpleMirror } from '../api/mirror'

// 按需引入 highlight.js
import hljs from 'highlight.js/lib/core'
import xml from 'highlight.js/lib/languages/xml'
import groovy from 'highlight.js/lib/languages/groovy'
import yaml from 'highlight.js/lib/languages/yaml'
import ini from 'highlight.js/lib/languages/ini'
import r from 'highlight.js/lib/languages/r'
import bash from 'highlight.js/lib/languages/bash'
import 'highlight.js/styles/github-dark.css'

// 注册语言
hljs.registerLanguage('xml', xml)
hljs.registerLanguage('groovy', groovy)
hljs.registerLanguage('yaml', yaml)
hljs.registerLanguage('ini', ini)
hljs.registerLanguage('r', r)
hljs.registerLanguage('bash', bash)

const message = useMessage()
const mirrors = ref<SimpleMirror[]>([])

// 获取镜像列表
onMounted(async () => {
  try {
    const response = await mirrorApi.getSimpleMirrors()
    mirrors.value = response.data
  } catch (error) {
    console.error('获取镜像列表失败:', error)
  }
})

// 根据工具类型返回对应的语言高亮
function getLanguage(tool: string): string {
  switch (tool) {
    case 'settings.xml':
      return 'xml'
    case 'Gradle':
      return 'groovy'
    case 'pip.conf':
    case '.gemrc':
    case '.condarc':
      return 'yaml'
    case 'config.toml':
    case '.gitconfig':
      return 'ini'
    case '.Rprofile':
    case 'Rprofile.site':
    case 'R':
      return 'r'
    default:
      return 'bash'
  }
}

// 生成配置指南
function getConfigGuides(mirror: SimpleMirror): { tool: string; command: string }[] {
  switch (mirror.type) {
    case 'NPM':
      return [
        {
          tool: 'npm',
          command: `npm config set registry ${mirror.access_point}`
        },
        {
          tool: 'pnpm',
          command: `pnpm config set registry ${mirror.access_point}`
        },
        {
          tool: 'yarn',
          command: `yarn config set registry ${mirror.access_point}`
        }
      ]
    case 'Maven':
      return [
        {
          tool: 'settings.xml',
          command: `<mirror>
    <id>mirror</id>
    <mirrorOf>central</mirrorOf>
    <name>mirror</name>
    <url>${mirror.access_point}</url>
</mirror>`
        },
        {
          tool: 'Gradle',
          command: `repositories {
    maven {
        url "${mirror.access_point}"
    }
}`
        }
      ]
    case 'PyPI':
      return [
        {
          tool: 'pip',
          command: `pip config set global.index-url ${mirror.access_point}/simple && \
pip config set global.trusted-host ${new URL(mirror.access_point).hostname}`
        },
        {
          tool: 'pip.conf',
          command: `[global]
index-url = ${mirror.access_point}/simple
trusted-host = ${new URL(mirror.access_point).hostname}`
        },
        {
          tool: 'poetry',
          command: `poetry config repositories.mirror ${mirror.access_point}/simple && \
poetry config certificates.mirror.client-cert false`
        }
      ]
    case 'RubyGems':
      return [
        {
          tool: 'gem',
          command: `gem sources --add ${mirror.access_point} --remove https://rubygems.org/`
        },
        {
          tool: 'bundler',
          command: `bundle config mirror.https://rubygems.org ${mirror.access_point}`
        },
        {
          tool: '.gemrc',
          command: `---
:backtrace: false
:bulk_threshold: 1000
:sources:
- ${mirror.access_point}
:update_sources: true
:verbose: true`
        }
      ]
    case 'Conda':
      return [
        {
          tool: 'conda',
          command: `conda config --add channels ${mirror.access_point}/pkgs/main
conda config --set show_channel_urls yes
conda config --set default_channels ${mirror.access_point}/pkgs/main`
        },
        {
          tool: '.condarc',
          command: `channels:
  - ${mirror.access_point}/pkgs/main
show_channel_urls: true
default_channels:
  - ${mirror.access_point}/pkgs/main
custom_channels:
  conda-forge: ${mirror.access_point}/cloud
  pytorch: ${mirror.access_point}/cloud`
        },
        {
          tool: 'mamba',
          command: `mamba config --add channels ${mirror.access_point}/pkgs/main
mamba config --set show_channel_urls yes
mamba config --set default_channels ${mirror.access_point}/pkgs/main`
        }
      ]
    case 'Cargo':
      return [
        {
          tool: 'cargo',
          command: `# 配置 crates.io 源
cargo config set source.crates-io.replace-with mirror
cargo config set source.mirror.registry "sparse+${mirror.access_point}"`
        },
        {
          tool: 'config.toml',
          command: `# 编辑 $CARGO_HOME/config.toml 文件
[source.crates-io]
replace-with = "mirror"

[source.mirror]
registry = "sparse+${mirror.access_point}"`
        },
        {
          tool: 'env',
          command: `# 设置环境变量
export CARGO_HOME="${HOME}/.cargo"
export RUSTUP_DIST_SERVER=${mirror.access_point}
export RUSTUP_UPDATE_ROOT=${mirror.access_point}/rustup`
        }
      ]
    case 'Go':
      const hostname = new URL(mirror.access_point).hostname
      return [
        {
          tool: 'go',
          command: `go env -w GOPROXY=${mirror.access_point},direct
go env -w GOSUMDB=off`
        },
        {
          tool: 'env',
          command: `# 设置环境变量
export GOPROXY=${mirror.access_point},direct
export GOSUMDB=off
export GOPRIVATE=${hostname}`
        },
        {
          tool: '.gitconfig',
          command: `[url "${mirror.access_point}"]
    insteadOf = https://proxy.golang.org
    insteadOf = https://goproxy.io
    insteadOf = https://goproxy.cn`
        }
      ]
    case 'R':
      return [
        {
          tool: 'R',
          command: `# 在 R 控制台中设置
options(repos = c(CRAN = "${mirror.access_point}"))
# 或者使用命令行
R -e 'options(repos = c(CRAN = "${mirror.access_point}"))'`
        },
        {
          tool: '.Rprofile',
          command: `# 用户级配置文件 ~/.Rprofile
local({
  r <- getOption("repos")
  r["CRAN"] <- "${mirror.access_point}"
  options(repos = r)
})

# 设置下载选项
options(
  download.file.method = "libcurl",
  pkgType = "both"
)`
        },
        {
          tool: 'Rprofile.site',
          command: `# 系统级配置文件 R_HOME/etc/Rprofile.site
local({
  # 设置 CRAN 镜像
  r <- getOption("repos")
  r["CRAN"] <- "${mirror.access_point}"
  options(repos = r)
  
  # Bioconductor 镜像设置（如果支持）
  options(BioC_mirror = "${mirror.access_point}/bioconductor")
  
  # 其他常用设置
  options(
    download.file.method = "libcurl",
    pkgType = "both",
    install.packages.check.source = "yes",
    install.packages.compile.from.source = "interactive"
  )
})`
        }
      ]
    default:
      return []
  }
}

// 复制命令
async function copyCommand(command: string) {
  try {
    await navigator.clipboard.writeText(command)
    message.success('复制成功')
  } catch (err) {
    message.error('复制失败')
  }
}
</script>

<style scoped>
.dashboard-container {
  padding: 20px;
}

.mirror-guide {
  max-width: 800px;
  margin: 0 auto;
}

.command-card {
  margin-top: 8px;
}

/* 调整代码容器样式 */
:deep(.n-code) {
  max-width: 100%;
}
</style> 