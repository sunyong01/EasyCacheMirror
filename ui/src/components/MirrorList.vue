<template>
  <div class="mirror-list">
    <n-card>
      <div class="operation-bar">
        <n-space>
          <n-statistic label="已用总空间">
            {{ totalStorageText }}
          </n-statistic>
          <n-divider vertical />
          <n-button type="primary" @click="handleAddMirror">
            <template #icon>
              <n-icon><add /></n-icon>
            </template>
            添加新镜像
          </n-button>
        </n-space>
      </div>
      
      <n-data-table
        :columns="columns"
        :data="mirrorData"
        :pagination="pagination"
        :bordered="false"
        :loading="loading"
        striped
      />
    </n-card>

    <!-- 编辑/添加镜像对话框 -->
    <n-modal
      v-model:show="showEditModal"
      :mask-closable="false"
      :title="editingMirror ? '编辑镜像' : '添加镜像'"
      preset="dialog"
      positive-text="确认"
      negative-text="取消"
      style="width: 600px"
      @positive-click="handleModalConfirm"
      @negative-click="closeEditModal"
    >
      <n-form
        ref="formRef"
        :model="formModel"
        :rules="rules"
        label-placement="left"
        label-width="120"
        require-mark-placement="right-hanging"
      >
        <n-form-item label="镜像名称" path="name">
          <n-input v-model:value="formModel.name" placeholder="请输入镜像名称" />
        </n-form-item>
        <n-form-item label="镜像类型" path="type">
          <n-select
            v-model:value="formModel.type"
            :options="mirrorTypeOptions"
            placeholder="请选择镜像类型"
          />
        </n-form-item>
        <n-form-item label="上游源地址" path="upstreamUrl">
          <n-input v-model:value="formModel.upstreamUrl" placeholder="请输入上游源地址" />
        </n-form-item>
        <n-form-item label="使用代理" path="useProxy">
          <n-switch v-model:value="formModel.useProxy" />
        </n-form-item>
        <n-form-item
          label="代理地址"
          path="proxyUrl"
          :show="formModel.useProxy"
          :required="formModel.useProxy"
        >
          <n-input v-model:value="formModel.proxyUrl" placeholder="请输入HTTP代理地址" />
        </n-form-item>
        <n-form-item label="最大容量" path="maxSize">
          <div class="size-input-container">
            <n-input-number
              v-model:value="formModel.displaySize"
              :min="1"
              :show-button="false"
              @update:value="handleSizeChange"
            />
            <n-select
              v-model:value="sizeUnit"
              :options="sizeUnitOptions"
              class="size-unit-select"
              @update:value="handleSizeChange"
            />
          </div>
        </n-form-item>
        <n-form-item label="Blob存储位置" path="blobPath">
          <n-input
            v-model:value="formModel.blobPath"
            placeholder="请输入Blob存储路径"
          />
        </n-form-item>
        <n-form-item label="访问地址" path="accessUrl">
          <n-input
            v-model:value="formModel.accessUrl"
            placeholder="请输入访问地址"
          />
        </n-form-item>
        <n-form-item label="缓存时间" path="cacheTime">
          <n-input-number
            v-model:value="formModel.cacheTime"
            :min="1"
            :max="525600"
            placeholder="请输入缓存时间"
          >
            <template #suffix>分钟</template>
          </n-input-number>
        </n-form-item>
        <n-form-item label="向外服务地址" prop="serviceUrl">
          <n-input 
            v-model:value="formModel.serviceUrl" 
            placeholder="例如: http://192.168.0.124:8080"
          >
            <template #append>
              <n-tooltip 
                content="镜像对外提供服务的基础URL，用于替换包中的下载地址" 
                placement="top"
              >
                <n-icon><Help /></n-icon>
              </n-tooltip>
            </template>
          </n-input>
        </n-form-item>
      </n-form>
    </n-modal>

    <!-- 删除确认对话框 -->
    <n-modal
      v-model:show="showDeleteModal"
      preset="dialog"
      title="确认删除"
      content="确定要删除这个镜像吗？此操作不可恢复。"
      positive-text="确认"
      negative-text="取消"
      @positive-click="confirmDelete"
      @negative-click="closeDeleteModal"
    />
  </div>
</template>

