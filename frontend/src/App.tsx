import React, { Suspense, useEffect } from "react";
import { lazyWithReload as lazy } from "@/lib/lazy";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { AppProviders, useAuth } from "@/contexts";
import { QueryClient, QueryClientProvider, MutationCache, QueryCache } from "@tanstack/react-query";
import { BrowserRouter, Routes, Route, Navigate, useLocation } from "react-router-dom";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { ResponsivePage } from "@/components/ResponsivePage";
import Login from "./pages/Login";
import ForgotPassword from "./pages/ForgotPassword";
import ResetPassword from "./pages/ResetPassword";
import AdminLayout from "./layouts/AdminLayout";
import PartnerLayout from "./layouts/PartnerLayout";

// Admin Desktop Pages (lazy-loaded)
const UsersPage = lazy(() => import("./pages/admin/UsersPage"));
const ProjectsPage = lazy(() => import("./pages/admin/ProjectsPage"));
const EmployeesPage = lazy(() => import("./pages/admin/EmployeesPage"));
const SettingsPage = lazy(() => import("./pages/admin/SettingsPage"));
const EmailPage = lazy(() => import("./pages/admin/EmailPage"));
const PayrateEditPage = lazy(() => import("./pages/admin/PayrateEditPage"));
const AuditLogPage = lazy(() => import("./pages/admin/AuditLogPage"));

// Admin Mobile Pages (lazy-loaded)
const UsersPageMobile = lazy(() => import("./pages/mobile/admin/UsersPage"));
const ProjectsPageMobile = lazy(() => import("./pages/mobile/admin/ProjectsPage"));
const EmployeesPageMobile = lazy(() => import("./pages/mobile/admin/EmployeesPage"));
const SettingsPageMobile = lazy(() => import("./pages/mobile/admin/SettingsPage"));
const AuditLogPageMobile = lazy(() => import("./pages/mobile/admin/AuditLogPage"));
const PayrateEditPageMobile = lazy(() => import("./pages/mobile/admin/PayrateEditPage"));

// Partner Desktop Pages (lazy-loaded)
const PartnerProjectsPage = lazy(() => import("./pages/partner/ProjectsPage"));
const PartnerEmployeesPage = lazy(() => import("./pages/partner/EmployeesPage"));
const PartnerDashboardPage = lazy(() => import("./pages/partner/DashboardPage"));
const FeeInsightsPage = lazy(() => import("./pages/partner/OverviewPage"));
const FeeInsightsPageMobile = lazy(() =>
  import("./pages/partner/OverviewPage/Mobile").then((m) => ({ default: m.FeeInsightsPageMobile })),
);
const PartnerLedgerPage = lazy(() => import("./pages/partner/LedgerPage"));
const PartnerLedgerPageMobile = lazy(() => import("./pages/mobile/partner/LedgerPage"));
const PartnerPaymentHistoryPage = lazy(() => import("./pages/partner/PaymentHistoryPage"));
const PartnerPaymentHistoryPageMobile = lazy(() => import("./pages/mobile/partner/PaymentHistoryPage"));

// Partner Mobile Pages (lazy-loaded)
const PartnerProjectsPageMobile = lazy(() => import("./pages/mobile/partner/ProjectsPage"));
const PartnerEmployeesPageMobile = lazy(() => import("./pages/mobile/partner/EmployeesPage"));
const PartnerDashboardMobile = lazy(() => import("./pages/mobile/partner/DashboardPage"));

// Shared Timesheet Page — one canonical /timesheet for admin + partner
// roles; desktop and mobile render role-adaptively.
const TimesheetPage = lazy(() => import("./pages/shared/TimesheetPage"));
const TimesheetPageMobile = lazy(() => import("./pages/shared/TimesheetPage/Mobile"));

// Employee Pages (lazy-loaded)
const EmployeeRouter = lazy(() => import("./pages/employee/EmployeeRouter"));

import NotFound from "./pages/NotFound";
import ProtectedRoute from "./components/ProtectedRoute";
import { authManager } from "@/lib/auth";
import { useLogout } from "@/hooks/api/useAuth";
import { SecureModalProvider } from "@/components/modals/SecureModalProvider";
import ModalRouter from "@/components/modals/ModalRouter";
import { ForceChangePasswordDialog } from "@/components/ForceChangePasswordDialog";
import { showErrorNotification, isNetworkError, handleNetworkError } from "@/utils/error-handler";
import { initInvalidationService } from "@/lib/cache/invalidationService";
import { setupQueryPersistence } from "@/lib/cache/queryPersister";
import { isIncognitoMode } from "@/utils/dom-safety";
import { EnvironmentBanner } from "@/components/EnvironmentBanner";
import { developmentBannerText, shouldShowDevelopmentBanner } from "@/utils/environment";

