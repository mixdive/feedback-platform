import { configureStore, createSlice, type PayloadAction } from '@reduxjs/toolkit'

import type { ApiUser } from '@/services/api'

// Auth state for the Console. The user lives here so MainLayout can render
// the email + roles in the sidebar without a separate query. The cookie
// itself is the source of truth — this slice is just a cache, refreshed
// from /api/me on app load.
type AuthStatus = 'idle' | 'loading' | 'authenticated' | 'unauthenticated'

type AuthState = {
  status: AuthStatus
  user: ApiUser | null
}

const initialAuth: AuthState = { status: 'idle', user: null }

const authSlice = createSlice({
  name: 'auth',
  initialState: initialAuth,
  reducers: {
    authLoading(state) {
      state.status = 'loading'
    },
    authResolved(state, action: PayloadAction<ApiUser>) {
      state.status = 'authenticated'
      state.user = action.payload
    },
    authCleared(state) {
      state.status = 'unauthenticated'
      state.user = null
    },
  },
})

export const { authLoading, authResolved, authCleared } = authSlice.actions

export const store = configureStore({
  reducer: { auth: authSlice.reducer },
})

export type RootState = ReturnType<typeof store.getState>
export type AppDispatch = typeof store.dispatch
