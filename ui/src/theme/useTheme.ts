/**
 * Halyard UI Theme System - Vue 3 Composable
 *
 * Syncs <meta name="color-scheme">, localStorage ('halyard-color-scheme'),
 * system preference media query changes, cross-tab storage events, and Vuetify theme.
 */

import { ref, computed } from 'vue';
import { getTheme, getAvailableThemes } from './themes.ts';
import type { ColorSchemePreference, ThemeDefinition, ThemeMode } from './types.ts';

const STORAGE_KEY = 'halyard-color-scheme';

function readStoredPreference(): ColorSchemePreference {
  if (typeof window === 'undefined') return 'system';
  try {
    const val = localStorage.getItem(STORAGE_KEY);
    if (val === 'dark' || val === 'light') return val;
  } catch {
    // Ignore storage access errors
  }
  return 'system';
}

function getSystemMode(): ThemeMode {
  if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  return 'dark';
}

const schemePreference = ref<ColorSchemePreference>(readStoredPreference());
const systemMode = ref<ThemeMode>(getSystemMode());
let vuetifyThemeRef: any = null;
let listenersBound = false;

const resolvedMode = computed<ThemeMode>(() => {
  if (schemePreference.value === 'dark' || schemePreference.value === 'light') {
    return schemePreference.value;
  }
  return systemMode.value;
});

function syncDomAndVuetify(): void {
  const metaContent = schemePreference.value === 'system' ? 'light dark' : schemePreference.value;

  if (typeof document !== 'undefined') {
    const meta = document.querySelector('meta[name="color-scheme"]');
    if (meta) {
      meta.setAttribute('content', metaContent);
    }
    document.documentElement.setAttribute('data-theme', resolvedMode.value);
  }

  if (vuetifyThemeRef && vuetifyThemeRef.global) {
    vuetifyThemeRef.global.name.value = resolvedMode.value;
  }
}

function bindGlobalListeners(): void {
  if (listenersBound || typeof window === 'undefined') return;
  listenersBound = true;

  if (typeof window.matchMedia === 'function') {
    const mql = window.matchMedia('(prefers-color-scheme: dark)');
    const onSystemChange = (e: MediaQueryListEvent) => {
      systemMode.value = e.matches ? 'dark' : 'light';
      syncDomAndVuetify();
    };
    if (typeof mql.addEventListener === 'function') {
      mql.addEventListener('change', onSystemChange);
    }
  }

  window.addEventListener('storage', (e: StorageEvent) => {
    if (e.key === STORAGE_KEY) {
      schemePreference.value = readStoredPreference();
      syncDomAndVuetify();
    }
  });
}

export function initTheme(vuetifyTheme?: any): void {
  if (vuetifyTheme) {
    vuetifyThemeRef = vuetifyTheme;
  }
  schemePreference.value = readStoredPreference();
  systemMode.value = getSystemMode();
  bindGlobalListeners();
  syncDomAndVuetify();
}

export function useTheme(vuetifyTheme?: any) {
  if (vuetifyTheme && !vuetifyThemeRef) {
    vuetifyThemeRef = vuetifyTheme;
  }

  const currentTheme = computed<ThemeDefinition>(() => getTheme(resolvedMode.value));
  const isDark = computed<boolean>(() => resolvedMode.value === 'dark');
  const availableThemes = computed<ThemeDefinition[]>(() => getAvailableThemes());

  function setScheme(pref: ColorSchemePreference): void {
    schemePreference.value = pref;
    try {
      if (pref === 'system') {
        localStorage.removeItem(STORAGE_KEY);
      } else {
        localStorage.setItem(STORAGE_KEY, pref);
      }
    } catch {
      // Ignore storage access errors
    }
    syncDomAndVuetify();
  }

  function setTheme(id: string): void {
    if (id === 'system') {
      setScheme('system');
    } else if (id === 'light' || id === 'dark') {
      setScheme(id);
    }
  }

  /**
   * Two-state toggle: switches to the opposite of the currently rendered scheme.
   * If the target matches the system preference, clears the stored override.
   */
  function toggleTheme(): void {
    const target: ThemeMode = resolvedMode.value === 'dark' ? 'light' : 'dark';
    if (target === systemMode.value) {
      setScheme('system');
    } else {
      setScheme(target);
    }
  }

  return {
    schemePreference,
    resolvedMode,
    systemMode,
    currentTheme,
    isDark,
    availableThemes,
    setScheme,
    setTheme,
    toggleTheme,
  };
}