// Minimal suspense fallback for lazy-loaded pages
const PageSuspenseFallback = React.memo(() => (
  <div className="flex h-[50vh] items-center justify-center">
    <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
  </div>
));

const LogoutRedirect = () => {
  // Revoke the server session too — clearing local state alone leaves the
  // cookie token valid until it expires.
  const { mutate: logout } = useLogout();
  useEffect(() => {
    logout();
  }, [logout]);
  return <Navigate to="/login" replace />;
};

// Component to handle root redirects safely
const RootRedirect = () => {
  const hasSession = authManager.hasSession();
  const userRole = hasSession ? authManager.getUserRole() : null;

  if (hasSession && (userRole === "admin" || userRole === "quanly" || userRole === "ketoan" || userRole === "giamdoc")) {
    return <Navigate to="/dashboard" replace />;
  }
  if (hasSession && userRole === "nhanvien") {
    return <Navigate to="/employee" replace />;
  }
  return <Navigate to="/login" replace />;
};

// One canonical /dashboard URL for every company role: the layout and the
// page split by role (PRD §5.1/§5.2), the URL stays the same.
const DashboardLayoutRouter = () => {
  const role = authManager.getUserRole();
  return role === "admin" ? <AdminLayout /> : <PartnerLayout />;
};

// PRD §5.3 (owner call): ONE canonical /users directory for admin + giamdoc —
// the split /admin/users + /partner/users routes are gone. Admin keeps the
// console shell; giamdoc keeps the partner shell. quanly/ketoan have no
// directory access (PRD §5.3).
const UsersShell = () => {
  const role = authManager.getUserRole();
  return role === "admin" ? <AdminLayout /> : <PartnerLayout roles={["giamdoc"]} />;
};

// PRD §5.7: admin crosses into the console shell on /payment-history — the
// partner nav has a single admin item, so the partner rail would render as a
// one-item strip and drop the admin chrome.
const PaymentHistoryShell = () => {
  const role = authManager.getUserRole();
  return role === "admin" ? <AdminLayout /> : <PartnerLayout roles={["quanly", "ketoan", "giamdoc"]} />;
};

// Canonical /projects + /employees + /ledger surfaces: one URL, the shell and
// the page follow the signed-in role — admin keeps the console chrome, partner
// roles keep the partner nav.
const ProjectsShell = () => {
  const role = authManager.getUserRole();
  return role === "admin" ? <AdminLayout /> : <PartnerLayout />;
};
const EmployeesShell = () => {
  const role = authManager.getUserRole();
  return role === "admin" ? <AdminLayout /> : <PartnerLayout />;
};
// PRD §5.8: Sổ cái — admin/ketoan/giamdoc only; PartnerLayout's roles prop
// blocks quanly, AdminLayout blocks everyone but admin.
const LedgerShell = () => {
  const role = authManager.getUserRole();
  return role === "admin" ? <AdminLayout /> : <PartnerLayout roles={["ketoan", "giamdoc"]} />;
};

const ProjectsPageRouter = () => {
  const role = authManager.getUserRole();
  return role === "admin" ? (
    <ResponsivePage desktopComponent={ProjectsPage} mobileComponent={ProjectsPageMobile} />
  ) : (
    <ResponsivePage desktopComponent={PartnerProjectsPage} mobileComponent={PartnerProjectsPageMobile} />
  );
};
const EmployeesPageRouter = () => {
  const role = authManager.getUserRole();
  return role === "admin" ? (
    <ResponsivePage desktopComponent={EmployeesPage} mobileComponent={EmployeesPageMobile} />
  ) : (
    <ResponsivePage desktopComponent={PartnerEmployeesPage} mobileComponent={PartnerEmployeesPageMobile} />
  );
};

const DashboardRoleRouter = () => {
  const role = authManager.getUserRole();
  if (role === "quanly") {
    return <ResponsivePage desktopComponent={PartnerDashboardPage} mobileComponent={PartnerDashboardMobile} />;
  }
  // admin + ketoan + giamdoc land on the fee-insights overview.
  return <ResponsivePage desktopComponent={FeeInsightsPage} mobileComponent={FeeInsightsPageMobile} />;
};

