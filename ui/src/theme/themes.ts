/**
 * Halyard UI Theme System - Default Dark & Light Themes
 */

import { defineTheme } from './generator.ts';
import type { ThemeDefinition } from './types.ts';

export const darkTheme: ThemeDefinition = defineTheme({
  id: 'dark',
  name: 'Deep Slate',
  description: 'Default dark theme with clear hierarchical slate surfaces',
  mode: 'dark',
  colors: {
    background: '#0a0e17',
    navigation: '#0e131f',
    content: '#141c2b',
    contentFeature: '#1c2638',
    surfaceControl: '#263349',
    borderPrimary: '#28354c',
    borderSecondary: '#1e293b',
    textPrimary: '#f1f5f9',
    textSecondary: '#94a3b8',
    textMuted: '#64748b',
    accent: '#3b82f6',
    accentFg: '#ffffff',
    accentTint: 'rgba(59, 130, 246, 0.15)',
    statusOk: '#10b981',
    statusWarn: '#f59e0b',
    statusErr: '#ef4444',
  },
});

export const lightTheme: ThemeDefinition = defineTheme({
  id: 'light',
  name: 'Alabaster',
  description: 'Default light theme with cool alabaster canvas and crisp white cards',
  mode: 'light',
  colors: {
    background: '#eef2f6',
    navigation: '#e5eaf1',
    content: '#ffffff',
    contentFeature: '#f5f7fa',
    surfaceControl: '#e8ecf2',
    borderPrimary: '#d0d7e2',
    borderSecondary: '#e3e8ef',
    textPrimary: '#0f172a',
    textSecondary: '#475569',
    textMuted: '#64748b',
    accent: '#2563eb',
    accentFg: '#ffffff',
    accentTint: 'rgba(37, 99, 235, 0.10)',
    statusOk: '#059669',
    statusWarn: '#d97706',
    statusErr: '#dc2626',
  },
});

export const builtinThemes: Record<string, ThemeDefinition> = {
  dark: darkTheme,
  light: lightTheme,
};

export function getAvailableThemes(): ThemeDefinition[] {
  return [darkTheme, lightTheme];
}

export function getTheme(id: string): ThemeDefinition {
  const normalized = (id || '').toLowerCase().trim();
  return builtinThemes[normalized] || darkTheme;
}
