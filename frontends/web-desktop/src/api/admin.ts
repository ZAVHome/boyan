import { api } from './client'
import type { Book, Role } from './types'

export interface StorageStats {
  total_books: number
  total_authors: number
  total_series: number
  total_files: number
  total_bytes: number
  format_counts: Record<string, number>
}

export interface HostMetrics {
  alloc_bytes: number
  total_alloc_bytes: number
  sys_bytes: number
  heap_alloc_bytes: number
  heap_sys_bytes: number
  num_gc: number
  num_goroutine: number
  num_cpu: number
  go_version: string
  os: string
  arch: string
  uptime_seconds: number
  disk_total_bytes: number
  disk_free_bytes: number
  disk_used_bytes: number
  disk_used_percent: number
}

export interface DatabaseMetrics {
  db_size_bytes: number
  wal_size_bytes: number
  db_path: string
  fts_count: number
}

export interface CacheMetrics {
  cache_dir: string
  total_files: number
  size_bytes: number
}

export interface TelegramStatus {
  enabled: boolean
  has_token: boolean
  allowed_users: number
}

export interface LogEntry {
  timestamp: string
  level: string
  message: string
  attrs?: Record<string, any>
}

export interface AdminUser {
  id: string
  username: string
  role: Role
  is_active: boolean
  created_at: string
}

export interface UserListResponse {
  users: AdminUser[]
  total: number
}

export interface AdminTask {
  id: string
  type: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'
  progress: number
  processed_count: number
  total_count: number
  current_item?: string
  errors?: string[]
  message?: string
  created_at: string
  started_at?: string
  finished_at?: string
}

export interface AuthorInput {
  name: string
  role?: string
  order?: number
}

export interface SeriesInput {
  name: string
  index?: number
}

export interface UpdateBookPayload {
  title: string
  original_title?: string
  annotation?: string
  language?: string
  publisher?: string
  published_date?: string
  isbn?: string
  authors: AuthorInput[]
  series: SeriesInput[]
  genres: string[]
}

export interface BatchBookPayload {
  book_ids: string[]
  action: 'delete' | 'set_genre' | 'set_series' | 'regenerate_cover'
  delete_files?: boolean
  genre_code?: string
  series_name?: string
}

export interface BatchResult {
  success_count: number
  error_count: number
  errors?: string[]
}

export const adminApi = {
  // Dashboard & Metrics
  getStats: () => api.get<StorageStats>('/api/v1/admin/dashboard/stats'),
  getHostMetrics: () => api.get<HostMetrics>('/api/v1/admin/system/host'),
  getDatabaseMetrics: () => api.get<DatabaseMetrics>('/api/v1/admin/system/database'),
  checkpointDatabase: () => api.post<{ message: string }>('/api/v1/admin/system/database/checkpoint'),
  getCacheMetrics: () => api.get<CacheMetrics>('/api/v1/admin/system/cache'),
  purgeCache: () => api.post<{ message: string; deleted_files: number }>('/api/v1/admin/system/cache/purge'),
  getLogs: (params?: { limit?: number; level?: string }) => api.get<{ entries: LogEntry[]; total: number }>('/api/v1/admin/system/logs', params),
  getTelegramStatus: () => api.get<TelegramStatus>('/api/v1/admin/system/telegram'),

  // Users
  listUsers: (params?: { q?: string; role?: string; is_active?: boolean; limit?: number; offset?: number }) =>
    api.get<UserListResponse>('/api/v1/admin/users', params),
  createUser: (payload: { username: string; password: string; role: Role; is_active: boolean }) =>
    api.post<AdminUser>('/api/v1/admin/users', payload),
  getUser: (id: string) => api.get<AdminUser>(`/api/v1/admin/users/${id}`),
  updateUser: (id: string, payload: { role: Role; is_active: boolean }) =>
    api.put<{ message: string }>(`/api/v1/admin/users/${id}`, payload),
  updatePassword: (id: string, password: string) =>
    api.put<{ message: string }>(`/api/v1/admin/users/${id}/password`, { password }),
  deleteUser: (id: string) => api.delete<{ message: string }>(`/api/v1/admin/users/${id}`),

  // Books Curation
  listBooks: (params?: { q?: string; limit?: number; offset?: number }) =>
    api.get<{ books: Book[]; total: number; limit: number; offset: number }>('/api/v1/admin/books', params),
  updateBook: (id: string, payload: UpdateBookPayload) =>
    api.put<Book>(`/api/v1/admin/books/${id}`, payload),
  deleteBook: (id: string, deleteFiles: boolean) =>
    api.delete<{ message: string; deleted_from_disk_count: number }>(`/api/v1/admin/books/${id}?delete_files=${deleteFiles}`),
  regenerateCover: (id: string) =>
    api.post<{ message: string }>(`/api/v1/admin/books/${id}/cover`),
  batchAction: (payload: BatchBookPayload) =>
    api.post<BatchResult>('/api/v1/admin/books/batch', payload),

  // Tasks & Scanner
  runScan: (target: 'watch_dir' | 'library_dir') =>
    api.post<AdminTask>('/api/v1/admin/scanner/run', { target }),
  repairFB2: () =>
    api.post<AdminTask>('/api/v1/admin/storage/repair-fb2'),
  listTasks: (limit = 20) =>
    api.get<{ tasks: AdminTask[]; total: number }>(`/api/v1/admin/tasks?limit=${limit}`),
  getTask: (id: string) =>
    api.get<AdminTask>(`/api/v1/admin/tasks/${id}`),
  cancelTask: (id: string) =>
    api.post<{ message: string }>(`/api/v1/admin/tasks/${id}/cancel`),

  // Settings
  getSettings: () => api.get<any>('/api/v1/admin/settings'),
  updateSettings: (settings: any) => api.put<{ message: string }>('/api/v1/admin/settings', settings),
  reloadServices: () => api.post<{ message: string }>('/api/v1/admin/services/reload')
}
