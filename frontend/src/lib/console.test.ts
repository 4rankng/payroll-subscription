import { describe, expect, it } from 'vitest';

import { getConsoleKicker } from './console';

describe('getConsoleKicker', () => {
  it('returns no kicker for staff dashboards and ops routes (owner 28/11)', () => {
    expect(getConsoleKicker('/admin')).toBe('');
    expect(getConsoleKicker('/admin/projects')).toBe('');
    expect(getConsoleKicker('/timesheet')).toBe('');
    expect(getConsoleKicker('/admin/settings')).toBe('');
    expect(getConsoleKicker('/admin/audit-log')).toBe('');
    expect(getConsoleKicker('/admin/email')).toBe('');
    expect(getConsoleKicker('/dashboard')).toBe('');
    expect(getConsoleKicker('/partner/overview')).toBe('');
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
    expect(getConsoleKicker('/payment-history')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/payment-history?ky=3')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/partner/ledger')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/partner/ledger/')).toBe('TÀI CHÍNH');
  });

  it('does not let a prefix capture a longer sibling segment', () => {
    expect(getConsoleKicker('/admin/users-archive')).toBe('');
    expect(getConsoleKicker('/admin/employeesx')).toBe('');
    expect(getConsoleKicker('/payment-history-x')).toBe('');
  });

  it('falls back to no kicker for non-staff and unknown paths', () => {
    expect(getConsoleKicker('/login')).toBe('');
    expect(getConsoleKicker('/')).toBe('');
    expect(getConsoleKicker('/definitely-not-a-route')).toBe('');
  });

  it('ignores query strings and trailing slashes', () => {
    expect(getConsoleKicker('/partner/ledger?tab=pending')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/admin/users/')).toBe('NHÂN SỰ');
  });
});
