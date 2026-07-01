jest.mock('./PreferencesService', () => ({
  PreferencesService: jest.fn().mockImplementation(() => ({
    patch: jest.fn(),
  })),
}));

jest.mock('../services/context_srv', () => ({
  contextSrv: { isSignedIn: false },
}));

jest.mock('../app_events', () => ({
  appEvents: {
    publish: jest.fn(),
    subscribe: jest.fn(() => ({ unsubscribe: jest.fn() })),
  },
}));

import { createTheme } from '@grafana/data';

import { applyAccentFromUrl } from './theme';

describe('applyAccentFromUrl', () => {
  const originalSearch = window.location.search;

  afterEach(() => {
    Object.defineProperty(window, 'location', {
      value: { search: originalSearch },
      writable: true,
    });
  });

  it('applies a valid accent color to the theme object', () => {
    Object.defineProperty(window, 'location', {
      value: { search: '?accent=%23ff0000' },
      writable: true,
    });

    const baseTheme = createTheme({ colors: { mode: 'dark' } });
    const themed = applyAccentFromUrl(baseTheme);

    expect(themed.colors.primary.main).toBe('#ff0000');
    expect(themed.colors.accent.main).toBe('#ff0000');
  });

  it('ignores invalid accent values instead of injecting raw CSS', () => {
    Object.defineProperty(window, 'location', {
      value: { search: '?accent=not-a-color' },
      writable: true,
    });

    const baseTheme = createTheme({ colors: { mode: 'dark' } });
    const themed = applyAccentFromUrl(baseTheme);

    expect(themed).toBe(baseTheme);
  });

  it('accepts named CSS colors via tinycolor validation', () => {
    Object.defineProperty(window, 'location', {
      value: { search: '?accent=orange' },
      writable: true,
    });

    const themed = applyAccentFromUrl(createTheme({ colors: { mode: 'dark' } }));

    expect(themed.colors.primary.main.toLowerCase()).toBe('#ffa500');
  });

  it('preserves semantic colors and visualization palettes when applying an accent', () => {
    Object.defineProperty(window, 'location', {
      value: { search: '?accent=%23ff0000' },
      writable: true,
    });

    const baseTheme = createTheme({
      colors: {
        mode: 'dark',
        success: { main: '#2EC4B6' },
        error: { main: '#E8467C', text: '#F57098' },
        warning: { main: '#FADE2A' },
      },
      visualization: {
        hues: [
          {
            name: 'green',
            shades: [{ color: '#2EC4B6', name: 'green', primary: true }],
          },
        ],
        palette: ['green', 'red'],
      },
    });

    const themed = applyAccentFromUrl(baseTheme);

    expect(themed.colors.primary.main).toBe('#ff0000');
    expect(themed.colors.success.main).toBe('#2EC4B6');
    expect(themed.colors.error.main).toBe('#E8467C');
    expect(themed.colors.error.text).toBe('#F57098');
    expect(themed.colors.warning.main).toBe('#FADE2A');
    expect(themed.visualization.getColorByName('green')).toBe('#2EC4B6');
    expect(themed.visualization.palette).toEqual(['green', 'red']);
  });
});
