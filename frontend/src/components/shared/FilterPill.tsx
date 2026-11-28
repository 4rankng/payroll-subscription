import * as React from 'react';
import * as DropdownMenuPrimitive from '@radix-ui/react-dropdown-menu';
import { Check, ChevronDown } from '@untitledui/icons';
import { cn } from '@/lib/utils';

export interface FilterOption {
  value: string;
  label: string;
}

export interface FilterPillProps {
  value: string;
  onChange: (value: string) => void;
  defaultValue?: string;
  placeholder: string;
  options: FilterOption[];
  icon?: React.ReactNode;
  className?: string;
  /** Extra classes for the dropdown content (overlay surface). */
  contentClassName?: string;
}

/**
 * Inline filter dropdown built on Radix DropdownMenu with `modal={false}`.
 *
 * The previous Radix Select implementation kept Radix's
 * `disableOutsidePointerEvents: true` while open, so the first outside click
 * only closed the menu and never reached its target — the "+ Thêm người dùng"
 * button and table rows swallowed clicks whenever a filter dropdown was open.
 * A non-modal menu dismisses on outside pointer-down AND lets that same click
 * activate the element under the pointer.
 */
export const FilterPill = React.memo(function FilterPill({
  value,
  onChange,
  defaultValue = 'all',
  placeholder,
  options,
  icon,
  className,
  contentClassName,
}: FilterPillProps) {
  const isActive = value !== defaultValue;
  const activeLabel = options.find(o => o.value === value)?.label;

  return (
    <DropdownMenuPrimitive.Root modal={false}>
      <DropdownMenuPrimitive.Trigger
        type="button"
        className={cn(
          'group inline-flex min-h-11 items-center gap-1.5 rounded-lg px-3 py-0 text-sm font-medium whitespace-nowrap sm:min-h-9 sm:px-2.5',
          'border transition-colors duration-100 select-none outline-none',
          'focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1',
          !isActive && 'border-border/60 bg-card text-muted-foreground hover:border-border hover:text-foreground hover:bg-accent/40',
          isActive && 'border-primary/30 bg-primary/8 text-primary hover:bg-primary/12',
          className,
        )}
        aria-label={placeholder}
      >
        {icon && <span className="shrink-0 opacity-70">{icon}</span>}
        <span>{isActive ? activeLabel : placeholder}</span>
        <ChevronDown className="h-3.5 w-3.5 shrink-0 opacity-50 group-data-[state=open]:rotate-180 transition-transform duration-150" aria-hidden="true" />
      </DropdownMenuPrimitive.Trigger>

      <DropdownMenuPrimitive.Portal>
        <DropdownMenuPrimitive.Content
          side="bottom"
          align="start"
          sideOffset={4}
          className={cn(
            'z-40 min-w-[10rem] overflow-hidden rounded-lg border bg-popover text-popover-foreground shadow-sm',
            'max-h-[min(20rem,var(--radix-dropdown-menu-content-available-height))]',
            'data-[state=open]:animate-in data-[state=closed]:animate-out',
            'data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0',
            'data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95',
            'data-[side=bottom]:slide-in-from-top-2',
            contentClassName,
          )}
        >
          <div className="max-h-[inherit] overflow-y-auto overscroll-contain p-1">
            <DropdownMenuPrimitive.RadioGroup value={value} onValueChange={onChange}>
              <FilterPillItem value={defaultValue} label={placeholder} />
              {options.map(o => (
                <FilterPillItem key={o.value} value={o.value} label={o.label} />
              ))}
            </DropdownMenuPrimitive.RadioGroup>
          </div>
        </DropdownMenuPrimitive.Content>
      </DropdownMenuPrimitive.Portal>
    </DropdownMenuPrimitive.Root>
  );
});

const FilterPillItem = React.memo(function FilterPillItem({
  value,
  label,
}: {
  value: string;
  label: string;
}) {
  return (
    <DropdownMenuPrimitive.RadioItem
      value={value}
      className={cn(
        'relative flex min-h-11 w-full cursor-default select-none items-center rounded-md py-2 pl-7 pr-3 text-sm outline-none whitespace-nowrap sm:min-h-8 sm:py-1.5',
        'focus:bg-accent focus:text-accent-foreground',
        'data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
      )}
    >
      <span className="absolute left-2 flex h-4 w-4 items-center justify-center">
        <DropdownMenuPrimitive.ItemIndicator>
          <Check className="h-3.5 w-3.5" aria-hidden="true" />
        </DropdownMenuPrimitive.ItemIndicator>
      </span>
      {label}
    </DropdownMenuPrimitive.RadioItem>
  );
});
