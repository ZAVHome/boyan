export class ApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.status = status
    this.name = 'ApiError'
  }
}

const TOKEN_KEY = 'boyan_jwt_token'

export const tokenStorage = {
  get: () => localStorage.getItem(TOKEN_KEY),
  set: (token: string) => localStorage.setItem(TOKEN_KEY, token),
  clear: () => localStorage.removeItem(TOKEN_KEY)
}

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const token = tokenStorage.get()
  const headers = new Headers(options.headers || {})

  if (!headers.has('Accept')) {
    headers.set('Accept', 'application/json')
  }

  if (token && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  // Не устанавливаем Content-Type для FormData (браузер сам выставит multipart/form-data boundary)
  if (!(options.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  const response = await fetch(endpoint, {
    ...options,
    headers
  })

  if (!response.ok) {
    let errorMsg = `HTTP ${response.status} ${response.statusText}`
    try {
      const errBody = await response.json()
      if (errBody.error) {
        errorMsg = errBody.error
      }
    } catch {
      // Игнорируем ошибку парсинга JSON тела
    }

    if (response.status === 401) {
      tokenStorage.clear()
    }

    throw new ApiError(errorMsg, response.status)
  }

  if (response.status === 204) {
    return {} as T
  }

  return response.json() as Promise<T>
}

export const api = {
  get: <T>(url: string, params?: Record<string, any>) => {
    let fullUrl = url
    if (params) {
      const sp = new URLSearchParams()
      for (const [k, v] of Object.entries(params)) {
        if (v !== undefined && v !== null && v !== '') {
          sp.append(k, String(v))
        }
      }
      const qs = sp.toString()
      if (qs) {
        fullUrl += (url.includes('?') ? '&' : '?') + qs
      }
    }
    return request<T>(fullUrl, { method: 'GET' })
  },

  post: <T>(url: string, data?: any) => {
    const isFormData = data instanceof FormData
    return request<T>(url, {
      method: 'POST',
      body: isFormData ? data : (data !== undefined ? JSON.stringify(data) : undefined)
    })
  },

  put: <T>(url: string, data?: any) => {
    return request<T>(url, {
      method: 'PUT',
      body: data !== undefined ? JSON.stringify(data) : undefined
    })
  },

  delete: <T>(url: string) => {
    return request<T>(url, { method: 'DELETE' })
  },

  getCoverUrl: (bookId: string) => `/covers/${bookId}`,
  getDownloadUrl: (bookId: string, format: string) => `/api/v1/books/${bookId}/download/${format}`
}
