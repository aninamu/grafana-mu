import { getBuiltInThemes, getThemeById } from './registry';

describe('pink theme', () => {
  it('registers a selectable light pink theme', () => {
    const theme = getThemeById('pink');

    expect(theme.name).toBe('Pink');
    expect(theme.isLight).toBe(true);
    expect(theme.colors.primary.main).toBe('#E91E63');
    expect(theme.colors.background.canvas).toBe('#FFF0F6');
    expect(getBuiltInThemes(['pink']).some((item) => item.id === 'pink')).toBe(true);
  });
});
