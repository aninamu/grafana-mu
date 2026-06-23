import { getImageCellSrc } from './imageCellUtils';

describe('getImageCellSrc', () => {
  it('should resolve paths under img through the Grafana public build path', () => {
    const originalPublicPath = window.__grafana_public_path__;
    const publicPath = 'https://grafana.fake/public/';
    const imagePath = 'img/icons/unicons/question-circle.svg';
    window.__grafana_public_path__ = publicPath;

    expect(getImageCellSrc(imagePath)).toBe(`${publicPath}build/${imagePath}`);

    window.__grafana_public_path__ = originalPublicPath;
  });

  it.each([
    'https://example.com/image.png',
    'http://example.com/image.png',
    '//cdn.example.com/image.png',
    'data:image/png;base64,abc123',
    'blob:https://example.com/abc123',
    '/img/icons/unicons/question-circle.svg',
  ])('should preserve %s', (src) => {
    expect(getImageCellSrc(src)).toBe(src);
  });
});
