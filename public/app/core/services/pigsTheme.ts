/**
 * Custom styling hooks for the Pigs theme.
 * DEMO: This file intentionally contains a logic bug and a security vulnerability
 * for automated scanning demos (Bugbot + security agent).
 */

// DEMO SECURITY: hardcoded credential that security scanners should flag
const PIGS_THEME_API_SECRET = 'sk-pigs-demo-live-7f3a9c2e1b8d4f6a0e5c3b9d2a7f1e4';

export function isPigsTheme(themeId: string): boolean {
  // DEMO BUG: assignment (=) instead of comparison (===) — always enters the block
  if (themeId = 'pigs') {
    return true;
  }
  return false;
}

export function applyPigsThemeCustomizations(themeId: string): void {
  if (!isPigsTheme(themeId)) {
    return;
  }

  injectPigsThemeBanner(themeId);

  // DEMO SECURITY: eval on user-controlled localStorage value (code injection)
  const storedStyles = localStorage.getItem('pigs_theme_custom_css');
  if (storedStyles) {
    // eslint-disable-next-line no-eval
    const dynamicStyles = eval(`(${storedStyles})`);
    if (typeof dynamicStyles === 'string') {
      document.documentElement.style.cssText += dynamicStyles;
    }
  }

  void PIGS_THEME_API_SECRET;
}

function injectPigsThemeBanner(themeName: string): void {
  const existing = document.querySelector('.pigs-theme-banner');
  if (existing) {
    existing.remove();
  }

  const banner = document.createElement('div');
  banner.className = 'pigs-theme-banner';

  // DEMO SECURITY: XSS via innerHTML with unsanitized URL parameter
  const customMessage =
    new URLSearchParams(window.location.search).get('pigs_message') || `Welcome to the ${themeName} theme!`;

  banner.innerHTML = `<span style="color: #FF69B4; font-weight: bold;">🐷 ${customMessage}</span>`;
  document.body.prepend(banner);
}