<script setup lang="ts">
import { h, ref, watch, onMounted, computed } from 'vue'
import {
  NCard,
  NButton,
  NDataTable,
  NIcon,
  NTag,
  NTime,
  NModal,
  NForm,
  NFormItem,
  NInput,
  NSelect,
  NInputNumber,
  NSwitch,
  useMessage,
  useDialog,
  type FormRules,
  type FormInst,
  type DataTableColumns,
  NStatistic,
  NDivider,
  NSpace,
  NProgress,
  NTooltip
} from 'naive-ui'
import { Add, Create, TrashBin, Help } from '@vicons/ionicons5'
import { mirrorApi, type Mirror, type MirrorForm } from '../api/mirror'

const pagination = { pageSize: 10 }
const mirrorData = ref<Mirror[]>([])
const loading = ref(false)
const message = useMessage()
const dialog = useDialog()

// 加载镜像列表
const loadMirrors = async () => {
  loading.value = true
  try {
    const response = await mirrorApi.list()
    mirrorData.value = response.data
  } catch (error) {
    message.error('加载镜像列表失败')
    console.error('加载失败:', error)
  } finally {
    loading.value = false
  }
}

// 表单相关
const formRef = ref<FormInst | null>(null)
const showEditModal = ref(false)
const showDeleteModal = ref(false)
const editingMirror = ref<Mirror | null>(null)
const deletingMirror = ref<Mirror | null>(null)

const formModel = ref({
  name: '',
  type: 'NPM',
  upstreamUrl: '',
  useProxy: false,
  proxyUrl: '',
  maxSize: 10,
  sizeUnit: 'GB',
  displaySize: 10,
  blobPath: '',
  accessUrl: '',
  cacheTime: 7,
  serviceUrl: ''
})

const mirrorTypeOptions = [
  { label: 'NPM', value: 'NPM' },
  { label: 'Maven', value: 'Maven' },
  { label: 'PyPI', value: 'PyPI' },
  { label: 'R', value: 'R' },
  { label: 'Go', value: 'Go' },
  { label: 'RubyGems', value: 'RubyGems' },
  { label: 'Conda', value: 'Conda' },
  { label: 'Docker', value: 'Docker' },
  { label: 'Cargo', value: 'Cargo' }
]

const sizeUnitOptions = [
  { label: 'MB', value: 'MB' },
  { label: 'GB', value: 'GB' },
  { label: 'TB', value: 'TB' }
]

const rules: FormRules = {
  name: [
    { required: true, message: '请输入镜像名称' },
    { pattern: /^[a-z0-9-]+$/, message: '只能包含小写字母、数字和连字符' }
  ],
  type: [
    { required: true, message: '请选择镜像类型' }
  ],
  upstreamUrl: [
    { required: true, message: '请输入上游源地址' },
    { type: 'url', message: '请输入有效的URL地址' }
  ],
  proxyUrl: [
    {
      validator: (rule, value) => {
        if (!formModel.value.useProxy) {
          return true
        }
        if (!value) {
          return new Error('请输入代理地址')
        }
        try {
          new URL(value)
          return true
        } catch {
          return new Error('请输入有效的URL地址')
        }
      },
      trigger: ['blur', 'change']
    }
  ],
  maxSize: [
    { required: true, message: '请输入最大容量' },
    { type: 'number', min: 1, message: '容量必须大于0' }
  ],
  blobPath: [
    { required: true, message: '请输入Blob存储位置' }
  ],
  accessUrl: [
    { required: true, message: '请输入访问地址' },
    { 
      validator: (rule, value) => {
        // 允许以 / 开头的相对路径
        if (value.startsWith('/')) {
          return true
        }
        // 如果不是相对路径，则验证是否为有效的URL
        try {
          new URL(value)
          return true
        } catch {
          return new Error('请输入有效的URL地址或以/开头的相对路径')
        }
      },
      trigger: ['blur', 'change']
    }
  ],
  cacheTime: [
    { required: true, message: '请输入缓存时间' },
    { type: 'number', min: 1, max: 525600, message: '缓存时间必须在1-525600分钟之间' }
  ],
  serviceUrl: [
    { required: true, message: '请输入向外服务地址', trigger: 'blur' },
    { type: 'url', message: '请输入有效的URL地址', trigger: 'blur' }
  ]
}

