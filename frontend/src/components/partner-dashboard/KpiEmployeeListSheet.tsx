import { useMemo } from 'react';
import { useEmployees } from '@/hooks/api/useEmployees';
import { UserAvatar } from '@/components/ui/user-avatar';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { EmptyState } from '@/components/shared/EmptyState';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from '@/components/ui/sheet';

export type KpiEmployeeCategory = 'all' | 'working' | 'new';

interface KpiEmployeeListSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  category: KpiEmployeeCategory;
  month?: string;
  onEmployeeClick?: (employeeId: number) => void;
}

const CATEGORY_LABELS: Record<KpiEmployeeCategory, string> = {
  all: 'Tất cả nhân viên',
  working: 'Đang làm việc',
  new: 'Nhân viên mới',
};

function getCategoryFilter(category: KpiEmployeeCategory, month?: string) {
  switch (category) {
    case 'working':
      return { status: 'working' as const, pageSize: 100 };
    case 'new':
      return { pageSize: 100, ...(month ? { month } : {}) };
    case 'all':
    default:
      return { pageSize: 100 };
  }
}

export function KpiEmployeeListSheet({
  open,
  onOpenChange,
  category,
  month,
  onEmployeeClick,
}: KpiEmployeeListSheetProps) {
  const filters = useMemo(() => getCategoryFilter(category, month), [category, month]);
  const { data, isLoading } = useEmployees(open ? filters : undefined);
  const employees = useMemo(() => {
    if (!data) return [];
    return Array.isArray(data) ? data : ((data as { employees?: unknown[] }).employees ?? []);
  }, [data]);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="bottom" className="max-h-[80vh] flex flex-col">
        <SheetHeader>
          <SheetTitle>{CATEGORY_LABELS[category]}</SheetTitle>
          <SheetDescription>
            {employees.length > 0
              ? `${employees.length} nhân viên`
              : 'Danh sách nhân viên'}
          </SheetDescription>
        </SheetHeader>
        <div className="flex-1 overflow-y-auto px-4 pb-6">
          {isLoading ? (
            <div className="space-y-3 pt-2">
              {Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="flex items-center gap-3">
                  <Skeleton className="size-10 rounded-full" />
                  <div className="flex-1 space-y-1.5">
                    <Skeleton className="h-3.5 w-32" />
                    <Skeleton className="h-2.5 w-24" />
                  </div>
                </div>
              ))}
            </div>
          ) : employees.length === 0 ? (
            <EmptyState
              showIllustration={false}
              title="Không có nhân viên"
              description="Không tìm thấy nhân viên nào trong danh mục này."
              size="sm"
            />
          ) : (
            <ul className="space-y-1">
              {employees.map((emp: { id: number; fullname?: string; full_name?: string; cccd?: string; status?: string; current_project?: { name: string } | null }) => {
                const name = emp.fullname ?? emp.full_name ?? `NV #${emp.id}`;
                return (
                  <li key={emp.id}>
                    <button
                      type="button"
                      onClick={() => onEmployeeClick?.(emp.id)}
                      className="flex w-full items-center gap-3 rounded-md px-2 py-2.5 text-left transition-colors hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring min-h-[44px]"
                    >
                      <UserAvatar name={name} size="sm" />
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium">{name}</p>
                        <p className="text-xs text-muted-foreground truncate">
                          {emp.cccd ?? ''}
                          {emp.current_project ? ` · ${emp.current_project.name}` : ''}
                        </p>
                      </div>
                      {emp.status && (
                        <Badge variant={emp.status === 'active' ? 'default' : 'secondary'} className="shrink-0 text-xs">
                          {emp.status === 'active' ? 'Hoạt động' : 'Nghỉ'}
                        </Badge>
                      )}
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}
