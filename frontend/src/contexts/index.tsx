import React from 'react';
import { AuthProvider } from './AuthContext';
import { UserPreferencesProvider } from './UserPreferencesContext';
import { CommandPaletteProvider } from './CommandPaletteContext';
import { AppStateProvider } from './AppStateContext';
import { NotificationProvider } from '@/components/notifications/NotificationProvider';
import { BottomNavProvider } from './BottomNavContext';

interface AppProvidersProps {
  children: React.ReactNode;
}

/**
 * Combined provider that wraps the entire app with all necessary contexts
 * Order is important: Auth -> UserPreferences -> CommandPalette -> AppState -> Notifications
 */
export const AppProviders = ({ children }: AppProvidersProps) => {
  return (
    <AuthProvider>
      <UserPreferencesProvider>
        <CommandPaletteProvider>
          <AppStateProvider>
            <NotificationProvider>
              {children}
            </NotificationProvider>
          </AppStateProvider>
        </CommandPaletteProvider>
      </UserPreferencesProvider>
    </AuthProvider>
  );
};

// Re-export all context hooks for convenience
export { useAuth } from './AuthContext';
export { useUserPreferences } from './UserPreferencesContext';
export { useCommandPalette } from './CommandPaletteContext';
export { useAppState } from './AppStateContext';

export { useBottomNav } from './BottomNavContext';
export type { UserAction, UserPreferences, ActionFrequency } from '@/lib/storage/types';
