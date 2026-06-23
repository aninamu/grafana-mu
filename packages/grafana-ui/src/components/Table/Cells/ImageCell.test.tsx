import { render } from '@testing-library/react';

import { type Field, FieldType } from '@grafana/data';

import { TableCellDisplayMode, type TableCellProps } from '../types';

import { ImageCell } from './ImageCell';

function renderImageCell(src: string) {
  const field: Field = {
    name: 'image',
    type: FieldType.string,
    config: {
      custom: {
        cellOptions: {
          type: TableCellDisplayMode.Image,
        },
      },
    },
    values: [src],
    display: jest.fn(() => ({
      text: src,
      numeric: NaN,
    })),
  };

  return render(
    <ImageCell
      field={field}
      cell={{ value: src } as TableCellProps['cell']}
      tableStyles={
        {
          cellHeight: 30,
          imageCell: 'imageCell',
          cellContainer: 'cellContainer',
        } as TableCellProps['tableStyles']
      }
      row={{} as TableCellProps['row']}
      cellProps={{ style: {} }}
    />
  );
}

describe('ImageCell', () => {
  it('should resolve src values under img through the Grafana public build path', () => {
    const originalPublicPath = window.__grafana_public_path__;
    const publicPath = 'https://grafana.fake/public/';
    const imagePath = 'img/icons/unicons/question-circle.svg';
    window.__grafana_public_path__ = publicPath;

    const { container } = renderImageCell(imagePath);
    const img = container.querySelector('img');

    expect(img).toHaveAttribute('src', `${publicPath}build/${imagePath}`);

    window.__grafana_public_path__ = originalPublicPath;
  });
});
