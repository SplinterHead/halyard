import { test, describe } from 'node:test';
import assert from 'node:assert/strict';

import {
  toCssVariables,
  toVuetifyTheme,
  getTheme,
  getAvailableThemes,
  darkTheme,
  lightTheme,
} from './index.ts';

describe('Halyard Hierarchical Surface Theme System', () => {
  test('built-in Deep Slate (dark) and Alabaster (light) themes match specification', () => {
    assert.equal(darkTheme.id, 'dark');
    assert.equal(darkTheme.name, 'Deep Slate');
    assert.equal(darkTheme.mode, 'dark');
    assert.equal(darkTheme.colors.background, '#0a0e17');
    assert.equal(darkTheme.colors.navigation, '#0e131f');
    assert.equal(darkTheme.colors.content, '#141c2b');
    assert.equal(darkTheme.colors.contentFeature, '#1c2638');
    assert.equal(darkTheme.colors.surfaceControl, '#263349');
    assert.equal(darkTheme.colors.borderPrimary, '#28354c');
    assert.equal(darkTheme.colors.borderSecondary, '#1e293b');
    assert.equal(darkTheme.colors.accent, '#3b82f6');

    assert.equal(lightTheme.id, 'light');
    assert.equal(lightTheme.name, 'Alabaster');
    assert.equal(lightTheme.mode, 'light');
    assert.equal(lightTheme.colors.background, '#eef2f6');
    assert.equal(lightTheme.colors.navigation, '#e5eaf1');
    assert.equal(lightTheme.colors.content, '#ffffff');
    assert.equal(lightTheme.colors.contentFeature, '#f5f7fa');
    assert.equal(lightTheme.colors.surfaceControl, '#e8ecf2');
    assert.equal(lightTheme.colors.borderPrimary, '#d0d7e2');
    assert.equal(lightTheme.colors.borderSecondary, '#e3e8ef');
    assert.equal(lightTheme.colors.accent, '#2563eb');
  });

  test('getAvailableThemes returns default dark and light themes', () => {
    const list = getAvailableThemes();
    assert.equal(list.length, 2);
    assert.deepEqual(
      list.map((t) => t.id),
      ['dark', 'light']
    );
  });

  test('getTheme resolves dark/light and defaults to dark', () => {
    assert.equal(getTheme('dark').id, 'dark');
    assert.equal(getTheme('light').id, 'light');
    assert.equal(getTheme('unknown').id, 'dark');
  });

  test('every registered theme generates distinct hierarchical surface & table tokens', () => {
    const requiredKeys = [
      '--surface-0',
      '--surface-nav',
      '--surface-1',
      '--surface-2',
      '--surface-3',
      '--border-1',
      '--border-2',
      '--text-1',
      '--text-2',
      '--text-3',
      '--accent',
      '--table-th-bg',
      '--table-footer-bg',
      '--btn-primary-bg',
    ];

    for (const theme of getAvailableThemes()) {
      const vars = toCssVariables(theme);
      for (const key of requiredKeys) {
        assert.ok(vars[key], `${theme.id} must define ${key}`);
      }

      // Surface hierarchy must be distinct so layers never bleed into each other
      assert.notEqual(vars['--surface-nav'], vars['--surface-0'], `${theme.id} nav must differ from canvas`);
      assert.notEqual(vars['--surface-1'], vars['--surface-0'], `${theme.id} surface-1 must differ from canvas`);
      assert.notEqual(vars['--surface-2'], vars['--surface-1'], `${theme.id} surface-2 must differ from surface-1`);
      assert.notEqual(vars['--surface-3'], vars['--surface-2'], `${theme.id} surface-3 must differ from surface-2`);
      assert.equal(vars['--table-th-bg'], vars['--surface-2'], `${theme.id} table header must use surface-2`);
      assert.equal(vars['--table-footer-bg'], vars['--surface-2'], `${theme.id} table footer must use surface-2`);
      assert.equal(vars['--btn-primary-bg'], vars['--accent'], `${theme.id} primary button must use accent`);
    }
  });

  test('toVuetifyTheme maps dark and light themes accurately', () => {
    const vDark = toVuetifyTheme(darkTheme);
    assert.equal(vDark.dark, true);
    assert.equal(vDark.colors.background, '#0a0e17');
    assert.equal(vDark.colors.surface, '#141c2b');
    assert.equal(vDark.colors.primary, '#3b82f6');

    const vLight = toVuetifyTheme(lightTheme);
    assert.equal(vLight.dark, false);
    assert.equal(vLight.colors.background, '#eef2f6');
    assert.equal(vLight.colors.surface, '#ffffff');
    assert.equal(vLight.colors.primary, '#2563eb');
  });
});
