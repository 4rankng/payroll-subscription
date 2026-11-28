import type { ReactNode } from 'react';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { useBillingKySummary } from '@/hooks/api/useBillingDashboard';
import FeeInsightsPage from '.';

// Regression (staging report): a failing billing endpoint whose body carries
// no message (proxy 502/504 HTML, empty panic JSON) rendered the alert as
// "Không thể tải dữ liệu tổng quan: " with an empty tail. The page must fall
// back to the HTTP status or a connection notice.

vi.mock('@/hooks/api/useBillingDashboard', () => ({
  useBillingKySummary: vi.fn(),
}));

vi.mock('@/components/shared/PageHeader', () => ({
  PageHeader: ({ title }: { title: ReactNode }) => <h1>{title}</h1>,
}));

const mockHooks = () => {
  vi.mocked(useBillingKySummary).mockReturnValue({
    data: undefined,
    isLoading: false,
    isError: true,
    // Deliberately messageless: what axios rejects for a truthy-but-empty
    // error body (`error.response?.data || error`).
    error: { status: 'error', http_status: 502 },
  } as unknown as ReturnType<typeof useBillingKySummary>);
};

describe('FeeInsightsPage error alert', () => {
  it('falls back to the HTTP status when the error body has no message', () => {
    mockHooks();
    render(<MemoryRouter><FeeInsightsPage /></MemoryRouter>);

    const alert = screen.getByText(/Không thể tải dữ liệu tổng quan:/);
    expect(alert).toHaveTextContent('Lỗi máy chủ (HTTP 502)');
  });

  it('shows the server message when the error body carries one', () => {
    mockHooks();
    vi.mocked(useBillingKySummary).mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      error: { status: 'error', message: 'Kỳ chưa chốt', http_status: 409 },
    } as unknown as ReturnType<typeof useBillingKySummary>);
    render(<MemoryRouter><FeeInsightsPage /></MemoryRouter>);

    const alert = screen.getByText(/Không thể tải dữ liệu tổng quan:/);
    expect(alert).toHaveTextContent('Kỳ chưa chốt');
  });

  it('falls back to a connection notice when there is no HTTP status', () => {
    mockHooks();
    vi.mocked(useBillingKySummary).mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      error: { status: 'error' },
    } as unknown as ReturnType<typeof useBillingKySummary>);
    render(<MemoryRouter><FeeInsightsPage /></MemoryRouter>);

    const alert = screen.getByText(/Không thể tải dữ liệu tổng quan:/);
    expect(alert).toHaveTextContent('Lỗi kết nối đến máy chủ.');
  });
});
