/**
 * Partner-portal navigation model (PRD PayrollSubscriptionPRD §4 ma trận
 * Trang × Vai trò). One source of truth for the desktop sidebar and the
 * mobile bottom nav: each item carries the roles allowed to see it.
 */
import type { ComponentType } from "react";
import {
  BookOpen01,
  Briefcase01,
  Calendar,
  FileCheck01,
  Grid01,
  Users01,
} from "@untitledui/icons";
import type { AppRole } from "@/lib/auth";

export interface PartnerNavItem {
  title: string;
  icon: ComponentType<{ className?: string }>;
  path: string;
  end?: boolean;
  roles: AppRole[];
}

const ALL_STAFF: AppRole[] = ["quanly", "ketoan", "giamdoc"];

export const PARTNER_NAV_ITEMS: PartnerNavItem[] = [
  // Tổng quan is role-split per PRD §5.1/§5.2: quanly keeps the partner
  // mirror dashboard; ketoan/giamdoc land on the fee-insights overview.
  { title: "Tổng quan", icon: Grid01, path: "/partner/dashboard", end: true, roles: ["quanly"] },
  { title: "Tổng quan", icon: Grid01, path: "/partner/overview", end: true, roles: ["ketoan", "giamdoc"] },
  { title: "Người dùng", icon: Users01, path: "/partner/users", end: true, roles: ["giamdoc"] },
  { title: "Dự án", icon: Briefcase01, path: "/partner/projects", roles: ALL_STAFF },
  { title: "Nhân viên", icon: Users01, path: "/partner/employees", roles: ALL_STAFF },
  { title: "Bảng công", icon: Calendar, path: "/partner/timesheet", end: true, roles: ALL_STAFF },
  // PAYR-42: rebuilt payroll payment history for all staff roles.
  { title: "Lịch sử trả lương", icon: FileCheck01, path: "/partner/payment-history", roles: ALL_STAFF },
  // PAYR-41: company transfer ledger + sao kê export/email.
  { title: "Sổ cái", icon: BookOpen01, path: "/partner/ledger", end: true, roles: ["ketoan", "giamdoc"] },
];

export const partnerNavItems = (role?: string): PartnerNavItem[] =>
  PARTNER_NAV_ITEMS.filter((item) => role !== undefined && item.roles.includes(role as AppRole));
