import { describe, expect, it } from 'vitest';

import { getConsoleKicker } from './console';

describe('getConsoleKicker', () => {
  it('returns no kicker for staff dashboards and ops routes (owner 28/11)', () => {
    expect(getConsoleKicker('/admin/dashboard')).toBe('');
    expect(getConsoleKicker('/admin/projects')).toBe('');
    expect(getConsoleKicker('/admin/timesheet')).toBe('');
    expect(getConsoleKicker('/admin/approvals')).toBe('');
    expect(getConsoleKicker('/admin/settings')).toBe('');
    expect(getConsoleKicker('/admin/system-health')).toBe('');
    expect(getConsoleKicker('/admin/audit-log')).toBe('');
    expect(getConsoleKicker('/admin/zns-tracking')).toBe('');
    expect(getConsoleKicker('/partner/dashboard')).toBe('');
    expect(getConsoleKicker('/partner/timesheet')).toBe('');
    expect(getConsoleKicker('/')).toBe('');
  });

  it('maps people routes to NHÂN SỰ', () => {
    expect(getConsoleKicker('/admin/users')).toBe('NHÂN SỰ');
    expect(getConsoleKicker('/admin/users/42')).toBe('NHÂN SỰ');
    expect(getConsoleKicker('/admin/employees')).toBe('NHÂN SỰ');
    expect(getConsoleKicker('/admin/employees/')).toBe('NHÂN SỰ');
    expect(getConsoleKicker('/partner/employees')).toBe('NHÂN SỰ');
  });

  it('maps money routes to TÀI CHÍNH', () => {
    expect(getConsoleKicker('/admin/ledger')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/payment-history')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/payment-history?ky=3')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/admin/loans')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/admin/loans/lenders')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/admin/advance-payments/check-in-settings')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/admin/wallet')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/adv-partner/advance-payments')).toBe('TÀI CHÍNH');
  });

  it('maps the accountant workspace to KẾ TOÁN', () => {
    expect(getConsoleKicker('/accountant')).toBe('KẾ TOÁN');
  });

  it('does not let a prefix capture a longer sibling segment', () => {
    expect(getConsoleKicker('/admin/users-archive')).toBe('');
    expect(getConsoleKicker('/admin/employeesx')).toBe('');
  });

  it('falls back to no kicker for non-staff and unknown paths', () => {
    expect(getConsoleKicker('/login')).toBe('');
    expect(getConsoleKicker('/')).toBe('');
    expect(getConsoleKicker('/definitely-not-a-route')).toBe('');
  });

  it('ignores query strings and trailing slashes', () => {
    expect(getConsoleKicker('/admin/ledger?tab=pending')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/accountant/')).toBe('KẾ TOÁN');
  });
});
