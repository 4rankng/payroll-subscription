import { useEffect } from "react";
import { Outlet } from "react-router-dom";
import PartnerSidebar from "@/components/PartnerSidebar";
import ProtectedRoute from "@/components/ProtectedRoute";
import { SidebarProvider } from "@/components/ui/sidebar";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { partnerNavItems } from "@/config/partnerNav";
import { SectionErrorBoundary } from "@/components/ErrorBoundary";
import { Briefcase01, Calendar, Grid01, Users01, Wallet01 } from "@untitledui/icons";
import type { NavGroup } from "@/components/MobileBottomNav";
import { useAuth } from "@/contexts";
import type { AppRole } from "@/lib/auth";

const PartnerLayoutInner = () => {
  useEffect(() => {
    // One console theme for every role this layout serves — quanly, ketoan,
    // giamdoc and admin on /payment-history: the same token blocks, structural
    // rules and static-ink re-skin layers key off html.admin-console-toto
    // (variables.css maps the partner alias at the --admin-* tokens).
    // partner-route-active keeps the role scope for partner-specific styling.
    document.documentElement.classList.add("partner-route-active", "admin-console-toto");
    return () => document.documentElement.classList.remove("partner-route-active", "admin-console-toto");
  }, []);

  return (
    <div
      data-partner-ui=""
      data-theme="congtruong"
      className="partner-shell-scope relative flex h-dvh w-full group/layout"
    >
      <PartnerSidebar />
      <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
        <main
          id="main-content"
          className="console-shell min-h-0 flex-1 overflow-auto"
        >
          <div className="min-h-full mobile-main-content animate-page-enter max-w-[1480px] mx-auto">
            <SectionErrorBoundary sectionName="trang quản lý">
              <Outlet />
            </SectionErrorBoundary>
          </div>
        </main>
      </div>
    </div>
  );
};

const PartnerLayout = ({ roles = ["quanly", "ketoan", "giamdoc"] }: { roles?: AppRole[] }) => {
  const { user } = useAuth();
  const navGroups = partnerNavItems(user?.role);

  return (
    // Mirrors the /partner route guard: quanly operates; ketoan reads;
    // giamdoc reads, approves and exports (PRD §4 ma trận Trang × Vai trò).
    // /payment-history reuses the layout with admin added.
    <ProtectedRoute requiredRole={roles}>
      <SidebarProvider defaultOpen={true}>
        <PartnerLayoutInner />
        <MobileBottomNav groups={navGroups} />
      </SidebarProvider>
    </ProtectedRoute>
  );
};

export default PartnerLayout;