const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (error) => {
      // Handle network errors specifically
      if (isNetworkError(error)) {
        handleNetworkError();
        return;
      }

      // For other query errors, only show notification if it's a critical error
      // Most query errors are handled by components showing loading states
      console.error('Query Error:', error);
    },
  }),
  mutationCache: new MutationCache({
    onSuccess: (_data, _variables, _context, mutation) => {
      // Only invalidate related queries based on mutation metadata
      // This prevents unnecessary refetching of unrelated data
      if (mutation.meta?.invalidates) {
        const queryKeys = Array.isArray(mutation.meta.invalidates)
          ? mutation.meta.invalidates
          : [mutation.meta.invalidates];
        queryKeys.forEach((queryKey: unknown[]) => {
          queryClient.invalidateQueries({ queryKey });
        });
      }
    },
    onError: (error, _variables, _context, mutation) => {
      // Skip error notification if the mutation has its own onError handler
      // This prevents double notifications
      if (mutation.options.onError) {
        return;
      }

      // Skip error notification if the mutation explicitly opts out
      if (mutation.meta?.skipGlobalError) {
        return;
      }

      // Handle network errors specifically
      if (isNetworkError(error)) {
        handleNetworkError();
        return;
      }

      // Show error notification for all other mutation failures
      showErrorNotification(error);
    },
  }),
  defaultOptions: {
    queries: {
      staleTime: 30 * 1000, // 30 seconds
      gcTime: 30 * 60 * 1000, // 30 minutes garbage collection
      // Note: keepPreviousData is now placeholderData in TanStack Query v5
      placeholderData: (previousData: unknown) => previousData,
    },
  },
});

// Initialize the global invalidation service
initInvalidationService(queryClient);

// Persist query cache to sessionStorage (survives page refresh)
setupQueryPersistence(queryClient);

// Component that uses location hook inside Router context
/** Legacy /admin|/partner bookmarks → canonical URL, query kept. */
const LEGACY_PREFIX_TARGETS: Record<string, string> = {
  "/": "/dashboard",
  "/overview": "/dashboard",
  "/timesheets": "/timesheet",
};
const LegacyPrefixRedirect = () => {
  const { pathname, search } = useLocation();
  const rest = pathname.replace(/^\/(admin|partner)/, "") || "/";
  const target = LEGACY_PREFIX_TARGETS[rest] ?? rest;
  return <Navigate to={`${target}${search}`} replace />;
};

/** /timesheet shell follows the signed-in role — admin keeps the
 *  admin console nav (PartnerLayout would strand it: partnerNavItems('admin')
 *  grants no /admin links and skips admin theming); partner roles keep the
 *  partner nav. One URL, two shells. */
const TimesheetShell = () => {
  const { user } = useAuth();
  return user?.role === "admin" ? (
    <ProtectedRoute requiredRole="admin">
      <AdminLayout />
    </ProtectedRoute>
  ) : (
    <PartnerLayout />
  );
};

