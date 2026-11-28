import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import SettingsPage from './index';
import SettingsPageMobile from '../../mobile/admin/SettingsPage';

const mocks = vi.hoisted(() => ({
  retryLoading: vi.fn(),
  formState: {
    subscriptionFeeVnd: '10000',
    originalSubscriptionFeeVnd: '10000',
    loadError: null as string | null,
    isSaving: false,
    isLoading: false,
    setSubscriptionFeeVnd: vi.fn((v: string) => {
      mocks.formState.subscriptionFeeVnd = v;
    }),
    handleSaveSubscriptionFee: vi.fn(),
    retryLoading: vi.fn(),
  },
}));

vi.mock('@/hooks/settings/useSettingsForm', () => ({
  useSettingsForm: () => mocks.formState,
}));

vi.mock('@/components/email/AdminEmailComposer', () => ({
  AdminEmailComposer: () => null,
}));

// The fee card reads the settings API directly; stubbed so the tab wiring
// and the fee card are under test.
vi.mock('@/hooks/api/useSettings', () => ({
  useSettings: () => ({ data: [], isLoading: false, isError: false, refetch: vi.fn() }),
  useSetting: () => ({ data: undefined, isLoading: false, isError: false, refetch: vi.fn() }),
  useSettingByKey: () => ({ data: undefined, isLoading: false, isError: false, refetch: vi.fn() }),
  useMultipleSettings: () => ({
    data: [undefined],
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useUpdateSetting: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useCreateSetting: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
}));

describe.each([
  ['desktop', SettingsPage],
  ['mobile', SettingsPageMobile],
] as const)('SettingsPage %s parity', (_view, PageComponent) => {
  beforeEach(() => {
    mocks.formState.loadError = null;
    mocks.formState.subscriptionFeeVnd = '10000';
    mocks.formState.originalSubscriptionFeeVnd = '10000';
    mocks.formState.retryLoading = mocks.retryLoading;
    mocks.formState.setSubscriptionFeeVnd.mockClear();
    mocks.formState.handleSaveSubscriptionFee.mockClear();
    mocks.retryLoading.mockClear();
  });

  it('renders the subscription fee setting with the full constraints', () => {
    render(
      <MemoryRouter>
        <PageComponent />
      </MemoryRouter>,
    );

    const feeInput = screen.getByLabelText('Phí (đ/NV/kỳ)') as HTMLInputElement;
    expect(feeInput).toHaveValue(10000);
    expect(feeInput).toHaveAttribute('min', '0');
    expect(feeInput).toHaveAttribute('max', '999999999');
    expect(feeInput).toHaveAttribute('step', '1000');
    expect(screen.getByText('Phí')).toBeInTheDocument();
  });

  it('saves a changed fee through the settings form pipeline', () => {
    const { rerender } = render(
      <MemoryRouter>
        <PageComponent />
      </MemoryRouter>,
    );

    fireEvent.change(screen.getByLabelText('Phí (đ/NV/kỳ)'), { target: { value: '12000' } });
    expect(mocks.formState.setSubscriptionFeeVnd).toHaveBeenCalledWith('12000');
    // The form state holder is a plain mock object: re-render to let the
    // card observe the dirty state the change handler just recorded.
    rerender(
      <MemoryRouter>
        <PageComponent />
      </MemoryRouter>,
    );
    const save = screen.getByRole('button', { name: 'Lưu thay đổi' });
    fireEvent.click(save);
    expect(mocks.formState.handleSaveSubscriptionFee).toHaveBeenCalledTimes(1);
  });

  it('shows a page-level retry state when any required setting cannot load', () => {
    mocks.formState.loadError = 'Không thể tải cài đặt. Vui lòng thử lại.';

    render(
      <MemoryRouter>
        <PageComponent />
      </MemoryRouter>,
    );

    expect(screen.getByRole('alert')).toHaveTextContent('Không thể tải cài đặt');
    expect(screen.queryByLabelText('Phí (đ/NV/kỳ)')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Thử lại' }));
    expect(mocks.retryLoading).toHaveBeenCalledOnce();
  });
});
