/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

declare module 'epubjs' {
  const ePub: any
  export default ePub
  export type Book = any
  export type Rendition = any
}
