/**
 * Halyard UI Theme System - Generator & Compiler
 */

import type { ThemeColors, ThemeDefinition, ThemeInput, ThemeMode } from './types.ts';

export function defineTheme(input: ThemeInput): ThemeDefinition {
  const mode: ThemeMode = input.mode;
  const c = input.colors;
  const isDark = mode === 'dark';

  const fullColors: ThemeColors = {
    background: c.background,
    navigation: c.navigation || (isDark ? '#0e131f' : '#e5eaf1'),
    content: c.content || (isDark ? '#141c2b' : '#ffffff'),
    contentFeature: c.contentFeature || (isDark ? '#1c2638' : '#f5f7fa'),
    surfaceControl: c.surfaceControl || (isDark ? '#263349' : '#e8ecf2'),
    borderPrimary: c.borderPrimary || (isDark ? '#28354c' : '#d0d7e2'),
    borderSecondary: c.borderSecondary || (isDark ? '#1e293b' : '#e3e8ef'),
    textPrimary: c.textPrimary || (isDark ? '#f1f5f9' : '#0f172a'),
    textSecondary: c.textSecondary || (isDark ? '#94a3b8' : '#475569'),
    textMuted: c.textMuted || '#64748b',
    accent: c.accent || (isDark ? '#3b82f6' : '#2563eb'),
    accentFg: c.accentFg || '#ffffff',
    accentTint: c.accentTint || (isDark ? 'rgba(59, 130, 246, 0.15)' : 'rgba(37, 99, 235, 0.10)'),
    statusOk: c.statusOk || (isDark ? '#10b981' : '#059669'),
    statusWarn: c.statusWarn || (isDark ? '#f59e0b' : '#d97706'),
    statusErr: c.statusErr || (isDark ? '#ef4444' : '#dc2626'),
  };

  return {
    id: input.id,
    name: input.name,
    description: input.description || `${input.name} theme`,
    mode,
    colors: fullColors,
  };
}

/**
 * Maps a ThemeDefinition to the core hierarchical CSS variables and table/button aliases.
 */
export function toCssVariables(theme: ThemeDefinition): Record<string, string> {
  const c = theme.colors;

  return {
    // Core Hierarchical Surface System
    '--surface-0': c.background,
    '--surface-nav': c.navigation,
    '--surface-1': c.content,
    '--surface-2': c.contentFeature,
    '--surface-3': c.surfaceControl,

    // Borders
    '--border-1': c.borderPrimary,
    '--border-2': c.borderSecondary,

    // Typography
    '--text-1': c.textPrimary,
    '--text-2': c.textSecondary,
    '--text-3': c.textMuted,

    // Accent & Primary Actions
    '--accent': c.accent,
    '--accent-fg': c.accentFg,
    '--accent-tint': c.accentTint,

    // Semantic Status
    '--status-ok': c.statusOk,
    '--status-warn': c.statusWarn,
    '--status-err': c.statusErr,

    // Table & Button Mappings (fixes invisible headers/footers and washed-out primary buttons)
    '--table-bg': c.content,
    '--table-th-bg': c.contentFeature,
    '--table-footer-bg': c.contentFeature,
    '--table-row-hover': c.contentFeature,
    '--btn-primary-bg': c.accent,
    '--btn-primary-text': c.accentFg,
  };
}

/**
 * Converts a ThemeDefinition into Vuetify's theme color configuration.
 */
export function toVuetifyTheme(theme: ThemeDefinition) {
  const c = theme.colors;
  return {
    dark: theme.mode === 'dark',
    colors: {
      primary: c.accent,
      secondary: c.textSecondary,
      accent: c.accent,
      background: c.background,
      surface: c.content,
      'surface-variant': c.contentFeature,
      'on-surface': c.textPrimary,
      'on-background': c.textPrimary,
      'on-primary': c.accentFg,
      error: c.statusErr,
      info: c.accent,
      success: c.statusOk,
      warning: c.statusWarn,
    },
  };
}
