<template>
  <n-config-provider>
    <n-message-provider>
      <n-dialog-provider>
        <n-layout>
          <n-layout-header bordered>
            <div class="header-content">
              <h2>EasyCacheMirror</h2>
            </div>
          </n-layout-header>
          <n-layout has-sider>
            <n-layout-sider
              bordered
              collapse-mode="width"
              :collapsed-width="64"
              :width="240"
              show-trigger
              @collapse="collapsed = true"
              @expand="collapsed = false"
            >
              <n-menu
                v-model:value="activeKey"
                :collapsed="collapsed"
                :options="menuOptions"
                :collapsed-width="64"
                :collapsed-icon-size="22"
              />
            </n-layout-sider>
            <n-layout-content content-style="padding: 24px;">
              <keep-alive>
                <component :is="currentComponent" />
              </keep-alive>
            </n-layout-content>
          </n-layout>
        </n-layout>
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { ref, computed, h } from 'vue'
import {
  NConfigProvider,
  NLayout,
  NLayoutHeader,
  NLayoutContent,
  NLayoutSider,
  NMenu,
  NMessageProvider,
  NDialogProvider,
  NIcon,
  useMessage
} from 'naive-ui'
import { HomeOutline, ServerOutline, FolderOutline } from '@vicons/ionicons5'
import DashBoard from './components/DashBoard.vue'
import MirrorList from './components/MirrorList.vue'
import Storage from './components/Storage.vue'

const activeKey = ref('dashboard')
const collapsed = ref(false)

// 使用计算属性来处理组件切换
const currentComponent = computed(() => {
  switch (activeKey.value) {
    case 'dashboard':
      return DashBoard
    case 'mirror-list':
      return MirrorList
    case 'storage':
      return Storage
    default:
      return DashBoard
  }
})

// 使用渲染函数来创建图标
const renderIcon = (icon: any) => {
  return () => h(NIcon, null, { default: () => h(icon) })
}

const menuOptions = [
  {
    label: '控制面板',
    key: 'dashboard',
    icon: renderIcon(HomeOutline)
  },
  {
    label: '镜像列表',
    key: 'mirror-list',
    icon: renderIcon(ServerOutline)
  },
  {
    label: '存储管理',
    key: 'storage',
    icon: renderIcon(FolderOutline)
  }
]
</script>

<style scoped>
.header-content {
  padding: 0 24px;
  height: 64px;
  display: flex;
  align-items: center;
}

.header-content h2 {
  margin: 0;
  font-size: 18px;
}

:deep(.n-layout-sider) {
  background: #fff;
}

:deep(.n-menu) {
  height: calc(100vh - 64px);
}
</style>

<style>
.n-layout-content {
  min-height: calc(100vh - 64px);
}
</style> 