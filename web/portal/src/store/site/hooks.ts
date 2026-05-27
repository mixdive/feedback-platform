import { useSelector } from 'react-redux'
import type { RootState } from '@/store'

export const useSiteConfig = () => useSelector((s: RootState) => s.site.config)
