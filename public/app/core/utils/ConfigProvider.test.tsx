jest.mock('../services/PreferencesService', () => ({
  PreferencesService: jest.fn().mockImplementation(() => ({
    patch: jest.fn(),
  })),
}));

import { render, screen, waitFor } from '@testing-library/react';

import { createTheme, ThemeContext } from '@grafana/data';
import { config } from '@grafana/runtime';

import { ThemeProvider } from './ConfigProvider';

describe('ThemeProvider', () => {
  const originalSearch = window.location.search;
  const originalTheme = config.theme2;

  afterEach(() => {
    Object.defineProperty(window, 'location', {
      value: { search: originalSearch },
      writable: true,
    });
    config.theme2 = originalTheme;
  });

  it('applies the URL accent on cold load without clearing it during mount sync', async () => {
    Object.defineProperty(window, 'location', {
      value: { search: '?accent=%23ff0000' },
      writable: true,
    });

    const baseTheme = createTheme({ colors: { mode: 'dark' } });
    config.theme2 = baseTheme;

    render(
      <ThemeProvider value={baseTheme}>
        <ThemeContext.Consumer>
          {(theme) => <span data-testid="primary-color">{theme.colors.primary.main}</span>}
        </ThemeContext.Consumer>
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('primary-color')).toHaveTextContent('#ff0000');
      expect(config.theme2.colors.primary.main).toBe('#ff0000');
    });
  });
});
