import { css } from '@emotion/css';

import { FeatureState, type GrafanaTheme2, type ThemeRegistryItem } from '@grafana/data';
import { t } from '@grafana/i18n';
import { FeatureBadge, RadioButtonDot, useStyles2 } from '@grafana/ui';

import { ThemePreview } from '../Theme/ThemePreview';

import { getThemeSwatches } from './getThemeSwatches';

interface ThemeCardProps {
  themeOption: ThemeRegistryItem;
  isExperimental?: boolean;
  isSelected?: boolean;
  onSelect: () => void;
}

export function ThemeCard({ themeOption, isExperimental, isSelected, onSelect }: ThemeCardProps) {
  const theme = themeOption.build();
  const label = getTranslatedThemeName(themeOption);
  const styles = useStyles2(getStyles);
  const swatches = getThemeSwatches(theme, 3);

  return (
    // this is a convenience for mouse users. keyboard/screen reader users will use the radio button
    // eslint-disable-next-line jsx-a11y/no-static-element-interactions,jsx-a11y/click-events-have-key-events
    <div className={styles.card} onClick={onSelect}>
      <div className={styles.header}>
        <RadioButtonDot
          id={`theme-${theme.name}`}
          name={'theme'}
          label={label}
          onClick={(event) => {
            // prevent propagation so that onSelect is only called once when clicking the radio button
            event.stopPropagation();
          }}
          onChange={onSelect}
          checked={isSelected}
        />
        {isExperimental && <FeatureBadge featureState={FeatureState.experimental} />}
      </div>
      <div className={styles.swatches}>
        {swatches.map((color, index) => (
          <span key={index} className={styles.swatch} style={{ backgroundColor: color }} />
        ))}
      </div>
      <ThemePreview theme={theme} />
    </div>
  );
}

const getStyles = (theme: GrafanaTheme2) => {
  return {
    card: css({
      border: `1px solid ${theme.colors.border.weak}`,
      borderRadius: theme.shape.radius.default,
      display: 'flex',
      flexDirection: 'column',
      overflow: 'hidden',
      cursor: 'pointer',
      '&:hover': {
        border: `1px solid ${theme.colors.border.medium}`,
      },
    }),
    header: css({
      alignItems: 'center',
      borderBottom: `1px solid ${theme.colors.border.weak}`,
      display: 'flex',
      justifyContent: 'space-between',
      padding: theme.spacing(1),
      // The RadioButtonDot is not correctly implemented at the moment, missing cursor (And click ability for the label and input)
      '> label': {
        cursor: 'pointer',
      },
    }),
    swatches: css({
      display: 'flex',
      gap: theme.spacing(0.5),
      padding: theme.spacing(0.5, 1),
    }),
    swatch: css({
      borderRadius: theme.shape.radius.circle,
      height: theme.spacing(1.5),
      width: theme.spacing(1.5),
    }),
  };
};

function getTranslatedThemeName(theme: ThemeRegistryItem) {
  switch (theme.id) {
    case 'dark':
      return t('shared.preferences.theme.dark-label', 'Dark');
    case 'light':
      return t('shared.preferences.theme.light-label', 'Light');
    case 'system':
      return t('shared.preferences.theme.system-label', 'System preference');
    default:
      return theme.name;
  }
}
