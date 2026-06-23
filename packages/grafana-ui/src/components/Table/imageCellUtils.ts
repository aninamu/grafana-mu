export function getImageCellSrc(src: string): string {
  if (!src.startsWith('img/')) {
    return src;
  }

  const publicPath = typeof window !== 'undefined' && window.__grafana_public_path__;
  return `${publicPath || 'public/'}build/${src}`;
}
