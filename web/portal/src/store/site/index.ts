import { createSlice, type PayloadAction } from '@reduxjs/toolkit'

export interface PortalConfig {
  projectName: string
  logoUrl?: string
  primaryColor?: string
  customAuthEnabled: boolean
  authUrl?: string
  customAuthButtonText?: string
  googleAuthEnabled: boolean
  googleClientId?: string
  entryTypeTemplates: Record<string, Record<string, string>>
  supportRequestEnabled: boolean
  supportRequestUrl?: string
  uploadsEnabled: boolean
}

interface SiteState {
  config: PortalConfig | null
}

const initialState: SiteState = { config: null }

const slice = createSlice({
  name: 'site',
  initialState,
  reducers: {
    setConfig(state, action: PayloadAction<PortalConfig>) {
      state.config = action.payload
    },
  },
})

export const { setConfig } = slice.actions
export default slice.reducer
