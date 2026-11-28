// Common formatting utilities used across all pages

import { VN_TIMEZONE } from '@/utils/vn-time';

const DEFAULT_CURRENCY_CODE = 'VND';

export interface CurrencyDisplayParts {
  amount: string;
  unit?: string;
  isCurrencyUnit: boolean;
}

/**
 * Splits a trailing VND symbol from an already localized display string so
 * money KPIs can render the number and its currency unit with distinct type.
 */
export const splitCurrencyDisplay = (value: string, explicitUnit?: string): CurrencyDisplayParts => {
  if (explicitUnit) {
    const normalizedUnit = explicitUnit.trim();
    const isCurrencyUnit = /^(đ|₫|VNĐ)$/u.test(normalizedUnit);
    return {
      amount: value,
      unit: isCurrencyUnit ? '₫' : normalizedUnit,
      isCurrencyUnit,
    };
  }

  const match = value.match(/^(.*?)[\s\u00A0]*(đ|₫|VNĐ)$/u);
  if (!match) {
    return { amount: value, isCurrencyUnit: false };
  }

  return { amount: match[1].trimEnd(), unit: '₫', isCurrencyUnit: true };
};

/**
 * Normalize currency codes so that UI helpers never pass invalid codes
 * to Intl.NumberFormat. Currently we only support VND, but we gracefully
 * map common Vietnamese currency symbols to the ISO code.
 */
export const normalizeCurrencyCode = (currency?: string | null | number): string => {
  if (!currency || typeof currency !== 'string') return DEFAULT_CURRENCY_CODE;

  const trimmed = currency.trim();
  if (!trimmed) return DEFAULT_CURRENCY_CODE;

  const upperCased = trimmed.toUpperCase();
  if (trimmed === 'đ' || trimmed === '₫' || upperCased === 'VNĐ') {
    return DEFAULT_CURRENCY_CODE;
  }

  return upperCased.length === 3 ? upperCased : DEFAULT_CURRENCY_CODE;
};

export const formatCurrency = (amount: number | null | undefined, currency: string = DEFAULT_CURRENCY_CODE) => {
  if (amount === null || amount === undefined || isNaN(amount)) {
    return '- ₫';
  }
  const normalizedCurrency = normalizeCurrencyCode(currency);
  return new Intl.NumberFormat('vi-VN', {
    style: 'currency',
    currency: normalizedCurrency
  }).format(amount);
};

export const formatDate = (date: string | Date, format: 'short' | 'long' = 'short') => {
  const dateObj = typeof date === 'string' ? new Date(date) : date;

  if (format === 'long') {
    return new Intl.DateTimeFormat('vi-VN', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    }).format(dateObj);
  }

  return new Intl.DateTimeFormat('vi-VN').format(dateObj);
};

/**
 * Format date with time: dd/MM/yyyy HH:mm
 */
