<template>
  <div class="storage-manager">
    <n-card title="存储管理">
      <n-space vertical>
        <n-space>
          <n-button @click="refreshTree" :loading="loading">
            <template #icon>
              <n-icon><refresh /></n-icon>
            </template>
            刷新
          </n-button>
        </n-space>
        
        <n-tree
          block-line
          :data="treeData"
          :expand-on-click="true"
          :selectable="false"
          :loading="loading"
          :node-props="nodeProps"
          @update:expanded-keys="handleExpandedKeysChange"
        >
          <template #default="{ option: node }">
            <span class="node-label" @contextmenu.prevent="(e) => handleContextMenu(e, node)">
              {{ node.label }}
              <span class="node-meta">
                <template v-if="node.isDirectory">
                  {{ formatSize(node.size) }} | {{ node.fileCount }} 个文件, {{ node.dirCount }} 个目录
                </template>
                <template v-else>
                  {{ formatSize(node.size) }} | {{ formatTime(node.modTime) }}
                </template>
              </span>
            </span>
          </template>
        </n-tree>
      </n-space>
    </n-card>

    <!-- 右键菜单 -->
    <n-dropdown
      trigger="manual"
      :show="showDropdown"
      :options="dropdownOptions"
      :x="dropdownX"
      :y="dropdownY"
      @select="handleDropdownSelect"
      @clickoutside="closeDropdown"
    />

    <!-- 文件详情对话框 -->
    <n-modal
      v-model:show="showDetailModal"
      :title="selectedFile ? '文件详情' : '目录详情'"
      preset="dialog"
      style="width: 500px"
    >
      <n-descriptions bordered>
        <n-descriptions-item label="名称">
          {{ selectedFile?.name || selectedDir?.name }}
        </n-descriptions-item>
        <n-descriptions-item label="路径">
          {{ selectedFile?.path || selectedDir?.path }}
        </n-descriptions-item>
        <n-descriptions-item label="大小">
          {{ formatSize(selectedFile?.size || selectedDir?.size) }}
        </n-descriptions-item>
        <n-descriptions-item label="修改时间">
          {{ formatTime(selectedFile?.modTime || selectedDir?.modTime) }}
        </n-descriptions-item>
        <template v-if="selectedDir">
          <n-descriptions-item label="文件数量">
            {{ selectedDir.fileCount }}
          </n-descriptions-item>
          <n-descriptions-item label="子目录数量">
            {{ selectedDir.dirCount }}
          </n-descriptions-item>
        </template>
      </n-descriptions>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, h, computed, type Component } from 'vue'
import type { TreeOption } from 'naive-ui'
import type { TreeRenderProps } from 'naive-ui/es/tree/src/interface'
import {
  NCard,
  NSpace,
  NButton,
  NTree,
  NIcon,
  NModal,
  NDescriptions,
  NDescriptionsItem,
  useMessage,
  useDialog,
  NDropdown
} from 'naive-ui'
import {
  Refresh,
  Folder,
  FolderOpen,
  Document,
  InformationCircle,
  TrashBin
} from '@vicons/ionicons5'
import { mirrorApi, type FileNode } from '../api/mirror'
import type { DropdownOption } from 'naive-ui'

const loading = ref(false)
const treeData = ref<TreeOption[]>([])
const message = useMessage()
const dialog = useDialog()

// 选中的文件/目录
const selectedFile = ref<FileNode | null>(null)
const selectedDir = ref<FileNode | null>(null)
const showDetailModal = ref(false)

// 将 FileNode 转换为 TreeOption
const convertToTreeOption = (node: FileNode): TreeOption => {
  const { key, ...rest } = node
  return {
    key,
    label: node.name,
    ...rest,
    children: node.children?.map(convertToTreeOption),
    prefix: () => h(NIcon, null, {
      default: () => h(node.isDirectory ? Folder : Document)
    }),
    suffix: () => h('div', { class: 'node-actions' }, [
      h(
        NButton,
        {
          text: true,
          type: 'info',
          size: 'tiny',
          onClick: (e: MouseEvent) => {
            e.stopPropagation()
            showDetails(node)
          }
        },
        { default: () => '详情' }
      )
    ])
  }
}

