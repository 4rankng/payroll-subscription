import { Button } from '@/components/ui/button';
// W13a follow-up (approved): icon slots widened from lucide's LucideIcon to
// plain ComponentType so both lucide and @untitledui/icons fit — LucideIcon
// is structurally assignable to this; UU's plain-FC icons were not to the old type.
import type { ComponentType, SVGProps } from 'react';
import { TooltipProvider } from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';

interface PageHeaderAction {
  label: string;
  onClick: () => void;
  icon?: ComponentType<SVGProps<SVGSVGElement>>;
  variant?: 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'link' | 'monochrome' | 'monochrome-outline';
  className?: string;
  disabled?: boolean;
}

interface PageHeaderProps {
  title: string;
  description?: string;
  /** Mono domain kicker above the title (e.g. getConsoleKicker(pathname)). */
  kicker?: string;
  /**
   * Sans-serif role eyebrow above the title (DASH-05: quanly → QUẢN LÝ…).
   * Rendered without the mono console-kicker treatment; takes precedence
   * visually only when kicker is absent.
   */
  eyebrow?: string;
  /** 11px mono metadata line (period + count + updated time). */
  meta?: string;
  /** 'mono' (default) keeps the 11px financial mono meta; 'sans' renders it as a 12px gray-500 sans subtitle. */
  metaVariant?: 'mono' | 'sans';
  actions?: PageHeaderAction[];
  children?: React.ReactNode;
  icon?: ComponentType<SVGProps<SVGSVGElement>>;
  className?: string;
}

export const PageHeader = ({
  title,
  description,
  kicker,
  eyebrow,
  meta,
  metaVariant = 'mono',
  actions = [],
  children,
  icon: Icon,
  className,
}: PageHeaderProps) => {
  const hasControls = actions.length > 0 || !!children;

  return (
    <TooltipProvider>
      <section
        data-slot="page-header"
        data-admin-surface="header"
        className={cn(
          // Console strip: 3px signal mark, hairline bottom — no card, no
          // shadow, no icon chip.
          'console-page-header flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-start sm:justify-between sm:gap-4',
          className,
        )}
      >
        <div className="min-w-0 flex-1 sm:min-w-64 sm:basis-64">
          <div className="flex items-center gap-2">
              {Icon && (
                <Icon className="h-5 w-5 shrink-0 text-muted-foreground" aria-hidden="true" />
              )}
              <div className="min-w-0">
                {eyebrow ? (
                  <p
                    data-slot="page-header-eyebrow"
                    className="mb-1 text-[11px] font-semibold leading-[1.35] tracking-[0.03em] text-gray-500"
                  >
                    {eyebrow}
                  </p>
                ) : null}
                {!eyebrow && kicker ? (
                  <p data-slot="page-header-eyebrow" className="console-kicker mb-1">
                    {kicker}
                  </p>
                ) : null}
                {!eyebrow && !kicker ? (
                  <p data-slot="page-header-eyebrow" className="hidden" />
                ) : null}
                <h1 className="font-display text-xl font-bold tracking-tight text-foreground sm:text-2xl">
                  {title}
                </h1>
                {meta && (
                  <p
                    data-slot="page-header-meta"
                    className={cn(
                      'mt-1',
                      metaVariant === 'sans'
                        ? 'text-xs text-gray-500'
                        : 'font-financial text-[11px] text-muted-foreground',
                    )}
                  >
                    {meta}
                  </p>
                )}
              </div>
          </div>
          {description && (
            <p data-slot="page-header-description" className={cn(
              'mt-1 text-sm text-muted-foreground leading-normal',
              Icon ? 'sm:ml-7' : '',
            )}>
              {description}
            </p>
          )}
        </div>

        {hasControls && (
          <div
            data-slot="page-header-controls"
            className="flex max-w-full shrink-0 flex-wrap items-center gap-2 sm:justify-end"
          >
            {actions.map((action, index) => (
              <Button
                key={index}
                onClick={action.onClick}
                variant={action.variant || 'default'}
                size="sm"
                className={action.className}
                disabled={action.disabled}
              >
                {action.icon && <action.icon className="h-3.5 w-3.5 mr-1" />}
                {action.label}
              </Button>
            ))}
            {children}
          </div>
        )}
      </section>
    </TooltipProvider>
  );
};
