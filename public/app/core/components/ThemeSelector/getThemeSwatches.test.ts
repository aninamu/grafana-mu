import { createTheme } from '@grafana/data';

import { getThemeSwatches } from './getThemeSwatches';

describe('getThemeSwatches', () => {
  const theme = createTheme({ colors: { mode: 'dark' } });

  it('returns exactly the requested number of swatches', () => {
    expect(getThemeSwatches(theme, 3)).toHaveLength(3);
  });

  it('does not include undefined colors when count exceeds palette size', () => {
    const swatches = getThemeSwatches(theme, 10);
    expect(swatches.every((color) => color !== undefined)).toBe(true);
  });
});
