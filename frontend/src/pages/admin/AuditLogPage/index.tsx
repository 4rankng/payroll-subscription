import { ErrorState } from "@/components/ui/error-state";
import React, { useState, useCallback, useMemo } from 'react';
import { useAppState } from '@/contexts';
// UU PRO restyle (W13c): lucide → verified @untitledui/icons.
import { Clipboard, Loading01, ChevronRight } from '@untitledui/icons';
import { format } from 'date-fns';
import { cn } from '@/lib/utils';
import { useLocation } from 'react-router-dom';
import { useInfiniteAuditLogs } from '@/hooks/api/useAuditLogs';
import { getConsoleKicker } from '@/lib/console';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll.tsx';
import {
  AdminPageCanvas,
  AdminPageHeaderCard,
  AdminSectionCard,
  AdminFilterRow,
} from '@/components/shared/AdminPageFrame';
import { PageHeader } from '@/components/shared/PageHeader';
import { EmptyState } from '@/components/shared/EmptyState';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { AuditLogFilters } from './AuditLogFilters';
import { AuditLogCardSkeleton } from './AuditLogCard';
import {
  getActionLabel,
  getActionVariant,
  getEntityLabel,
  extractLocation,
  VARIANT_CLASSES,
} from './utils';
import { AuditLogDetailSheet } from './AuditLogDetailSheet';
import type { AuditLogsQueryParams } from '@/types/api/audit.types';
import type { BackendAuditLog } from '@/types/api/audit.types';

type Filters = Omit<AuditLogsQueryParams, 'page' | 'pageSize'>;

