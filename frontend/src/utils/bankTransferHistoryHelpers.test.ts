import { describe, expect, it } from 'vitest';

import { getCurrentMonthValue } from './bankTransferHistoryHelpers';

describe('getCurrentMonthValue', () => {
  it('formats the current month as YYYY-MM with a zero-padded month', () => {
    const now = new Date();
    const expected = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
    expect(getCurrentMonthValue()).toBe(expected);
  });
});
