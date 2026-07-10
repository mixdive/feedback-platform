import { useEffect } from 'react'

// useDocumentTitle sets the browser tab title while the calling component
// is mounted and restores whatever it was on unmount. Pass an empty /
// undefined value (e.g. while data is still loading) to leave the title
// untouched until the real value is available.
export function useDocumentTitle(title: string | undefined) {
  useEffect(() => {
    if (!title) return
    const previous = document.title
    document.title = title
    return () => {
      document.title = previous
    }
  }, [title])
}
