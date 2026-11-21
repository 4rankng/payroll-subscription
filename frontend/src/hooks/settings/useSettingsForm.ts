import { useState, useEffect } from 'react';
import { useCreateSetting, useMultipleSettings, useUpdateSetting } from '@/hooks/api/useSettings';

// PAYSUB (PRD §6.4): the reduced Settings page tracks the subscription fee
// only — every donor-era percentage/bank/wallet/self-check-in key is dropped.
const SETTINGS_KEYS = {
  SUBSCRIPTION_FEE_PER_EMPLOYEE_PER_KY_VND: 'subscription_fee_per_employee_per_ky_vnd',
} as const;

export interface SettingsFormState {
  subscriptionFeeVnd: string;
  originalSubscriptionFeeVnd: string;
  setSubscriptionFeeVnd: (value: string) => void;
  handleSaveSubscriptionFee: () => void;
  isSaving: boolean;
  isLoading: boolean;
  loadError: string | null;
  retryLoading: () => void;
}

export function useSettingsForm(): SettingsFormState {
  const {
    data: settings,
    isError: isSettingsError,
    isLoading,
    refetch,
  } = useMultipleSettings([SETTINGS_KEYS.SUBSCRIPTION_FEE_PER_EMPLOYEE_PER_KY_VND]);

  const feeSetting = settings?.find((s) => s?.key === SETTINGS_KEYS.SUBSCRIPTION_FEE_PER_EMPLOYEE_PER_KY_VND);

  const [subscriptionFeeVnd, setSubscriptionFeeVnd] = useState('');
  const [originalSubscriptionFeeVnd, setOriginalSubscriptionFeeVnd] = useState('');

  // Sync display values once the settings query lands.
  useEffect(() => {
    if (feeSetting) {
      setSubscriptionFeeVnd(feeSetting.value ?? '');
      setOriginalSubscriptionFeeVnd(feeSetting.value ?? '');
    }
  }, [feeSetting]);

  const createMutation = useCreateSetting();
  const updateMutation = useUpdateSetting();
  const isSaving = createMutation.isPending || updateMutation.isPending;

  // Upsert: the fee row may not exist yet on an environment that never saved
  // it — the backend defaults the fee until an admin stores one.
  const handleSaveSubscriptionFee = () => {
    const value = subscriptionFeeVnd;
    const onSaved = () => setOriginalSubscriptionFeeVnd(value);
    if (feeSetting?.id) {
      updateMutation.mutate({ id: feeSetting.id, data: { value } }, { onSuccess: onSaved });
    } else {
      createMutation.mutate(
        { key: SETTINGS_KEYS.SUBSCRIPTION_FEE_PER_EMPLOYEE_PER_KY_VND, value, value_type: 'number' },
        { onSuccess: onSaved },
      );
    }
  };

  const loadError = isSettingsError ? 'Không thể tải cài đặt. Vui lòng thử lại.' : null;

  return {
    subscriptionFeeVnd,
    originalSubscriptionFeeVnd,
    setSubscriptionFeeVnd,
    handleSaveSubscriptionFee,
    isSaving,
    isLoading,
    loadError,
    retryLoading: () => refetch(),
  };
}
