import { Briefcase } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { useAuth } from '@/contexts';
import { useNavigate } from 'react-router-dom';
import { cn } from '@/lib/utils';
import type { AppRole } from '@/lib/auth';

// Custom icon component for PNG images
interface IconImageProps {
  src: string;
  alt: string;
  className?: string;
}

const IconImage = ({ src, alt, className }: IconImageProps) => (
  <img
    src={src}
    alt={alt}
    className={cn("h-4 w-4", className)}
  />
);

interface FixedAction {
  id: string;
  label: string;
  icon: typeof Briefcase | React.ComponentType<{ className?: string }>;
  href?: string;
  onClick?: () => void;
  permissions: AppRole[];
}

interface QuickActionsProps {
  className?: string;
  maxActions?: number;
}

export const QuickActions = ({ className }: QuickActionsProps) => {
  const { user } = useAuth();
  const navigate = useNavigate();

  // Fixed list of quick actions - ALL actions available for both roles
  const fixedActions: FixedAction[] = [
    {
      id: 'create-project',
      label: 'Tạo dự án',
      icon: () => <IconImage src="/icons/add-project.png" alt="Add project" />,
      href: '/admin/projects?modal=project_create',
      permissions: ['admin', 'quanly'],
    },
    {
      id: 'create-employee',
      label: 'Thêm nhân viên',
      icon: () => <IconImage src="/icons/add-worker.png" alt="Add employee" />,
      href: '/admin/employees?modal=add_employee',
      permissions: ['admin', 'quanly'],
    },
    {
      id: 'create-timesheet',
      label: 'Tạo chấm công',
      icon: () => <IconImage src="/icons/add-event.png" alt="Add timesheet" />,
      href: '/timesheet?modal=timesheet_entry',
      permissions: ['admin', 'quanly'],
    },
  ];

  const handleActionClick = (action: FixedAction) => {
    // Handle onClick actions first
    if (action.onClick) {
      action.onClick();
      return;
    }

    // Handle navigation actions
    if (action.href) {
      let targetHref = action.href;

      if (user.role === 'quanly') {
        // Update URLs for partner role with same modals
        if (action.id === 'create-project') {
          targetHref = '/partner/projects?modal=project_create';
        } else if (action.id === 'create-employee') {
          targetHref = '/partner/employees?modal=add_employee';
        }
        // create-timesheet needs no role branch: /timesheet is role-shared.
      }

      navigate(targetHref);
    }
  };

  if (!user) {
    return null;
  }

  // Filter actions based on user permissions
  const allowedActions = fixedActions.filter(action =>
    action.permissions.includes(user.role)
  );

  return (
    <TooltipProvider>
      <div className={cn("flex items-center gap-1", className)}>
        {allowedActions.map((action) => {
          const IconComponent = action.icon;

          return (
            <Tooltip key={action.id}>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="sm"
                  className="relative h-9 w-9 p-0 hover:bg-accent hover:text-accent-foreground touch-manipulation"
                  onClick={() => handleActionClick(action)}
                  aria-label={action.label}
                >
                  <IconComponent className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                <p className="font-medium">{action.label}</p>
              </TooltipContent>
            </Tooltip>
          );
        })}
      </div>
    </TooltipProvider>
  );
};
