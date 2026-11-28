import { describe, expect, it } from 'vitest';

import { getConsoleKicker } from './console';

describe('getConsoleKicker', () => {
  it('returns no kicker for staff dashboards and ops routes (owner 28/11)', () => {
    expect(getConsoleKicker('/projects')).toBe('');
    expect(getConsoleKicker('/projects/42/payrates/9/edit')).toBe('');
    expect(getConsoleKicker('/timesheet')).toBe('');
    expect(getConsoleKicker('/settings')).toBe('');
    expect(getConsoleKicker('/audit-log')).toBe('');
    expect(getConsoleKicker('/email')).toBe('');
    expect(getConsoleKicker('/dashboard')).toBe('');
    expect(getConsoleKicker('/overview')).toBe('');
    expect(getConsoleKicker('/')).toBe('');
  });

  it('maps people routes to NHÂN SỰ', () => {
    expect(getConsoleKicker('/users')).toBe('NHÂN SỰ');
    expect(getConsoleKicker('/users/42')).toBe('NHÂN SỰ');
    expect(getConsoleKicker('/employees')).toBe('NHÂN SỰ');
    expect(getConsoleKicker('/employees/')).toBe('NHÂN SỰ');
  });

  it('maps money routes to TÀI CHÍNH', () => {
    expect(getConsoleKicker('/payment-history')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/payment-history?ky=3')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/ledger')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/ledger/')).toBe('TÀI CHÍNH');
  });

  it('does not let a prefix capture a longer sibling segment', () => {
    expect(getConsoleKicker('/users-archive')).toBe('');
    expect(getConsoleKicker('/employeesx')).toBe('');
    expect(getConsoleKicker('/payment-history-x')).toBe('');
  });

  it('falls back to no kicker for non-staff and unknown paths', () => {
    expect(getConsoleKicker('/login')).toBe('');
    expect(getConsoleKicker('/')).toBe('');
    expect(getConsoleKicker('/definitely-not-a-route')).toBe('');
  });

  it('ignores query strings and trailing slashes', () => {
    expect(getConsoleKicker('/ledger?tab=pending')).toBe('TÀI CHÍNH');
    expect(getConsoleKicker('/users/')).toBe('NHÂN SỰ');
  });
});
