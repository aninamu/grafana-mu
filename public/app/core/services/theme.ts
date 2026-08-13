import tinycolor from 'tinycolor2';

import { createTheme, type GrafanaTheme2 } from '@grafana/data';
import { getThemeById } from '@grafana/data/internal';
import { config, ThemeChangedEvent } from '@grafana/runtime';

import { appEvents } from '../app_events';
import { contextSrv } from '../services/context_srv';

import { PreferencesService } from './PreferencesService';

function getValidAccentFromUrl(): string | null {
  const accent = new URLSearchParams(window.location.search).get('accent');
  if (!accent) {
    return null;
  }

  const color = tinycolor(accent);
  return color.isValid() ? color.toString() : null;
}

function applyAccentToTheme(theme: GrafanaTheme2, accent: string): GrafanaTheme2 {
  const color = tinycolor(accent);
  const main = color.toHexString();
  const lightAccent = color.lighten(20).toHexString();
  const { palette, hues } = theme.visualization;

  return createTheme({
    name: theme.name,
    colors: {
      mode: theme.colors.mode,
      primary: { main },
      accent: { main },
      text: {
        primary: theme.colors.text.primary,
        secondary: theme.colors.text.secondary,
        disabled: theme.colors.text.disabled,
        link: main,
        maxContrast: theme.colors.text.maxContrast,
      },
      background: theme.colors.background,
      border: theme.colors.border,
      tertiary: theme.colors.tertiary,
      info: theme.colors.info,
      error: theme.colors.error,
      success: theme.colors.success,
      warning: theme.colors.warning,
      secondary: theme.colors.secondary,
      action: {
        ...theme.colors.action,
        selectedBorder: main,
        hover: color.setAlpha(0.16).toRgbString(),
        selected: color.setAlpha(0.12).toRgbString(),
        focus: color.setAlpha(0.16).toRgbString(),
        disabledBackground: color.setAlpha(0.08).toRgbString(),
      },
      gradients: {
        brandHorizontal: `linear-gradient(270deg, ${main} 0%, ${lightAccent} 100%)`,
        brandVertical: `linear-gradient(0.01deg, ${main} 0.01%, ${lightAccent} 99.99%)`,
      },
      contrastThreshold: theme.colors.contrastThreshold,
      hoverFactor: theme.colors.hoverFactor,
      tonalOffset: theme.colors.tonalOffset,
      scrollbar: theme.colors.scrollbar,
    },
    visualization: { palette, hues },
  });
}

export function applyAccentFromUrl(theme: GrafanaTheme2 = config.theme2): GrafanaTheme2 {
  const accent = getValidAccentFromUrl();
  if (!accent) {
    return theme;
  }

  return applyAccentToTheme(theme, accent);
}

export function initAccentFromUrl() {
  const accent = getValidAccentFromUrl();
  if (!accent) {
    return;
  }

  appEvents.publish(new ThemeChangedEvent(applyAccentFromUrl()));
}

export async function changeTheme(themeId: string, runtimeOnly?: boolean) {
  const oldTheme = config.theme2;

  const newTheme = applyAccentFromUrl(getThemeById(themeId));

  appEvents.publish(new ThemeChangedEvent(newTheme));

  // Add css file for new theme
  if (oldTheme.colors.mode !== newTheme.colors.mode) {
    const newCssLink = document.createElement('link');
    newCssLink.rel = 'stylesheet';
    newCssLink.href = config.bootData.assets[newTheme.colors.mode];
    newCssLink.onload = () => {
      // Remove old css file
      const bodyLinks = document.getElementsByTagName('link');
      for (let i = 0; i < bodyLinks.length; i++) {
        const link = bodyLinks[i];

        if (link.href && link.href.includes(`build/grafana.${oldTheme.colors.mode}`)) {
          // Remove existing link once the new css has loaded to avoid flickering
          // If we add new css at the same time we remove current one the page will be rendered without css
          // As the new css file is loading
          link.remove();
        }
      }
    };
    document.head.insertBefore(newCssLink, document.head.firstChild);
  }

  if (runtimeOnly) {
    return;
  }

  if (!contextSrv.isSignedIn) {
    return;
  }

  // Persist new theme
  const service = new PreferencesService('user');
  await service.patch({
    theme: themeId,
  });
}

export async function toggleTheme(runtimeOnly: boolean) {
  const currentTheme = config.theme2;
  changeTheme(currentTheme.isDark ? 'light' : 'dark', runtimeOnly);
}