// 默认值配置
const defaultMirrorConfigs = {
  NPM: {
    upstreamUrl: 'https://registry.npmmirror.com',
    accessUrl: '/npm',
    blobPath: '/app/data/npm'
  },
  Maven: {
    upstreamUrl: 'https://maven.aliyun.com/repository/public',
    accessUrl: '/maven',
    blobPath: '/app/data/maven'
  },
  PyPI: {
    upstreamUrl: 'https://mirrors.aliyun.com/pypi',
    accessUrl: '/pypi',
    blobPath: '/app/data/pypi'
  },
  R: {
    upstreamUrl: 'https://mirrors.tuna.tsinghua.edu.cn/CRAN',
    accessUrl: '/r',
    blobPath: '/app/data/r'
  },
  Go: {
    upstreamUrl: 'https://goproxy.cn',
    accessUrl: '/go',
    blobPath: '/app/data/go'
  },
  RubyGems: {
    upstreamUrl: 'https://mirrors.tuna.tsinghua.edu.cn/rubygems',
    accessUrl: '/rubygems',
    blobPath: '/app/data/rubygems'
  },
  Conda: {
    upstreamUrl: 'https://mirrors.tuna.tsinghua.edu.cn/anaconda',
    accessUrl: '/conda',
    blobPath: '/app/data/conda'
  },
  Docker: {
    upstreamUrl: 'https://registry.cn-hangzhou.aliyuncs.com',
    accessUrl: '/docker',
    blobPath: '/app/data/docker'
  },
  Cargo: {
    upstreamUrl: 'https://mirrors.tuna.tsinghua.edu.cn/crates.io-index',
    accessUrl: '/cargo',
    blobPath: '/app/data/cargo'
  }
}

// 监听镜像类型
watch(() => formModel.value.type, (newType) => {
  const defaultConfig = defaultMirrorConfigs[newType as keyof typeof defaultMirrorConfigs]
  if (defaultConfig) {
    if (!editingMirror.value) {
      formModel.value.upstreamUrl = defaultConfig.upstreamUrl
      formModel.value.accessUrl = defaultConfig.accessUrl
      formModel.value.blobPath = defaultConfig.blobPath
    }
  }
})

// 处理函数
const handleAddMirror = () => {
  editingMirror.value = null
  const defaultConfig = defaultMirrorConfigs['NPM']
  formModel.value = {
    name: '',
    type: 'NPM',
    upstreamUrl: defaultConfig.upstreamUrl,
    useProxy: false,
    proxyUrl: '',
    maxSize: 10,
    sizeUnit: 'GB',
    displaySize: 10,
    blobPath: defaultConfig.blobPath,
    accessUrl: defaultConfig.accessUrl,
    cacheTime: 7,
    serviceUrl: ''
  }
  showEditModal.value = true
}

const handleEdit = (row: Mirror) => {
  editingMirror.value = row
  const { size: displaySize } = bytesToDisplaySize(row.maxSize)

  formModel.value = {
    name: row.name,
    type: row.type,
    upstreamUrl: row.upstreamUrl,
    useProxy: row.useProxy,
    proxyUrl: row.proxyUrl || '',
    maxSize: row.maxSize,
    sizeUnit: row.sizeUnit || 'GB',
    displaySize,
    blobPath: row.blobPath,
    accessUrl: row.accessUrl,
    cacheTime: row.cacheTime,
    serviceUrl: row.serviceUrl
  }

  showEditModal.value = true
}

const handleDelete = (row: Mirror) => {
  deletingMirror.value = row
  showDeleteModal.value = true
}

