import { type GrafanaTheme2 } from '@grafana/data';
import { getThemeById } from '@grafana/data/internal';
import { config, ThemeChangedEvent } from '@grafana/runtime';

import { appEvents } from '../app_events';
import { contextSrv } from '../services/context_srv';

import { PreferencesService } from './PreferencesService';

const HEX_COLOR_PATTERN = /^#([0-9A-Fa-f]{3}|[0-9A-Fa-f]{6}|[0-9A-Fa-f]{8})$/;

function getAccentFromUrl(): string | undefined {
  const accent = new URLSearchParams(window.location.search).get('accent');
  if (accent && HEX_COLOR_PATTERN.test(accent)) {
    return accent;
  }
  return undefined;
}

function applyAccentToTheme(theme: GrafanaTheme2): GrafanaTheme2 {
  const accent = getAccentFromUrl();
  if (!accent) {
    return theme;
  }

  theme.colors.primary.main = accent;
  theme.colors.accent.main = accent;
  theme.colors.text.link = accent;
  theme.colors.action.selectedBorder = accent;

  return theme;
}

export function initAccentFromUrl() {
  const accent = getAccentFromUrl();
  if (!accent) {
    return;
  }

  const theme = applyAccentToTheme(config.theme2);
  appEvents.publish(new ThemeChangedEvent(theme));
}

export async function changeTheme(themeId: string, runtimeOnly?: boolean) {
  const oldTheme = config.theme2;

  const newTheme = applyAccentToTheme(getThemeById(themeId));

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
