export type Role = 'admin' | 'user' | 'restricted'

export interface User {
  id: string
  username: string
  role: Role
}

export interface Author {
  id: string
  name: string
  sort_name: string
  role?: string
  order?: number
}

export interface Series {
  id: string
  name: string
  sort_name: string
  index?: number
}

export interface Genre {
  code: string
  name_ru: string
  name_en: string
  category_ru: string
  category_en: string
}

export interface BookFile {
  id: string
  book_id: string
  format: string
  file_path: string
  archive_inner_path?: string
  file_size: number
  sha256: string
  created_at: string
}

export interface Book {
  id: string
  title: string
  original_title?: string
  annotation?: string
  language?: string
  published_date?: string
  publisher?: string
  isbn?: string
  cover_cached: boolean
  created_at: string
  updated_at: string
  authors?: Author[]
  series?: Series[]
  genres?: Genre[]
  files?: BookFile[]
}

export interface BookListResponse {
  items: Book[]
  total: number
  page: number
  per_page: number
  total_pages: number
}

export interface ReadProgress {
  user_id: string
  book_id: string
  format: string
  progress_percent: number
  position: string
  updated_at?: string
}

export interface QuarantineItem {
  id: string
  file_path: string
  file_size: number
  sha256: string
  format: string
  parsed_title: string
  parsed_authors: string
  existing_book_id?: string
  conflict_type: string
  created_at: string
}

export interface QuarantineListResponse {
  items: QuarantineItem[]
  total: number
  page: number
  per_page: number
  total_pages: number
}

export interface ProcessResult {
  status: 'imported' | 'format_attached' | 'quarantined' | 'skipped'
  book_id?: string
  existing_book_id?: string
  conflict_type?: string
  quarantine_id?: string
  message?: string
  source_file: string
  library_file?: string
  sha256?: string
}
