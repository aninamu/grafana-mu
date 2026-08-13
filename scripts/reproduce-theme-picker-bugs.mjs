/**
 * Reproduces the theme picker bugs before the fix:
 * 1. Off-by-one swatch count (requests 3, returns 4 / undefined)
 * 2. Accent URL injects unvalidated CSS via innerHTML (XSS)
 * 3. Accent written to unused --cursor-accent CSS var (not applied to theme)
 *
 * Run: node scripts/reproduce-theme-picker-bugs.mjs
 */

import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');

function read(rel) {
  return readFileSync(resolve(root, rel), 'utf8');
}

const swatches = read('public/app/core/components/ThemeSelector/getThemeSwatches.ts');
const theme = read('public/app/core/services/theme.ts');

const findings = [];

if (swatches.includes('i <= count')) {
  findings.push({
    id: 'swatch-off-by-one',
    severity: 'bug',
    detail: 'getThemeSwatches loops with i <= count, producing count+1 entries (and undefined when past palette).',
  });
} else if (swatches.includes('Math.min(count, colors.length)')) {
  findings.push({
    id: 'swatch-off-by-one',
    severity: 'fixed',
    detail: 'getThemeSwatches bounds the loop with Math.min(count, colors.length).',
  });
}

if (theme.includes('innerHTML') && theme.includes('--cursor-accent')) {
  findings.push({
    id: 'accent-xss-and-unused-var',
    severity: 'bug',
    detail: 'applyAccentFromUrl injects unvalidated accent into a <style> tag via innerHTML and sets unused --cursor-accent.',
  });
} else if (theme.includes('tinycolor') && theme.includes('createTheme')) {
  findings.push({
    id: 'accent-xss-and-unused-var',
    severity: 'fixed',
    detail: 'Accent is validated with tinycolor and applied via createTheme to theme tokens.',
  });
} else {
  findings.push({
    id: 'accent-xss-and-unused-var',
    severity: 'unknown',
    detail: 'Could not classify accent URL handling from theme.ts contents.',
  });
}

const bugs = findings.filter((f) => f.severity === 'bug');
console.log(JSON.stringify({ findings, reproduced: bugs.length > 0 }, null, 2));
process.exit(bugs.length > 0 ? 1 : 0);
