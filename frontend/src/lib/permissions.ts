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
 * Check if user can approve/reject timesheets — single entries, the entry
 * modal's "Duyệt & Lưu", and the pending-selection bulk actions. Admin,
 * quanly and ketoan hold this right; the global/destructive ops
 * (approve-all, reset-all, bulk-reset, reject-unpaid, add-to-payroll) stay
 * admin-only via canAdminTimesheetOps and are re-enforced by casbin.
 */
export const canApproveTimesheet = (userRole?: UserRole | null): boolean => {
  const role = userRole || authManager.getUserRole();

  return role === 'admin' || role === 'quanly' || role === 'ketoan';
};

/**
 * Check if user can run global/destructive timesheet ops (approve-all,
 * reset-all, bulk-reset, reject-unpaid, add-to-payroll). Admin-only.
 */
export const canAdminTimesheetOps = (userRole?: UserRole | null): boolean => {
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
 * Check if user can manage other users (create, edit, reset password,
 * delete). Only admin holds user-management writes: giamdoc reads
 * /partner/users read-only.
 */
export const canManageUsers = (userRole?: UserRole | null): boolean => {
  const role = userRole || authManager.getUserRole();

  return role === 'admin';
};
