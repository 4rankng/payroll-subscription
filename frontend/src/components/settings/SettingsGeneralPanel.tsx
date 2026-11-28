import { SettingsSection } from '@/components/settings/SettingsSection';
import { SettingCard } from '@/components/settings/SettingCard';
import type { SettingsFormState } from '@/hooks/settings/useSettingsForm';

interface SettingsGeneralPanelProps {
  form: SettingsFormState;
}

// PAYSUB (PRD §6.4): the reduced general panel carries the subscription fee
// only — donor-era percentage/wallet/bank/self-check-in cards are removed.
export const SettingsGeneralPanel = ({ form }: SettingsGeneralPanelProps) => {
  // PRD §5.9: K moves in 1.000 ₫ steps (max 999.999.999). The card formats
  // the value as text, so the step is enforced as a divisibility rule: the
  // inline error also disables the card's Save button (isInvalid).
  const rawDigits = form.subscriptionFeeVnd.replace(/\D/g, '');
  const feeStepError =
    form.subscriptionFeeVnd !== '' && rawDigits !== '' && Number(rawDigits) % 1000 !== 0
      ? 'K phải là bội số của 1.000 ₫'
      : null;

  return (
    <div className="columns-1 gap-3 xl:columns-2 [&>*]:mb-3 [&>*]:break-inside-avoid">
      <SettingsSection kicker="PAYSUB" title="Phí">
        <SettingCard
          title="Phí (đ/NV/kỳ)"
          description="Số tiền tính phí cho mỗi NV trong mỗi kỳ trả của một tháng làm việc. Bước nhảy 1.000 ₫."
          value={form.subscriptionFeeVnd}
          originalValue={form.originalSubscriptionFeeVnd}
          onChange={form.setSubscriptionFeeVnd}
          onSave={form.handleSaveSubscriptionFee}
          onReset={() => form.setSubscriptionFeeVnd(form.originalSubscriptionFeeVnd)}
          isDirty={form.subscriptionFeeVnd !== form.originalSubscriptionFeeVnd}
          isSaving={form.isSaving}
          displayMode="currency-vnd"
          min={0}
          max={999999999}
          step={1000}
          errorMessage={feeStepError}
        />
      </SettingsSection>
    </div>
  );
};
