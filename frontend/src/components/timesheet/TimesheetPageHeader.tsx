import { PageHeader } from '@/components/shared/PageHeader';
import { Plus } from 'lucide-react';

interface TimesheetPageHeaderProps {
  onAddTimesheet?: () => void;
  userRole?: 'admin' | 'quanly';
}

export const TimesheetPageHeader = ({ onAddTimesheet, userRole = 'admin' }: TimesheetPageHeaderProps) => {
  const description = userRole === 'quanly'
    ? 'Theo dõi và quản lý bảng công'
    : 'Theo dõi và duyệt bảng công';

  return (
    <PageHeader
      title="Bảng công"
      description={description}
      actions={onAddTimesheet ? [{ label: 'Nhập công', onClick: onAddTimesheet, icon: Plus }] : []}
    />
  );
};
