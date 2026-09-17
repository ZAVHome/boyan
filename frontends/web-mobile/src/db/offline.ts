import { get, set, del, values, createStore } from 'idb-keyval'

export interface OfflineBook {
  id: string
  title: string
  author: string
  format: string
  coverUrl?: string
  blob: Blob
  savedAt: number
  size: number
  progress?: {
    position: string
    percentage: number
    updatedAt: number
  }
}

const customStore = createStore('boyan-offline-db', 'offline-books')

export async function saveOfflineBook(book: OfflineBook): Promise<void> {
  await set(book.id, book, customStore)
}

export async function getOfflineBook(id: string): Promise<OfflineBook | undefined> {
  return await get<OfflineBook>(id, customStore)
}

export async function removeOfflineBook(id: string): Promise<void> {
  await del(id, customStore)
}

export async function listOfflineBooks(): Promise<OfflineBook[]> {
  const all = await values<OfflineBook>(customStore)
  return all.sort((a, b) => b.savedAt - a.savedAt)
}

export async function updateOfflineProgress(
  id: string,
  progress: { position: string; percentage: number }
): Promise<void> {
  const book = await getOfflineBook(id)
  if (book) {
    book.progress = {
      ...progress,
      updatedAt: Date.now(),
    }
    await set(id, book, customStore)
  }
}
