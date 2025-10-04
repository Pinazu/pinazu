// stores/theme.ts
import { writable, derived, readable } from 'svelte/store';

type Theme = 'light' | 'dark' | 'system';
type AppliedTheme = 'light' | 'dark';

// Check localStorage for a saved theme, or default to 'system'
const storedTheme = localStorage.getItem('theme') as Theme | null;
const currentTheme: Theme = storedTheme || 'system';

// Create a writable store with the current theme setting
export const themeSetting = writable<Theme>(currentTheme);

// Create a store for system theme preference
const systemTheme = readable<AppliedTheme>('light', (set) => {
    if (typeof window === 'undefined') return;
    
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const updateSystemTheme = () => set(mediaQuery.matches ? 'dark' : 'light');
    
    updateSystemTheme();
    mediaQuery.addEventListener('change', updateSystemTheme);
    
    return () => mediaQuery.removeEventListener('change', updateSystemTheme);
});

// Create a derived store for the actual applied theme
export const theme = derived([themeSetting, systemTheme], ([setting, system]) => {
    return setting === 'system' ? system : setting;
});

// Update the theme in localStorage whenever the setting changes
themeSetting.subscribe((value) => {
    localStorage.setItem('theme', value);
});

// Apply the actual theme whenever it changes
theme.subscribe((appliedTheme) => {
    if (typeof document !== 'undefined') {
        document.documentElement.classList.toggle('dark', appliedTheme === 'dark');
    }
});

// Apply initial theme
if (typeof window !== 'undefined') {
    const initialAppliedTheme = currentTheme === 'system' 
        ? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
        : currentTheme;
    document.documentElement.classList.toggle('dark', initialAppliedTheme === 'dark');
}