export const formatDateTime = (date: string | Date): string => {
  const dateObj = typeof date === 'string' ? new Date(date) : date;
  return new Intl.DateTimeFormat('vi-VN', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(dateObj);
};

/**
 * Format date with full time including seconds: dd/MM/yyyy HH:mm:ss
 */
export const formatDateTimeFull = (date: string | Date): string => {
  const dateObj = typeof date === 'string' ? new Date(date) : date;
  return new Intl.DateTimeFormat('vi-VN', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(dateObj);
};

export const formatDateForAPI = (date: string | Date): string => {
  const dateObj = typeof date === 'string' ? new Date(date) : date;
  // Format date without timezone conversion - use local date values directly
  const year = dateObj.getFullYear();
  const month = String(dateObj.getMonth() + 1).padStart(2, '0');
  const day = String(dateObj.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

export const formatPercentage = (value: number, decimals = 1) => {
  return `${value.toFixed(decimals)}%`;
};

export const formatFileSize = (bytes: number) => {
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  if (bytes === 0) return '0 Bytes';
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i];
};

export const formatDuration = (minutes: number) => {
  const hours = Math.floor(minutes / 60);
  const mins = minutes % 60;

  if (hours === 0) return `${mins}m`;
  if (mins === 0) return `${hours}h`;
  return `${hours}h ${mins}m`;
};

export const getInitials = (name: string) => {
  return name
    .split(' ')
    .slice(-2)
    .map(n => n[0])
    .join('')
    .toUpperCase();
};

export const truncateText = (text: string, maxLength: number) => {
  if (text.length <= maxLength) return text;
  return text.substring(0, maxLength) + '...';
};

/**
 * Format a currency amount given as a string (e.g. from API responses).
 * Parses the string to a number and delegates to formatCurrency.
 */
export const formatCurrencyFromString = (amountStr?: string | null, currency?: string): string => {
  if (!amountStr) return '- ₫';
  const num = Number(amountStr.replace(/,/g, ''));
  return formatCurrency(num, currency);
};

// ─── Canonical Vietnamese display helpers ───────────────────────────────────
// Timezone-pinned variants below render in Asia/Ho_Chi_Minh so output never
// depends on the host device timezone. The legacy formatDate/formatDateTime
// above keep their existing host-timezone behavior for existing callers.

const vnDateFormatter = new Intl.DateTimeFormat('en-GB', {
  timeZone: VN_TIMEZONE,
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
});

const vnHourMinuteFormatter = new Intl.DateTimeFormat('en-GB', {
  timeZone: VN_TIMEZONE,
  hour: '2-digit',
  minute: '2-digit',
  hourCycle: 'h23',
});

/** Canonical integer with vi-VN grouping: 1747 → "1.747". */
export const formatInt = (value: number | null | undefined): string => {
  if (value === null || value === undefined || Number.isNaN(value)) return '0';
  return new Intl.NumberFormat('vi-VN', { maximumFractionDigits: 0 }).format(value);
};

/** Canonical hours with vi-VN decimal comma: 8.6 → "8,60 giờ". */
export const formatHours = (value: number | null | undefined): string => {
  if (value === null || value === undefined || Number.isNaN(value)) return '0,00 giờ';
  return `${new Intl.NumberFormat('vi-VN', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value)} giờ`;
};

/** Canonical calendar date in Asia/Ho_Chi_Minh: → "DD/MM/YYYY". Invalid input → "—". */
export const formatDateVn = (date: string | Date | null | undefined): string => {
  const dateObj = date instanceof Date ? date : date ? new Date(date) : null;
  if (!dateObj || Number.isNaN(dateObj.getTime())) return '—';
  return vnDateFormatter.format(dateObj);
};

/** Canonical transfer timestamp in Asia/Ho_Chi_Minh: → "DD/MM/YYYY HH:mm (+07)". Invalid input → "—". */
export const formatDateTimeVn = (date: string | Date | null | undefined): string => {
  const dateObj = date instanceof Date ? date : date ? new Date(date) : null;
  if (!dateObj || Number.isNaN(dateObj.getTime())) return '—';
  const time = vnHourMinuteFormatter.format(dateObj);
  // "2-digit" h23 can render "24:xx" near midnight in some engines.
  const normalized = time.startsWith('24') ? `00${time.slice(2)}` : time;
  return `${vnDateFormatter.format(dateObj)} ${normalized} (+07)`;
};

/** Compact VND for tight KPI contexts: 105_000_000 → "105 tr". */
export const formatCompactVND = (value: number | null | undefined): string => {
  if (value === null || value === undefined || Number.isNaN(value)) return '0 tr';
  const millions = new Intl.NumberFormat('vi-VN', { maximumFractionDigits: 1 }).format(
    value / 1_000_000,
  );
  return `${millions} tr`;
};

export const formatNumber = (value: number, decimals = 1) => {
  return new Intl.NumberFormat('vi-VN', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals
  }).format(value);
};

/**
 * Full currency formatter for chart axes, tooltips, legends, and small card contexts.
 * Monetary values must always keep their full digits instead of K/M/B abbreviations.
 */
export const formatFullCurrency = (
  value: number,
  options: { showSymbol?: boolean } = {}
): string => {
  const { showSymbol = true } = options;
  const formattedValue = new Intl.NumberFormat('vi-VN', {
    maximumFractionDigits: 0,
  }).format(value);

  return `${formattedValue}${showSymbol ? ' ₫' : ''}`;
};


