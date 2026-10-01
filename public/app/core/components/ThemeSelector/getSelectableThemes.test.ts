import { getFeatureFlagClient } from '@grafana/runtime/internal';

import { getSelectableThemes } from './getSelectableThemes';

jest.mock('@grafana/runtime/internal', () => ({
  ...jest.requireActual('@grafana/runtime/internal'),
  getFeatureFlagClient: jest.fn(),
}));

describe('getSelectableThemes', () => {
  beforeEach(() => {
    (getFeatureFlagClient as jest.Mock).mockReturnValue({ getBooleanValue: () => false });
  });

  it('includes the pink theme', () => {
    const pink = getSelectableThemes().find((theme) => theme.id === 'pink');

    expect(pink?.name).toBe('Pink');
    expect(pink?.isExtra).toBe(true);
    expect(pink?.build().isLight).toBe(true);
    expect(pink?.build().colors.primary.main).toBe('#E91E63');
    expect(pink?.build().colors.background.canvas).toBe('#FFF0F6');
  });
});
