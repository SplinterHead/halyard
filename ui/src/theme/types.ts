/**
 * Halyard UI Theme System - Type Definitions
 *
 * Defines the hierarchical 5-surface elevation model and semantic color contract
 * for Halyard's default Dark (Deep Slate) and Light (Alabaster) modes.
 */

export type ThemeMode = 'dark' | 'light';
export type ColorSchemePreference = 'system' | 'dark' | 'light';

export interface ThemeColors {
  // Elevation Surfaces
  background: string;       // --surface-0: Page canvas & recessed log/code viewports
  navigation: string;       // --surface-nav: Left icon rail, top bar, mobile bottom bar
  content: string;          // --surface-1: Cards, table bodies, mobile card tops, modals
  contentFeature: string;   // --surface-2: Table headers/footers, card headers, nested boxes, mobile card bottoms
  surfaceControl: string;   // --surface-3: Interactive controls, telemetry tracks, secondary hover

  // Borders
  borderPrimary: string;    // --border-1: Primary structural border
  borderSecondary: string;  // --border-2: Subtle internal row divider

  // Typography
  textPrimary: string;      // --text-1: Primary text, headings, metric numbers
  textSecondary: string;    // --text-2: Secondary text, subtitles, table headers
  textMuted: string;        // --text-3: Muted labels, timestamps, empty states

  // Accent & Primary Button
  accent: string;           // --accent: Primary action button fill & active indicators
  accentFg: string;         // --accent-fg: Text color on --accent buttons
  accentTint: string;       // --accent-tint: Active navigation icon button background

  // Semantic Status
  statusOk: string;         // --status-ok
  statusWarn: string;       // --status-warn
  statusErr: string;        // --status-err
}

export interface ThemeDefinition {
  id: ThemeMode;
  name: string;
  description: string;
  mode: ThemeMode;
  colors: ThemeColors;
}

export interface ThemeInput {
  id: ThemeMode;
  name: string;
  description?: string;
  mode: ThemeMode;
  colors: Partial<ThemeColors> & { background: string };
}
