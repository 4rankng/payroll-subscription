import { memo } from 'react';
import { SearchBar } from '@/components/shared/SearchBar';
import { getCurrentMonthValue } from '@/utils/bankTransferHistoryHelpers';
import type { PaymentHistoryFilters as FiltersType } from '@/types/api/payroll.types';

/** PRD §5.7 cycle windows — labels mirror the Kỳ payment windows. */
const KY_OPTIONS = [
  { value: 0, label: 'Tất cả kỳ' },
  { value: 1, label: 'Kỳ 1 · ngày 1–7' },
  { value: 2, label: 'Kỳ 2 · ngày 8–14' },
  { value: 3, label: 'Kỳ 3 · ngày 15–21' },
  { value: 4, label: 'Kỳ 4 · ngày 22–cuối tháng' },
] as const;

const SELECT_CLASS =
  'h-9 rounded-lg border border-input bg-card px-2.5 text-[12px] font-medium text-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring';

interface PaymentHistoryFiltersProps {
  filters: FiltersType;
  onFiltersChange: (filters: FiltersType) => void;
  onClearFilters: () => void;
  hasFilters: boolean;
  totalResults?: number;
}

/**
 * PRD §5.7 filter set: work month (YYYY-MM, default current), Kỳ 1-4 and a
 * free-text search over name / CCCD / transfer note / bank transaction code.
 */
export const PaymentHistoryFilters = memo(function PaymentHistoryFilters({
  filters,
  onFiltersChange,
  onClearFilters,
  hasFilters,
}: PaymentHistoryFiltersProps) {
  const set = (patch: Partial<FiltersType>) =>
    onFiltersChange({ ...filters, ...patch, page: 1 });

  return (
    <div className="console-filter-bar overflow-x-auto">
      <label className="flex min-h-[36px] shrink-0 items-center gap-1.5 text-[12px] font-medium text-muted-foreground">
        Tháng
        <input
          type="month"
          value={filters.month ?? getCurrentMonthValue()}
          onChange={(event) => set({ month: event.target.value || undefined })}
          max={getCurrentMonthValue()}
          className={`${SELECT_CLASS} [color-scheme:light] tabular-nums`}
          aria-label="Chọn tháng đối chiếu"
        />
      </label>

      <label className="flex min-h-[36px] shrink-0 items-center gap-1.5 text-[12px] font-medium text-muted-foreground">
        Kỳ
        <select
          value={filters.ky ?? 0}
          onChange={(event) => {
            const ky = Number(event.target.value);
            set({ ky: ky > 0 ? ky : undefined });
          }}
          className={SELECT_CLASS}
          aria-label="Lọc theo kỳ thanh toán"
        >
          {KY_OPTIONS.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
      </label>

      <SearchBar
        searchTerm={filters.search ?? ''}
        onSearchChange={(value) => set({ search: value || undefined })}
        placeholder="Tìm tên, CCCD, mã CK, mã GD…"
        className="min-w-[220px] flex-1"
      />

      {hasFilters && (
        <button
          type="button"
          onClick={onClearFilters}
          className="shrink-0 text-[12px] font-medium text-muted-foreground underline-offset-2 hover:text-foreground hover:underline"
        >
          Xóa lọc
        </button>
      )}
    </div>
  );
});
