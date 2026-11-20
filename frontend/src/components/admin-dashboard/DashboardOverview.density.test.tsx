import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import {
  DashboardMetricStrip,
  DashboardPriorityList,
} from './DashboardOverview';

describe('DashboardOverview density', () => {
  it('uses compact metric cells on desktop', () => {
    render(
      <DashboardMetricStrip
        items={[
          {
            label: 'Đã trả kỳ này',
            value: '12.500.000 ₫',
            context: 'Tổng đã trả 40.000.000 ₫',
            tone: 'primary',
          },
        ]}
      />,
    );

    expect(screen.getByText('Đã trả kỳ này').parentElement?.parentElement).toHaveClass(
      'px-3',
      'py-2.5',
      'sm:px-4',
    );
  });

  it('keeps touch-safe actions while reducing desktop row height', () => {
    render(
      <DashboardPriorityList
        items={[
          {
            title: 'Bảng công chờ duyệt',
            detail: 'Xử lý trước khi khóa sổ lương.',
            value: '3',
            statusLabel: 'Cần duyệt',
            icon: () => null,
            tone: 'warning',
          },
        ]}
        isRefreshing={false}
        onRefresh={() => {}}
      />,
    );

    expect(screen.getByRole('button', { name: 'Làm mới số liệu' })).toHaveClass(
      'min-h-11',
      'sm:min-h-9',
    );
    expect(screen.getByRole('button', { name: /Bảng công chờ duyệt/ })).toHaveClass(
      'min-h-[64px]',
      'sm:min-h-[52px]',
    );
  });
});