const AppContent = () => {
  const location = useLocation();

  const { headline: developmentHeadline, detail: developmentDetail } = developmentBannerText;

  // Add DOM error recovery for production
  React.useEffect(() => {
    const handleError = (event: ErrorEvent) => {
      if (event.error?.message?.includes('removeChild') ||
          event.error?.message?.includes('insertBefore') ||
          event.error?.message?.includes('appendChild')) {

        console.warn('DOM manipulation error detected, preventing crash:', event.error.message);
        event.preventDefault();

        // Log if in incognito mode for debugging
        isIncognitoMode().then(isIncognito => {
          if (!isIncognito) {
            console.info('DOM error occurred in normal browsing mode - likely browser extension interference');
          }
        });
      }
    };

    window.addEventListener('error', handleError);
    return () => window.removeEventListener('error', handleError);
  }, []);

  const developmentBannerElement = React.useMemo(() => {
    if (!shouldShowDevelopmentBanner) {
      return null;
    }

    return (
      <EnvironmentBanner
        headline={developmentHeadline}
        detail={developmentDetail}
      />
    );
  }, [developmentDetail, developmentHeadline]);

  return (
    <>
      {developmentBannerElement}
      <Toaster />
      <Suspense fallback={<PageSuspenseFallback />}>
      <Routes>
        {/* Redirect root to appropriate dashboard or login */}
        <Route
          path="/"
          element={<RootRedirect />}
        />
        {/* no /home route by design */}

        {/* Login Route */}
        <Route path="/login" element={<Login />} />
        {/* Self-service password reset (email magic link) */}
        <Route path="/forgot-password" element={<ForgotPassword />} />
        <Route path="/reset-password" element={<ResetPassword />} />

        {/* Logout */}
        <Route path="/logout" element={<LogoutRedirect />} />

        {/* Canonical dashboard — one URL for every company role */}
        <Route path="/dashboard" element={<DashboardLayoutRouter />}>
          <Route index element={<DashboardRoleRouter />} />
        </Route>

        {/* Canonical role-split surfaces — one URL, content by role. */}
        <Route path="/projects" element={<ProjectsShell />}>
          <Route index element={<ProjectsPageRouter />} />
          <Route path=":projectId/payrates/:payrateId/edit" element={<ResponsivePage desktopComponent={PayrateEditPage} mobileComponent={PayrateEditPageMobile} />} />
        </Route>
        <Route path="/employees" element={<EmployeesShell />}>
          <Route index element={<EmployeesPageRouter />} />
        </Route>
        {/* PRD §5.8: Sổ cái — company transfer ledger + sao kê (admin/ketoan/giamdoc). */}
        <Route path="/ledger" element={<LedgerShell />}>
          <Route index element={<ResponsivePage desktopComponent={PartnerLedgerPage} mobileComponent={PartnerLedgerPageMobile} />} />
        </Route>
        <Route path="/settings" element={<ProtectedRoute requiredRole="admin"><AdminLayout /></ProtectedRoute>}>
          <Route index element={<ResponsivePage desktopComponent={SettingsPage} mobileComponent={SettingsPageMobile} />} />
        </Route>
        <Route path="/audit-log" element={<ProtectedRoute requiredRole="admin"><AdminLayout /></ProtectedRoute>}>
          <Route index element={<ResponsivePage desktopComponent={AuditLogPage} mobileComponent={AuditLogPageMobile} />} />
        </Route>
        <Route path="/email" element={<ProtectedRoute requiredRole="admin"><AdminLayout /></ProtectedRoute>}>
          <Route index element={<EmailPage />} />
        </Route>
        {/* Old fee-insights URL — same page lives on /dashboard for these roles. */}
        <Route path="/overview" element={<Navigate to="/dashboard" replace />} />
        {/* Legacy /admin|/partner bookmarks → canonical, query kept. */}
        <Route path="/admin/*" element={<LegacyPrefixRedirect />} />
        <Route path="/partner/*" element={<LegacyPrefixRedirect />} />

        {/* PRD §5.3 (owner call): one canonical /users directory — admin +
            giamdoc full control. Legacy /admin/users + /partner/users
            bookmarks redirect via the /admin|/partner catch-alls above. */}
        <Route path="/users" element={<UsersShell />}>
          <Route index element={<ResponsivePage desktopComponent={UsersPage} mobileComponent={UsersPageMobile} />} />
        </Route>

        {/* PRD §5.7: Lịch sử trả lương — payment history under one canonical
            URL shared by quanly/ketoan/giamdoc/admin. Admin crosses into the
            console shell (PAYSUB-36); partner roles keep the partner layout. */}
        <Route path="/payment-history" element={<PaymentHistoryShell />}>
          <Route index element={<ResponsivePage desktopComponent={PartnerPaymentHistoryPage} mobileComponent={PartnerPaymentHistoryPageMobile} />} />
        </Route>

        {/* Bảng công — one canonical /timesheet shared by admin +
            partner roles. The shell follows the signed-in role (see
            TimesheetShell); PartnerLayout carries the partner-role guard. */}
        <Route path="/timesheet" element={<TimesheetShell />}>
          <Route index element={<ResponsivePage desktopComponent={TimesheetPage} mobileComponent={TimesheetPageMobile} />} />
        </Route>

        {/* Employee Routes */}
        <Route path="/employee" element={<ProtectedRoute requiredRole="nhanvien"><EmployeeRouter /></ProtectedRoute>} />

        {/* 404 Route */}
        <Route path="*" element={<NotFound />} />
      </Routes>
      </Suspense>

      {/* Modal Router - centralized URL-driven modal system */}
      <ModalRouter />

      {/* Blocks employee accounts still on the default password behind a compulsory change-password dialog. */}
      <ForceChangePasswordDialog />
    </>
  );
};

const App = () => {
  return (
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <TooltipProvider>
          <BrowserRouter>
            <SecureModalProvider>
              <AppProviders>
                <AppContent />
              </AppProviders>
            </SecureModalProvider>
          </BrowserRouter>
        </TooltipProvider>
      </QueryClientProvider>
    </ErrorBoundary>
  );
};

export default App;
