import { useState, useCallback } from 'react';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { MonthPicker } from '@/components/ui/month-picker';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { format, addMonths, subMonths, startOfMonth, parse } from 'date-fns';
import { cn } from '@/lib/utils';

interface TimesheetMonthSelectorProps {
  value: string;
  onChange: (value: string) => void;
  className?: string;
  /**
   * Month-scoped consumers (Sổ cái) whose queries cannot span months pass
   * false to hide the "Tất cả" option entirely — selecting it there produced
   * a "Tháng all · 0 file" dead state.
   */
  allowAll?: boolean;
}

export const TimesheetMonthSelector: React.FC<TimesheetMonthSelectorProps> = ({
  value,
  onChange,
  className,
  allowAll = true,
}) => {
  const [isCalendarOpen, setIsCalendarOpen] = useState(false);

  const selectedDate = value !== 'all'
    ? startOfMonth(parse(value + '-01', 'yyyy-MM-dd', new Date()))
    : startOfMonth(new Date());

  const handlePreviousMonth = useCallback(() => {
    const current = value !== 'all' ? selectedDate : new Date();
    onChange(format(subMonths(current, 1), 'yyyy-MM'));
  }, [value, selectedDate, onChange]);

  const handleNextMonth = useCallback(() => {
    const current = value !== 'all' ? selectedDate : new Date();
    onChange(format(addMonths(current, 1), 'yyyy-MM'));
  }, [value, selectedDate, onChange]);

  const handleMonthSelect = useCallback((month: string) => {
    onChange(month);
    setIsCalendarOpen(false);
  }, [onChange]);

  const handleShowAll = useCallback(() => {
    onChange('all');
  }, [onChange]);

  // "T5/2026" compact label (PAYR-134); "Chọn tháng" when Tất cả
  const monthShort = `T${format(selectedDate, 'M/yyyy')}`;

  return (
    <div
      className={cn(
        'grid w-full grid-cols-[auto_auto_minmax(0,1fr)_auto] overflow-hidden rounded-xl border border-border/80 bg-card shadow-sm sm:w-auto',
        className,
      )}
    >
      {allowAll && (
        <button
          type="button"
          onClick={handleShowAll}
          className={cn(
            'h-11 border-r border-border/70 px-3 text-xs font-semibold transition-colors whitespace-nowrap sm:px-2.5 sm:h-8',
            value === 'all'
              ? 'bg-foreground text-background'
              : 'bg-card text-muted-foreground hover:bg-muted hover:text-foreground',
          )}
        >
          Tất cả
        </button>
      )}

      <button
        type="button"
        onClick={handlePreviousMonth}
        aria-label="Tháng trước"
        className="-ml-px flex h-11 w-11 shrink-0 items-center justify-center text-muted-foreground transition-colors hover:bg-muted hover:text-foreground sm:h-8 sm:w-8"
      >
        <ChevronLeft className="h-3.5 w-3.5" />
      </button>

      <Popover open={isCalendarOpen} onOpenChange={setIsCalendarOpen}>
        <PopoverTrigger asChild>
          <button
            type="button"
            className="-ml-px flex h-11 min-w-0 items-center justify-center bg-muted/35 px-2.5 transition-colors hover:bg-muted sm:h-8"
            aria-label="Chọn tháng"
          >
            {value === 'all' ? (
              <span className="whitespace-nowrap text-xs font-semibold">Chọn tháng</span>
            ) : (
              <span className="whitespace-nowrap text-xs font-semibold tabular-nums text-foreground">
                {monthShort}
              </span>
            )}
          </button>
        </PopoverTrigger>
        <PopoverContent className="w-auto p-0" align="end">
          <MonthPicker value={value} onChange={handleMonthSelect} />
        </PopoverContent>
      </Popover>

      <button
        type="button"
        onClick={handleNextMonth}
        aria-label="Tháng sau"
        className="-ml-px flex h-11 w-11 shrink-0 items-center justify-center text-muted-foreground transition-colors hover:bg-muted hover:text-foreground sm:h-8 sm:w-8"
      >
        <ChevronRight className="h-3.5 w-3.5" />
      </button>
    </div>
  );
};
