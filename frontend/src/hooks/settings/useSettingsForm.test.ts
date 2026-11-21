import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useSettingsForm } from './useSettingsForm';

const mocks = vi.hoisted(() => ({
  mutate: vi.fn(),
  mutateAsync: vi.fn(),
  refetch: vi.fn(),
  isError: false,
  settings: [
    {
      id: 1,
      key: 'subscription_fee_per_employee_per_ky_vnd',
      value: '10000',
      value_type: 'number',
    },
  ],
}));

vi.mock('@/hooks/api/useSettings', () => ({
  useMultipleSettings: () => ({
    data: mocks.settings,
    error: null,
    isError: mocks.isError,
    isLoading: false,
    refetch: mocks.refetch,
  }),
  useUpdateSetting: () => ({
    mutate: mocks.mutate,
    mutateAsync: mocks.mutateAsync,
    isPending: false,
  }),
  useCreateSetting: () => ({
    mutate: mocks.mutate,
    isPending: false,
  }),
}));

describe('useSettingsForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.isError = false;
    mocks.settings.splice(0, mocks.settings.length, {
      id: 1,
      key: 'subscription_fee_per_employee_per_ky_vnd',
      value: '10000',
      value_type: 'number',
    });
  });

  it('loads the subscription fee from the settings row', async () => {
    const { result } = renderHook(() => useSettingsForm());
    await waitFor(() => expect(result.current.subscriptionFeeVnd).toBe('10000'));
    expect(result.current.originalSubscriptionFeeVnd).toBe('10000');
  });

  it('saves the fee through the update mutation with the stored row id', async () => {
    const { result } = renderHook(() => useSettingsForm());
    await waitFor(() => expect(result.current.originalSubscriptionFeeVnd).toBe('10000'));

    act(() => result.current.setSubscriptionFeeVnd('12000'));
    act(() => result.current.handleSaveSubscriptionFee());

    expect(mocks.mutate).toHaveBeenCalledWith(
      { id: 1, data: { value: '12000' } },
      expect.anything(),
    );
  });

  it('creates the fee row when the environment never saved one', async () => {
    mocks.settings.splice(0, mocks.settings.length);
    const { result } = renderHook(() => useSettingsForm());
    await waitFor(() => expect(result.current.isLoading).toBe(false));

    act(() => result.current.setSubscriptionFeeVnd('10000'));
    act(() => result.current.handleSaveSubscriptionFee());

    expect(mocks.mutate).toHaveBeenCalledWith(
      { key: 'subscription_fee_per_employee_per_ky_vnd', value: '10000', value_type: 'number' },
      expect.anything(),
    );
  });

  it('exposes a recoverable load error when the settings query fails', async () => {
    mocks.isError = true;
    const { result } = renderHook(() => useSettingsForm());
    await waitFor(() => expect(result.current.loadError).not.toBeNull());
    act(() => result.current.retryLoading());
    expect(mocks.refetch).toHaveBeenCalled();
  });
});
