import { describe, expect, it } from 'vitest';

import {
  normalizeVietnamese,
  stripVietnameseDiacritics,
  vietnameseEquals,
  vietnameseIncludes,
} from './vietnameseNormalization';

// ASCII keywords must match Vietnamese text — search boxes fold
// diacritics (and Đ→D) on both sides before comparing.
describe('stripVietnameseDiacritics', () => {
  it('strips tone and vowel marks', () => {
    expect(stripVietnameseDiacritics('Nguyễn Văn Hùng')).toBe('Nguyen Van Hung');
    expect(stripVietnameseDiacritics('Lê Thị Thu Hà')).toBe('Le Thi Thu Ha');
    expect(stripVietnameseDiacritics('Phạm Đức Anh')).toBe('Pham Duc Anh');
  });

  it('maps Đ/đ to D/d (NFD does not decompose them)', () => {
    expect(stripVietnameseDiacritics('Đặng Văn Đ')).toBe('Dang Van D');
    expect(stripVietnameseDiacritics('Đồng phí')).toBe('Dong phi');
  });

  it('leaves ASCII and digits untouched', () => {
    expect(stripVietnameseDiacritics('FT26314151203706')).toBe('FT26314151203706');
    expect(stripVietnameseDiacritics('')).toBe('');
  });
});

describe('vietnameseIncludes', () => {
  it('matches ASCII keywords against Vietnamese text', () => {
    expect(vietnameseIncludes('Nguyễn Văn Cường', 'nguyen van')).toBe(true);
    expect(vietnameseIncludes('Công ty TNHH Đông Á', 'dong')).toBe(true);
    expect(vietnameseIncludes('Trần Thị Mỹ Duyên', 'duyen')).toBe(true);
  });

  it('matches Vietnamese keywords against ASCII-folded text', () => {
    expect(vietnameseIncludes('nguyen van cuong', 'Nguyên Văn')).toBe(true);
  });

  it('does not match unrelated terms', () => {
    expect(vietnameseIncludes('Nguyễn Văn Cường', 'tran')).toBe(false);
  });

  it('handles NFD (decomposed) input from macOS files', () => {
    const nfd = 'Mai Ngọc Trung'.normalize('NFD');
    expect(vietnameseIncludes(nfd, 'ngoc trung')).toBe(true);
  });
});

describe('normalizeVietnamese / vietnameseEquals', () => {
  it('normalizes case and diacritics together', () => {
    expect(normalizeVietnamese('NGUYỄN THỊ HẠ')).toBe('nguyen thi ha');
  });

  it('compares equal across diacritics and case', () => {
    expect(vietnameseEquals('Đỗ Thị Hồng', 'do thi hong')).toBe(true);
    expect(vietnameseEquals('Ngô', 'Trần')).toBe(false);
  });
});