const handleModalConfirm = async () => {
  try {
    await formRef.value?.validate()
    
    // 将显示大小转换为字节
    const maxSizeInBytes = displaySizeToBytes(formModel.value.displaySize, formModel.value.sizeUnit)
    
    const submitData = {
      name: formModel.value.name,
      type: formModel.value.type,
      upstreamUrl: formModel.value.upstreamUrl,
      useProxy: formModel.value.useProxy,
      proxyUrl: formModel.value.useProxy ? formModel.value.proxyUrl : undefined,
      maxSize: maxSizeInBytes,
      sizeUnit: formModel.value.sizeUnit,
      blobPath: formModel.value.blobPath,
      accessUrl: formModel.value.accessUrl,
      cacheTime: formModel.value.cacheTime,
      serviceUrl: formModel.value.serviceUrl
    }

    if (editingMirror.value) {
      await mirrorApi.update(editingMirror.value.id, submitData)
      message.success('镜像更新成功')
    } else {
      await mirrorApi.create(submitData)
      message.success('镜像创建成功')
    }
    
    closeEditModal()
    loadMirrors() // 重新加载列表
    return true
  } catch (error) {
    if (error instanceof Error) {
      message.error(error.message)
    } else {
      message.error('操作失败')
    }
    console.error('操作失败:', error)
    return false
  }
}

const confirmDelete = async () => {
  if (!deletingMirror.value) return
  
  try {
    await mirrorApi.delete(deletingMirror.value.id)
    message.success('镜像删除成功')
    loadMirrors() // 重新加载列表
  } catch (error) {
    message.error('删除失败')
    console.error('删除失败:', error)
  } finally {
    closeDeleteModal()
  }
}

const closeEditModal = () => {
  showEditModal.value = false
  editingMirror.value = null
  const defaultConfig = defaultMirrorConfigs['NPM']
  formModel.value = {
    name: '',
    type: 'NPM',
    upstreamUrl: defaultConfig.upstreamUrl,
    useProxy: false,
    proxyUrl: '',
    maxSize: 10,
    sizeUnit: 'GB',
    displaySize: 10,
    blobPath: defaultConfig.blobPath,
    accessUrl: defaultConfig.accessUrl,
    cacheTime: 7,
    serviceUrl: ''
  }
}

const closeDeleteModal = () => {
  showDeleteModal.value = false
  deletingMirror.value = null
}

// 格式化日期时间
const formatDateTime = (time: string) => {
  return new Date(time).toLocaleString()
}

// 在组件挂载时加载数据
onMounted(() => {
  loadMirrors()
})

// 更新表格列定义中的数据显示
const columns: DataTableColumns<Mirror> = [
  {
    title: '镜像名称',
    key: 'name',
    align: 'left',
    render(row) {
      return h(
        NTag,
        {
          type: 'info',
          bordered: false
        },
        { default: () => row.name }
      )
    }
  },
  {
    title: '镜像类型',
    key: 'type',
    align: 'center'
  },
  {
    title: '存储空间',
    key: 'storage',
    align: 'center',
    render(row) {
      const { size: usedSize, unit: usedUnit } = bytesToDisplaySize(row.usedSpace)
      const { size: maxSize, unit: maxUnit } = bytesToDisplaySize(row.maxSize)
      const percentage = Math.min(100, (row.usedSpace / row.maxSize * 100)).toFixed(1)
      
      return h('div', { class: 'storage-info' }, [
        h(NProgress, {
          type: 'line',
          percentage: Number(percentage),
          indicatorPlacement: 'inside',
          processing: row.usedSpace / row.maxSize > 0.9,
          status: row.usedSpace / row.maxSize > 0.9 ? 'warning' : 'success'
        }),
        h('span', { class: 'storage-text' }, 
          `${usedSize} ${usedUnit} / ${maxSize} ${maxUnit} (${percentage}%)`
        )
      ])
    }
  },
  {
    title: '上次使用时间',
    key: 'lastUsedTime',
    align: 'center',
    render(row) {
      return formatDateTime(row.lastUsedTime)
    }
  },
  {
    title: '缓存时间',
    key: 'cacheTime',
    align: 'center'
  },
  {
    title: '总请求数',
    key: 'request_count',
    align: 'center',
    width: 120,
    render(row) {
      return h(
        NTag,
        {
          type: 'info',
          bordered: false
        },
        { default: () => row.request_count || 0 }
      )
    }
  },
  {
    title: '缓存命中率',
    key: 'hitRate',
    align: 'center',
    width: 120,
    render(row) {
      return h(
        NTag,
        {
          type: 'info',
          bordered: false
        },
        { default: () => calculateHitRate(row) }
      )
    }
  },
  {
    title: '操作',
    key: 'actions',
    align: 'center',
    width: 280,
    render(row) {
      return h(
        NSpace,
        { align: 'center' },
        {
          default: () => [
            h(
              NButton,
              {
                size: 'small',
                quaternary: true,
                type: 'info',
                onClick: () => handleEdit(row)
              },
              { default: () => '编辑' }
            ),
            h(
              NButton,
              {
                size: 'small',
                quaternary: true,
                type: 'error',
                onClick: () => handleDelete(row)
              },
              { default: () => '删除' }
            ),
            h(
              NButton,
              {
                size: 'small',
                quaternary: true,
                type: 'warning',
                onClick: () => handleCleanup(row)
              },
              { default: () => '清理缓存' }
            )
          ]
        }
      )
    }
  }
]

