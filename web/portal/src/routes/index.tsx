import { createBrowserRouter } from 'react-router-dom'

import MasterLayout from '@/layouts/master'
import MainLayout from '@/layouts/main'
import ChangelogPage from '@/pages/changelog'
import ChangelogDetailPage from '@/pages/changelog/detail'
import EntriesPage from '@/pages/entries'
import EntryDetailPage from '@/pages/entries/detail'
import NewEntryPage from '@/pages/new'

export const router = createBrowserRouter([
  {
    element: <MasterLayout />,
    children: [
      {
        element: <MainLayout />,
        children: [
          { index: true, element: <EntriesPage /> },
          { path: 'new', element: <NewEntryPage /> },
          { path: 'entry/:id', element: <EntryDetailPage /> },
          { path: 'changelog', element: <ChangelogPage /> },
          { path: 'changelog/:slug', element: <ChangelogDetailPage /> },
        ],
      },
    ],
  },
])
