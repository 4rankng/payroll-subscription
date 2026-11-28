import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import type { PaidSeriesRow } from '@/services/api/billing.service';
import {
  PaidSeriesLegend,
  buildPaidSeriesDefs,
} from './PaidSeriesChart';

vi.mock('@/hooks/api/useProjects', () => ({
  useAllProjects: () => ({ data: undefined }),
}));
vi.mock('@/hooks/api/useBilling', () => ({
  usePaidSeries: () => ({ data: [], isLoading: false }),
}));

const row = (projectId: number, amount: number, employeeCount = 1): PaidSeriesRow => ({
  date: '2026-11-01',
  project_id: projectId,
  amount,
  employee_count: employeeCount,
});

describe('buildPaidSeriesDefs', () => {
  it('ranks projects by paid amount and folds the tail into Khác', () => {
    const rows = [
      row(1, 5_000_000),
      row(2, 9_000_000),
      row(3, 1_000_000),
      row(4, 7_000_000),
      row(5, 3_000_000),
      row(6, 8_000_000),
      row(7, 2_000_000),
      row(8, 6_000_000),
    ];

    const defs = buildPaidSeriesDefs(rows, [], 6);

    expect(defs).toHaveLength(7);
    expect(defs.slice(0, 6).map((d) => d.projectIds[0])).toEqual([2, 6, 4, 8, 1, 5]);
    const other = defs[6];
    expect(other.name).toBe('Khác (2 dự án)');
    expect(other.projectIds).toEqual([7, 3]);
  });

  it('keeps every project named when within the top-N budget', () => {
    const defs = buildPaidSeriesDefs([row(1, 100), row(2, 200)], [], 6);

    expect(defs.map((d) => d.name)).toEqual(['(dự án)', '(dự án) 2']);
    expect(defs.map((d) => d.projectIds[0])).toEqual([2, 1]);
  });

  it('prefers projects with amounts, falling back to headcount order', () => {
    const rows = [row(1, 0, 10), row(2, 0, 3), row(3, 500_000, 0)];

    const defs = buildPaidSeriesDefs(rows, [], 6);

    expect(defs.map((d) => d.projectIds[0])).toEqual([3, 1, 2]);
  });
});

describe('PaidSeriesLegend', () => {
  it('renders one chip per shared series in normal flow (no SVG legend)', () => {
    const defs = buildPaidSeriesDefs(
      Array.from({ length: 10 }, (_, i) => row(i + 1, (10 - i) * 1_000_000)),
      [],
    );

    render(<PaidSeriesLegend series={defs} />);

    expect(screen.getAllByRole('listitem')).toHaveLength(9);
    expect(screen.getByText('Khác (2 dự án)')).toBeInTheDocument();
  });

  it('renders nothing without series', () => {
    const { container } = render(<PaidSeriesLegend series={[]} />);
    expect(container).toBeEmptyDOMElement();
  });
});
