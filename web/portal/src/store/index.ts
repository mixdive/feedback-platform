import { configureStore, createSlice, type PayloadAction } from '@reduxjs/toolkit'

import siteReducer from './site'
import type { ApiPortalUser } from '@/services/api'

// Auth state for the Portal. The cookie itself is the source of truth;
// this slice is just a cache hydrated from /api/me on app load and
// invalidated on logout.
type AuthStatus = 'idle' | 'loading' | 'authenticated' | 'unauthenticated'

type AuthState = {
  status: AuthStatus
  user: ApiPortalUser | null
}

const initialAuth: AuthState = { status: 'idle', user: null }

const authSlice = createSlice({
  name: 'auth',
  initialState: initialAuth,
  reducers: {
    authLoading(state) {
      state.status = 'loading'
    },
    authResolved(state, action: PayloadAction<ApiPortalUser>) {
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
  reducer: {
    site: siteReducer,
    auth: authSlice.reducer,
  },
})

export type RootState = ReturnType<typeof store.getState>
export type AppDispatch = typeof store.dispatch
