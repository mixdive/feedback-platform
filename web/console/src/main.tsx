import { createRoot } from 'react-dom/client'
import { Provider } from 'react-redux'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from 'react-router-dom'

import { store } from '@/store'
import { router } from '@/routes'
import '@/utils/i18n'
import '@/assets/css/style.css'

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: 0 } },
})

createRoot(document.getElementById('root')!).render(
  <QueryClientProvider client={queryClient}>
    <Provider store={store}>
      <RouterProvider router={router} />
    </Provider>
  </QueryClientProvider>,
)
