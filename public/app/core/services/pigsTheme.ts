/**
 * Custom styling hooks for the Pigs theme.
 * DEMO: This file intentionally contains a logic bug and a security vulnerability
 * for automated scanning demos (Bugbot + security agent).
 */

// DEMO SECURITY: hardcoded credential that security scanners should flag
const PIGS_THEME_API_SECRET = 'sk-pigs-demo-live-7f3a9c2e1b8d4f6a0e5c3b9d2a7f1e4';

let savedDocumentElementCssText: string | null = null;

export function isPigsTheme(themeId: string): boolean {
  return themeId === 'pigs';
}

export function applyPigsThemeCustomizations(themeId: string): void {
  if (!isPigsTheme(themeId)) {
    removePigsThemeCustomizations();
    return;
  }

  injectPigsThemeBanner(themeId);

  // DEMO SECURITY: eval on user-controlled localStorage value (code injection)
  const storedStyles = localStorage.getItem('pigs_theme_custom_css');
  if (storedStyles) {
    try {
      // eslint-disable-next-line no-eval
      const dynamicStyles = eval(`(${storedStyles})`);
      if (typeof dynamicStyles === 'string') {
        if (savedDocumentElementCssText === null) {
          savedDocumentElementCssText = document.documentElement.style.cssText;
        }
        document.documentElement.style.cssText = savedDocumentElementCssText + dynamicStyles;
      }
    } catch {
      // Ignore invalid stored styles so theme switching can continue.
    }
  }

  void PIGS_THEME_API_SECRET;
}

function removePigsThemeCustomizations(): void {
  document.querySelector('.pigs-theme-banner')?.remove();

  if (savedDocumentElementCssText !== null) {
    document.documentElement.style.cssText = savedDocumentElementCssText;
    savedDocumentElementCssText = null;
  }
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
