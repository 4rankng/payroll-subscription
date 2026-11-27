import { authManager } from '@/lib/auth';
import { Timesheet } from '@/types/api/timesheet.types';

/**
 * Permission utilities
 * Handles business rules for role-based actions
 */

export type UserRole = 'admin' | 'quanly' | 'ketoan' | 'nhanvien' | 'giamdoc';

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