export default function AuditLogPage() {
  const { setPageTitle } = useAppState();
  const location = useLocation();
  React.useEffect(() => { setPageTitle('Nhật ký hoạt động'); }, [setPageTitle]);

  const [filters, setFilters] = useState<Filters>({});
  const [selectedLogId, setSelectedLogId] = useState<number | null>(null);

  const {
    data,
    isLoading,
    isError,
    isFetchNextPageError,
    refetch,
    isFetchingNextPage,
    fetchNextPage,
    hasNextPage,
  } = useInfiniteAuditLogs(filters);

  const logs = useMemo<BackendAuditLog[]>(
    () => data?.pages.flatMap((p) => p.data ?? []) ?? [],
    [data]
  );

  const totalRecords = data?.pages[0]?.pagination?.totalRecords ?? 0;

  const { observerRef } = useInfiniteScroll({
    hasMore: !isError && (hasNextPage ?? false),
    isLoading: isFetchingNextPage,
    onLoadMore: () => { fetchNextPage(); },
    rootMargin: '300px',
    threshold: 0.1,
  });

  const handleFiltersChange = useCallback((next: Filters) => {
    setFilters(next);
  }, []);

  return (
    <AdminPageCanvas>
      <AdminPageHeaderCard>
        <PageHeader
          icon={Clipboard}
          title="Nhật ký hoạt động"
          kicker={getConsoleKicker(location.pathname)}
          meta={totalRecords > 0 ? `${totalRecords.toLocaleString('vi-VN')} bản ghi` : undefined}
        />
      </AdminPageHeaderCard>

      {/* Filters */}
      <AdminSectionCard aria-label="Bộ lọc nhật ký">
        <AdminFilterRow>
          <p className="text-sm font-semibold text-fg-primary">Nhật ký</p>
          <div className="flex flex-1 items-center justify-end">
            <AuditLogFilters filters={filters} onChange={handleFiltersChange} />
          </div>
        </AdminFilterRow>
      </AdminSectionCard>

      {isError && (
        <div role="alert">
          <ErrorState
            message={isFetchNextPageError
              ? "Không thể tải thêm nhật ký. Các bản ghi đã tải vẫn được giữ lại."
              : "Không thể tải nhật ký hoạt động. Vui lòng thử lại."}
            onRetry={() => void (isFetchNextPageError ? fetchNextPage() : refetch())}
            className="px-4"
          />
        </div>
      )}

      {/* Initial loading */}
      {isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <AuditLogCardSkeleton key={i} />
          ))}
        </div>
      )}

        {/* Cards */}
        {!isLoading && !isError && logs.length === 0 && (
          <EmptyState
            title="Không có dữ liệu"
            description="Thử thay đổi bộ lọc để xem kết quả khác"
            size="sm"
          />
        )}

        {!isLoading && logs.length > 0 && (
          // Dense console table (34px header) — the stacked-card list wasted
          // vertical space for a data-dense admin surface. The detail sheet
          // keeps every field.
          <div className="console-panel overflow-hidden [&_.console-table_thead_tr]:h-[34px] [&_.console-table_thead_th]:h-[34px] [&_.console-table_thead_th]:py-[3px]">
            <div className="overflow-x-auto">
              <Table variant="console">
                <TableHeader>
                  <TableRow>
                    <TableHead className="whitespace-nowrap">Thời gian</TableHead>
                    <TableHead className="whitespace-nowrap">Người dùng</TableHead>
                    <TableHead className="whitespace-nowrap">Hành động</TableHead>
                    <TableHead className="whitespace-nowrap">Đối tượng</TableHead>
                    <TableHead className="whitespace-nowrap">IP · Thiết bị</TableHead>
                    <TableHead className="w-8"><span className="sr-only">Chi tiết</span></TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {logs.map((log) => {
                    const variant = getActionVariant(log.action);
                    const location = extractLocation(log.metadata);
                    const date = new Date(log.created_at);
                    return (
                      <TableRow
                        key={log.id}
                        onClick={() => setSelectedLogId(log.id)}
                        className="cursor-pointer"
                      >
                        <TableCell className="whitespace-nowrap tabular-nums text-muted-foreground">
                          {format(date, 'dd/MM/yyyy')}{' · '}
                          {date.toLocaleTimeString('vi-VN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })}
                        </TableCell>
                        <TableCell className="min-w-0">
                          <span className="block truncate font-medium text-foreground" title={log.user_fullname || log.user_username}>
                            {log.user_fullname || log.user_username || '—'}
                          </span>
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant="outline"
                            className={cn('shrink-0 border px-2 py-0 text-[11px] font-semibold', VARIANT_CLASSES[variant])}
                          >
                            {getActionLabel(log.action)}
                          </Badge>
                        </TableCell>
                        <TableCell className="whitespace-nowrap text-muted-foreground">
                          {getEntityLabel(log.entity_type)}
                          {log.entity_id ? ` · ${log.entity_id}` : ''}
                        </TableCell>
                        <TableCell className="min-w-0">
                          <span className="block truncate text-muted-foreground" title={[log.ip_address, location ? [location.city, location.country].filter(Boolean).join(', ') : null, [log.browser, log.platform].filter(Boolean).join(' · ')].filter(Boolean).join(' · ')}>
                            {[[log.ip_address, location ? [location.city, location.country].filter(Boolean).join(', ') : null].filter(Boolean).join(' · '), [log.browser, log.platform].filter(Boolean).join(' · ')].filter(Boolean).join(' — ') || '—'}
                          </span>
                        </TableCell>
                        <TableCell>
                          <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" aria-hidden="true" />
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>
          </div>
        )}

        {/* Infinite scroll sentinel */}
        {!isError && hasNextPage && !isFetchingNextPage && (
          <div ref={observerRef} className="h-px w-full" aria-hidden="true" />
        )}

        {/* Loading more */}
        {isFetchingNextPage && (
          <div className="flex items-center justify-center py-6">
            <Loading01 className="h-5 w-5 animate-spin text-muted-foreground" />
            <span className="ml-2 text-sm text-muted-foreground">Đang tải thêm...</span>
          </div>
        )}

        {/* End of list */}
        {!isError && !hasNextPage && !isLoading && logs.length > 0 && (
          <p className="text-center py-4 text-xs text-muted-foreground">
            Đã hiển thị tất cả {logs.length.toLocaleString('vi-VN')} bản ghi
          </p>
        )}

      <AuditLogDetailSheet
        logId={selectedLogId}
        onClose={() => setSelectedLogId(null)}
      />
    </AdminPageCanvas>
  );
}
