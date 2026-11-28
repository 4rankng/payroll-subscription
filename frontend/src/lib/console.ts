/**
 * Console kickers — the mono domain label shown in every staff page header
 * (toto.sg mission-control port). Pure path→domain mapping; no router
 * dependency so it is testable and usable from any layout.
 */

export type ConsoleKicker = '' | 'NHÂN SỰ' | 'TÀI CHÍNH';

// Owner 28/11: no default eyebrow — unmatched routes render no kicker text.
const DEFAULT_KICKER: ConsoleKicker = '';

/** Longest-prefix routes first is not required — prefixes are disjoint, but
 *  matching is segment-exact (`/admin/users` never captures
 *  `/admin/users-archive`). PAYR-126: dead prefixes removed; entries cover
 *  mounted routes only. */
const ROUTE_KICKERS: ReadonlyArray<{ prefix: string; kicker: ConsoleKicker }> = [
  // NHÂN SỰ — people records and accounts.
  { prefix: '/admin/users', kicker: 'NHÂN SỰ' },
  { prefix: '/admin/employees', kicker: 'NHÂN SỰ' },
  { prefix: '/partner/employees', kicker: 'NHÂN SỰ' },

  // TÀI CHÍNH — money movement.
  { prefix: '/payment-history', kicker: 'TÀI CHÍNH' },
  { prefix: '/partner/ledger', kicker: 'TÀI CHÍNH' },
];

export const getConsoleKicker = (pathname: string): ConsoleKicker => {
  const path = pathname.split('?')[0].replace(/\/+$/, '') || '/';
  const match = ROUTE_KICKERS.find(
    (route) => path === route.prefix || path.startsWith(`${route.prefix}/`),
  );
  return match?.kicker ?? DEFAULT_KICKER;
};
