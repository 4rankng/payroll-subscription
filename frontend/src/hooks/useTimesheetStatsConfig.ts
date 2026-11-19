import { useMemo } from 'react';
import {
  Calendar,
  CheckSquare,
  AlertCircle,
  XCircle,
  Users,
  DollarSign,
  Clock,
  Wallet
} from 'lucide-react';
import { formatCurrency } from '@/utils/formatters';
import { useTimesheetSummary } from '@/hooks/api/useTimesheets';
import type { StatCardConfig, StatGroupConfig } from '@/components/shared/SummaryStatsCards';

interface UseTimesheetStatsConfigParams {
  project_id?: number;
  employee_id?: number;
  fromDate?: string;
  toDate?: string;
}

export const useTimesheetStatsConfig = (params?: UseTimesheetStatsConfigParams) => {
  const { data: summary, isLoading, error } = useTimesheetSummary(params);

  // Flat stats config (backward compatible)
  const statsConfig: StatCardConfig[] = useMemo(() => {
    if (!summary) return [];

    return [
      {
        title: 'Tổng công',
        value: summary.totalEntries,
        icon: Calendar,
      },
      {
        title: 'Tổng NV',
        value: summary.totalEmployees ?? 0,
        icon: Users,
      },
      {
        title: 'Chờ duyệt',
        value: summary.pendingApproval,
        icon: AlertCircle,
      },
      {
        title: 'NV chờ thanh toán',
        value: summary.pendingEmployees ?? 0,
        icon: Clock,
      },
      {
        title: 'Đã duyệt',
        value: summary.approvedEntries,
        icon: CheckSquare,
      },
      {
        title: 'Đã thanh toán',
        value: summary.paidEntries,
        icon: DollarSign,
      },
      {
        title: 'Bị loại',
        value: summary.rejectedEntries,
        icon: XCircle,
      },
      {
        title: 'Chờ thanh toán',
        value: summary.pendingPaymentAmount ? formatCurrency(summary.pendingPaymentAmount) : formatCurrency(0),
        icon: Wallet,
      },
    ];
  }, [summary]);

  // Grouped stats config for enhanced visual organization
  const groupedStats: StatGroupConfig[] = useMemo(() => {
    if (!summary) return [];

    return [
      {
        title: 'Tổng quan',
        stats: [
          {
            title: 'Đã thanh toán',
            value: summary.paidAmount ? formatCurrency(summary.paidAmount) : formatCurrency(0),
            icon: DollarSign,
          },
          {
            title: 'NV đã thanh toán',
            value: summary.paidEmployees ?? 0,
            icon: Users,
          },
        ],
      },
      {
        title: 'Cần xử lý',
        stats: [
          {
            title: 'Chờ duyệt',
            value: summary.pendingApproval,
            icon: AlertCircle,
          },
          {
            title: 'NV chờ thanh toán',
            value: summary.pendingEmployees ?? 0,
            icon: Clock,
          },
          {
            title: 'Chờ thanh toán',
            value: summary.pendingPaymentAmount ? formatCurrency(summary.pendingPaymentAmount) : formatCurrency(0),
            icon: Wallet,
          },
        ],
      },
      {
        title: 'Hoàn tất',
        stats: [
          {
            title: 'Đã duyệt',
            value: summary.approvedEntries,
            icon: CheckSquare,
          },
          {
            title: 'Đã thanh toán',
            value: summary.paidEntries,
            icon: DollarSign,
          },
          {
            title: 'Bị loại',
            value: summary.rejectedEntries,
            icon: XCircle,
          },
        ],
      },
    ];
  }, [summary]);

  return {
    statsConfig,
    groupedStats,
    isLoading,
    error: !!error,
    summary
  };
};
