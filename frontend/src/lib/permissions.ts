import { authManager } from '@/lib/auth';
import { Timesheet } from '@/types/api/timesheet.types';

/**
 * Permission utilities
 * Handles business rules for role-based actions
 */

export type UserRole = 'admin' | 'quanly' | 'ketoan' | 'nhanvien' | 'giamdoc';

/** Vietnamese display name per role — single source for profile menus. */
export const roleLabel = (userRole?: UserRole | string | null): string => {
  switch (userRole) {
    case 'admin':
      return 'Quản trị viên';
    case 'quanly':
      return 'Quản lý';
    case 'ketoan':
      return 'Kế toán';
    case 'giamdoc':
      return 'Giám đốc';
    default:
      return 'Nhân viên';
  }
};

/**
 * Check if user can edit a timesheet entry
 * Rules (approved timesheets are immutable — reset them first):
 * - Paid timesheets are never editable
 * - Approved timesheets are never editable
 * - Admin can edit pending/rejected timesheets that are not paid
 * - Partner can edit pending/rejected timesheets that are not paid
 */
export const canEditTimesheet = (
  timesheet: Timesheet | { status: string; payment_status?: string },
  userRole?: UserRole | null
): boolean => {
  const role = userRole || authManager.getUserRole();

  if (role !== 'admin' && role !== 'quanly') return false;

  // Cannot edit if already paid
  const paymentStatus = 'payment_status' in timesheet ? timesheet.payment_status : undefined;
  if (paymentStatus === 'paid') return false;

  // Approved timesheets are immutable
  if (timesheet.status === 'approved') return false;

  return true;
};

/**
 * Check if user can manage projects
 * Both Admin and Partner have full project management permissions
 */
export const canPartnerManageProject = (userRole?: UserRole | null): boolean => {
  const role = userRole || authManager.getUserRole();

  if (!role) return false;

  return role === 'admin' || role === 'quanly';
};

/**
 * Check if user can approve/reject timesheets
 * Only Admins can approve/reject timesheets
 */
export const canApproveTimesheet = (userRole?: UserRole | null): boolean => {
  const role = userRole || authManager.getUserRole();

  return role === 'admin';
};

/**
 * Check if user can delete timesheets
 * Rules:
 * - Admin can always delete
 * - Partner can delete if not approved or rejected
 */
export const canDeleteTimesheet = (
  timesheet: Timesheet | { status: string },
  userRole?: UserRole | null
): boolean => {
  const role = userRole || authManager.getUserRole();

  if (!role) return false;

  // Admin has full access
  if (role === 'admin') return true;

  // Partner cannot delete approved or rejected timesheets
  if (role === 'quanly') {
    return timesheet.status !== 'approved' && timesheet.status !== 'rejected';
  }

  return false;
};

/**
 * Check if user can view partner employees (list and detail).
 * ketoan/giamdoc hold read-only access; nhanvien has none.
 */
export const canViewEmployees = (userRole?: UserRole | null): boolean => {
  const role = userRole || authManager.getUserRole();

  return role === 'admin' || role === 'quanly' || role === 'ketoan' || role === 'giamdoc';
};

/**
 * Check if user can claim employee management (yêu cầu quản lý).
 * Backend grants request-access to admin and quanly only.
 */
export const canClaimEmployee = (userRole?: UserRole | null): boolean => {
  const role = userRole || authManager.getUserRole();

  return role === 'admin' || role === 'quanly';
};

/**
 * Check if user can create users.
 * Backend grants POST /users to admin and giamdoc (PRD §5.3 full-control
 * user management); quanly/ketoan hold read-only user access.
 */
export const canCreateUser = (userRole?: UserRole | null): boolean => {
  const role = userRole || authManager.getUserRole();

  return role === 'admin' || role === 'giamdoc';
};
