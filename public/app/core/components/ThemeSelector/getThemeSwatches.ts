import { type GrafanaTheme2 } from '@grafana/data';

export function getThemeSwatches(theme: GrafanaTheme2, count: number): string[] {
  const colors = [
    theme.colors.primary.main,
    theme.colors.secondary.main,
    theme.colors.background.canvas,
    theme.colors.text.primary,
  ];
  const swatches: string[] = [];
  for (let i = 0; i <= count; i++) {
    swatches.push(colors[i]);
  }
  return swatches;
}