// 添加总空间计算
const totalStorageText = computed(() => {
  const total = mirrorData.value.reduce((sum, mirror) => {
    const size = mirror.usedSpace
    const unit = mirror.sizeUnit
    // 转换为字节
    let bytes = size
    if (unit === 'MB') bytes *= 1024 * 1024
    if (unit === 'GB') bytes *= 1024 * 1024 * 1024
    if (unit === 'TB') bytes *= 1024 * 1024 * 1024 * 1024
    return sum + bytes
  }, 0)

  // 转换为合适的单位
  if (total < 1024 * 1024) return `${(total / 1024).toFixed(2)} KB`
  if (total < 1024 * 1024 * 1024) return `${(total / 1024 / 1024).toFixed(2)} MB`
  if (total < 1024 * 1024 * 1024 * 1024) return `${(total / 1024 / 1024 / 1024).toFixed(2)} GB`
  return `${(total / 1024 / 1024 / 1024 / 1024).toFixed(2)} TB`
})

const sizeUnit = ref('GB')

// 字节转换为显示单位
function bytesToDisplaySize(bytes: number): { size: number; unit: string } {
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let size = bytes
  let unitIndex = 0

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex++
  }

  return {
    size: Number(size.toFixed(2)),
    unit: units[unitIndex]
  }
}

// 显示单位转换为字节
function displaySizeToBytes(size: number, unit: string): number {
  const multipliers = {
    'B': 1,
    'KB': 1024,
    'MB': 1024 * 1024,
    'GB': 1024 * 1024 * 1024,
    'TB': 1024 * 1024 * 1024 * 1024
  }
  return size * multipliers[unit]
}

// 处理容量变化
function handleSizeChange() {
  formModel.value.maxSize = displaySizeToBytes(formModel.value.displaySize, sizeUnit.value)
}

// 显示已用空间
const formatUsedSpace = (usedSpace: number) => {
  const { size, unit } = bytesToDisplaySize(usedSpace)
  return `${size} ${unit}`
}

// 计算缓存命中率
const calculateHitRate = (mirror) => {
  if (!mirror.request_count || mirror.request_count === 0) {
    return '0%'
  }
  const rate = (mirror.hit_count / mirror.request_count * 100).toFixed(1)
  return `${rate}%`
}

// 清理缓存
async function handleCleanup(row: any) {
  try {
    loading.value = true
    await mirrorApi.cleanupMirrorCache(row.id)
    message.success('缓存清理完成')
    // 刷新列表
    await loadMirrors()
  } catch (error: any) {
    message.error(error.message || '清理缓存失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.mirror-list {
  height: 100%;
  width: 100%;
}

:deep(.n-card) {
  height: 100%;
}

:deep(.n-card-content) {
  height: calc(100% - 40px);
  display: flex;
  flex-direction: column;
}

.operation-bar {
  margin-bottom: 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

:deep(.n-data-table) {
  flex: 1;
  min-height: 0;
}

.action-buttons {
  display: inline-flex;
  justify-content: center;
  align-items: center;
  white-space: nowrap;
}

:deep(.action-buttons .n-button) {
  padding: 0 4px;
}

.size-input-container {
  display: flex;
  align-items: center;
  gap: 8px;
}

.size-input-container :deep(.n-input-number) {
  width: 120px;
}

.size-unit-select {
  width: 80px;
}

.storage-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 200px;
}

.storage-text {
  font-size: 12px;
  color: var(--n-text-color-2);
  text-align: center;
}

:deep(.n-progress) {
  width: 100%;
}
</style> 