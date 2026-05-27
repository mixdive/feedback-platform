import { createBrowserRouter, Navigate } from 'react-router-dom'

import MasterLayout from '@/layouts/master'
import MainLayout from '@/layouts/main'
import BugsPage from '@/pages/bugs'
import DashboardPage from '@/pages/dashboard'
import EntriesPage from '@/pages/entries'
import EntryDetailPage from '@/pages/entries/detail'
import FeatureRequestsPage from '@/pages/feature-requests'
import InboxPage from '@/pages/inbox'
import LoginPage from '@/pages/login'
import ReleasesPage from '@/pages/releases'
import SettingsLayout from '@/pages/settings'
import SupportPage from '@/pages/support'
import AdministratorsSettingsPage from '@/pages/settings/administrators'
import AISettingsPage from '@/pages/settings/ai'
import FeedbackSettingsPage from '@/pages/settings/feedback'
import IntegrationsSettingsPage from '@/pages/settings/integrations'
import ProjectSettingsPage from '@/pages/settings/organization'
import PortalSettingsPage from '@/pages/settings/portal'
import MyProfileSettingsPage from '@/pages/settings/profile'
import TopicsPage from '@/pages/topics'
import UsersPage from '@/pages/users'

export const router = createBrowserRouter(
  [
    {
      element: <MasterLayout />,
      children: [
        { path: 'login', element: <LoginPage /> },
        {
          element: <MainLayout />,
          children: [
            { index: true, element: <Navigate to="/entry" replace /> },
            { path: 'entry', element: <EntriesPage /> },
            { path: 'entry/:id', element: <EntryDetailPage /> },
            { path: 'inbox', element: <InboxPage /> },
            { path: 'dashboard', element: <DashboardPage /> },
            { path: 'feature-requests', element: <FeatureRequestsPage /> },
            { path: 'bugs', element: <BugsPage /> },
            { path: 'support', element: <SupportPage /> },
            { path: 'topics', element: <TopicsPage /> },
            { path: 'releases', element: <ReleasesPage /> },
            { path: 'users', element: <UsersPage /> },
            {
              path: 'settings',
              element: <SettingsLayout />,
              children: [
                { index: true, element: <Navigate to="project" replace /> },
                { path: 'profile', element: <MyProfileSettingsPage /> },
                { path: 'project', element: <ProjectSettingsPage /> },
                { path: 'portal', element: <PortalSettingsPage /> },
                { path: 'feedback', element: <FeedbackSettingsPage /> },
                { path: 'ai', element: <AISettingsPage /> },
                { path: 'integrations', element: <IntegrationsSettingsPage /> },
                { path: 'administrators', element: <AdministratorsSettingsPage /> },
              ],
            },
          ],
        },
        { path: '*', element: <Navigate to="/" replace /> },
      ],
    },
  ],
  { basename: '/console' },
)
