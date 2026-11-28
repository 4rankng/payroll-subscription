import { Outlet } from "react-router-dom";
import { useLayoutEffect } from "react";
import AdminSidebar from "@/components/AdminSidebar";
import ProtectedRoute from "@/components/ProtectedRoute";
import { SidebarProvider } from "@/components/ui/sidebar";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { SectionErrorBoundary } from "@/components/ErrorBoundary";
import { useAuth } from "@/contexts";
import { useIsMobile, useIsTablet } from "@/hooks/useBreakpoint";
import {
  Briefcase01,
  Clipboard,
  Clock,
  Home01,
  Settings01,
  UserEdit,
  Users01,
} from "@untitledui/icons";
import type { NavGroup, NavLeaf } from "@/components/MobileBottomNav";

const ADMIN_NAV_GROUPS: NavGroup[] = [
  { title: "Tổng quan", icon: Home01, path: "/admin", end: true },
  { title: "Lương tuần", icon: Clock, path: "/timesheet" },
];

export const ADMIN_MORE_ITEMS: NavLeaf[] = [
  { title: "Nhật ký", icon: Clipboard, path: "/admin/audit-log" },
  { title: "Người dùng", icon: UserEdit, path: "/admin/users" },
  { title: "Dự án", icon: Briefcase01, path: "/admin/projects" },
  { title: "Nhân viên", icon: Users01, path: "/admin/employees" },
  { title: "Cài đặt", icon: Settings01, path: "/admin/settings" },
];

const AdminLayoutInner = () => {
  const isMobile = useIsMobile();

  return (
    <div
      className={
        isMobile
          ? "admin-shell relative flex min-h-dvh w-full group/layout"
          : "admin-shell relative flex h-dvh w-full group/layout"
      }
    >
      <AdminSidebar />
      <div
        className={
          isMobile
            ? "flex min-w-0 flex-1 flex-col"
            : "flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
        }
      >
        <main
          id="main-content"
          className={
            isMobile
              ? "admin-main console-shell min-h-dvh flex-1 overflow-visible"
              : "admin-main console-shell min-h-0 flex-1 overflow-auto"
          }
        >
          <div className="admin-shell-rail" aria-hidden="true" />
          <div className="admin-page-frame mobile-main-content animate-page-enter">
            <div className="admin-page-inner">
              <SectionErrorBoundary sectionName="trang quản trị">
                <Outlet />
              </SectionErrorBoundary>
            </div>
          </div>
        </main>
      </div>
    </div>
  );
};

const AdminLayout = () => {
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';
  // Tablet (768–1023px): start with the sidebar collapsed to the icon rail so
  // data tables get the full remaining width. Desktop keeps it expanded.
  const isTablet = useIsTablet();

  useLayoutEffect(() => {
    if (!isAdmin) return;
    const root = document.documentElement;
    // One theme for the whole admin console. The toto light day-console tokens
    // are promoted from the dashboard pilot to every admin route so navigating
    // between dashboard, users, projects and settings never repaints the shell.
    root.classList.add("admin-route-active", "admin-console-toto");
    return () => {
      root.classList.remove("admin-route-active", "admin-console-toto");
    };
  }, [isAdmin]);

  return (
    <ProtectedRoute requiredRole={["admin"]}>
      <SidebarProvider defaultOpen={!isTablet}>
        <div
          data-admin-ui={isAdmin ? "" : undefined}
          data-theme={isAdmin ? "congtruong" : undefined}
          className={isAdmin ? "admin-shell-scope" : "w-full min-w-0"}
        >
          <div className="w-full min-w-0 [container-type:inline-size]">
            <AdminLayoutInner />
          </div>
          <MobileBottomNav
            groups={ADMIN_NAV_GROUPS}
            moreItems={ADMIN_MORE_ITEMS}
          />
        </div>
      </SidebarProvider>
    </ProtectedRoute>
  );
};

export default AdminLayout;