// 加载存储树
const loadStorageTree = async () => {
  loading.value = true
  try {
    const response = await mirrorApi.getStorageTree()
    treeData.value = response.data.map(convertToTreeOption)
    console.log('Storage tree data:', treeData.value)
  } catch (error) {
    message.error('加载存储树失败')
    console.error('加载失败:', error)
  } finally {
    loading.value = false
  }
}

// 刷新树
const refreshTree = () => {
  loadStorageTree()
}

// 格式化文件大小
const formatSize = (size: number | undefined) => {
  if (size === undefined) return '-'
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(2)} KB`
  if (size < 1024 * 1024 * 1024) return `${(size / 1024 / 1024).toFixed(2)} MB`
  return `${(size / 1024 / 1024 / 1024).toFixed(2)} GB`
}

// 格式化时间
const formatTime = (time: string | Date | undefined) => {
  if (!time) return '-'
  if (typeof time === 'string') {
    return new Date(time).toLocaleString()
  }
  return time.toLocaleString()
}

// 显示详情
const showDetails = (node: FileNode) => {
  if (node.isDirectory) {
    selectedDir.value = node
    selectedFile.value = null
  } else {
    selectedFile.value = node
    selectedDir.value = null
  }
  showDetailModal.value = true
}

// 右键菜单状态
const showDropdown = ref(false)
const dropdownX = ref(0)
const dropdownY = ref(0)
const currentNode = ref<FileNode | null>(null)

// 右键菜单选项
const dropdownOptions = computed(() => {
  if (!currentNode.value) return []
  
  return [
    {
      label: '详情',
      key: 'details',
      icon: renderIcon(InformationCircle)
    }
  ] as DropdownOption[]
})

// 渲染图标辅助函数
function renderIcon(icon: Component) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

// 处理右键菜单
const handleContextMenu = (e: MouseEvent, node: TreeOption & FileNode) => {
  e.stopPropagation()
  e.preventDefault()
  currentNode.value = node
  dropdownX.value = e.clientX
  dropdownY.value = e.clientY
  showDropdown.value = true
}

// 处理菜单选择
const handleDropdownSelect = (key: string) => {
  if (!currentNode.value) return
  
  if (key === 'details') {
    showDetails(currentNode.value)
  }
  closeDropdown()
}

// 关闭右键菜单
const closeDropdown = () => {
  showDropdown.value = false
  currentNode.value = null
}

// 节点属性
const nodeProps = ({ option }: { option: TreeOption }) => {
  const node = option as unknown as FileNode
  return {
    onContextmenu(e: MouseEvent) {
      currentNode.value = node
      showDropdown.value = true
      dropdownX.value = e.clientX
      dropdownY.value = e.clientY
      e.preventDefault()
    }
  }
}

// 添加展开/折叠时的图标更新处理
const handleExpandedKeysChange = (
  _keys: Array<string | number>,
  _option: Array<TreeOption | null>,
  meta: {
    node: TreeOption | null
    action: 'expand' | 'collapse' | 'filter'
  }
) => {
  if (!meta.node?.isDirectory) return
  
  switch (meta.action) {
    case 'expand':
      meta.node.prefix = () => h(NIcon, null, { default: () => h(FolderOpen) })
      break
    case 'collapse':
      meta.node.prefix = () => h(NIcon, null, { default: () => h(Folder) })
      break
  }
}

// 初始加载
loadStorageTree()
</script>

<style scoped>
.storage-manager {
  height: 100%;
  width: 100%;
}

:deep(.n-card) {
  height: 100%;
}

:deep(.n-card-content) {
  height: calc(100% - 40px);
}

:deep(.n-space) {
  height: 100%;
}

:deep(.n-tree) {
  height: 100%;
}

.node-label {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.node-meta {
  color: #666;
  font-size: 0.9em;
  margin-left: 12px;
}

:deep(.n-tree-node-content) {
  padding: 4px 8px;
}

:deep(.n-tree-node-content:hover) {
  background-color: var(--n-node-color-hover);
}

.node-actions {
  display: flex;
  gap: 8px;
  opacity: 0;
  transition: opacity 0.2s;
}

:deep(.n-tree-node-content:hover) .node-actions {
  opacity: 1;
}
</style> 