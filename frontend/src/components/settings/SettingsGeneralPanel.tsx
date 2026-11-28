import { SettingsSection } from '@/components/settings/SettingsSection';
import { SettingCard } from '@/components/settings/SettingCard';
import type { SettingsFormState } from '@/hooks/settings/useSettingsForm';

interface SettingsGeneralPanelProps {
  form: SettingsFormState;
}

// PAYSUB (PRD §6.4): the reduced general panel carries the subscription fee
// only — donor-era percentage/wallet/bank/self-check-in cards are removed.
export const SettingsGeneralPanel = ({ form }: SettingsGeneralPanelProps) => (
  <div className="columns-1 gap-3 xl:columns-2 [&>*]:mb-3 [&>*]:break-inside-avoid">
    <SettingsSection kicker="PAYSUB" title="Phí">
      <SettingCard
        title="Phí (đ/NV/kỳ)"
        description="Số tiền tính phí cho mỗi NV trong mỗi kỳ trả của một tháng làm việc."
        value={form.subscriptionFeeVnd}
        originalValue={form.originalSubscriptionFeeVnd}
        onChange={form.setSubscriptionFeeVnd}
        onSave={form.handleSaveSubscriptionFee}
        onReset={() => form.setSubscriptionFeeVnd(form.originalSubscriptionFeeVnd)}
        isDirty={form.subscriptionFeeVnd !== form.originalSubscriptionFeeVnd}
        isSaving={form.isSaving}
        type="number"
        min={0}
        max={999999999}
        step={1000}
        suffix="đ"
      />
    </SettingsSection>
  </div>
);
