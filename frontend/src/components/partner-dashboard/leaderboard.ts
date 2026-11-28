/**
 * Shared partner-dashboard presentation helpers: one podium palette for the
 * Top-nhân-viên leaderboard (desktop LeaderRow + mobile TopEmployeeRow) and
 * the role eyebrow map for page headers.
 */

/** Podium badge tokens: #1 gold, #2 silver, #3 bronze, #4+ neutral (DASH-07). */
export const rankBadgeClass = (rank: number): string => {
  if (rank === 1) return 'bg-amber-100 text-amber-700 ring-1 ring-amber-300';
  if (rank === 2) return 'bg-slate-100 text-slate-600 ring-1 ring-slate-300';
  if (rank === 3) return 'bg-orange-100 text-orange-700 ring-1 ring-orange-300';
  return 'bg-muted text-muted-foreground ring-1 ring-border/60';
};

/** Uppercase role eyebrow for partner page headers (console-kicker renders). */
export const ROLE_EYEBROW: Record<string, string> = {
  quanly: 'QUẢN LÝ',
  ketoan: 'KẾ TOÁN',
  giamdoc: 'GIÁM ĐỐC',
};
