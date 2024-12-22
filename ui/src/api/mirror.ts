import axios from 'axios'

const baseURL = '/api'

const api = axios.create({
  baseURL,
  timeout: 5000
})

export interface MirrorForm {
  name: string
  type: string
  upstreamUrl: string
  useProxy: boolean
  proxyUrl?: string
  maxSize: number
  sizeUnit: string
  blobPath: string
  accessUrl: string
  cacheTime: number
  serviceUrl: string
}

export interface Mirror extends MirrorForm {
  id: number
  lastUsedTime: string
  createdAt: string
  updatedAt: string
  lastCleanup: string
  hit_count: number
  request_count: number
  usedSpace: number
}

// 添加文件节点接口定义
export interface FileNode {
  key: string
  name: string
  path: string
  size: number
  modTime: string
  isDirectory: boolean
  children?: FileNode[]
  fileCount?: number
  dirCount?: number
}

// 简化的镜像信息接口
export interface SimpleMirror {
  id: number
  type: string
  access_point: string
}

// 配置指南接口
export interface ConfigGuide {
  tool: string
  command: string
}

export const mirrorApi = {
  // 获取镜像列表
  list: () => {
    return api.get<Mirror[]>('/mirrors')
  },

  // 创建镜像
  create: (data: MirrorForm) => {
    return api.post<Mirror>('/mirrors', data)
  },

  // 更新镜像
  update: (id: number, data: MirrorForm) => {
    return api.put<Mirror>(`/mirrors/${id}`, data)
  },

  // 删除镜像
  delete: (id: number) => {
    return api.delete(`/mirrors/${id}`)
  },

  // 获取存储树
  getStorageTree() {
    return api.get<FileNode[]>('/storage')
  },

  // 删除存储项
  deleteStorageItem(path: string) {
    return api.delete(`/storage?path=${encodeURIComponent(path)}`)
  },

  // 清理镜像缓存
  cleanupMirrorCache(id: number) {
    return api.post(`/mirrors/${id}/cleanup`)
  },

  // 获取简化的镜像列表
  getSimpleMirrors: () => {
    return api.get<SimpleMirror[]>('/mirrors/simple')
  }
} 