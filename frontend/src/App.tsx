import React, { Suspense } from "react";
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
  authManager.clearSession();
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
/** Legacy /partner|/admin payment-history bookmarks → canonical URL, query kept. */
const PaymentHistoryRedirect = () => {
  const search = useLocation().search;
  return <Navigate replace to={{ pathname: "/payment-history", search }} />;
};

/** Legacy /admin|/partner timesheet bookmarks → canonical URL, query kept. */
const TimesheetRedirect = () => {
  const search = useLocation().search;
  return <Navigate replace to={{ pathname: "/timesheet", search }} />;
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

        {/* Admin Routes */}
        <Route path="/admin" element={<ProtectedRoute requiredRole="admin"><AdminLayout /></ProtectedRoute>}>
          {/* PRD §5.1: the old financial admin dashboard is replaced by the fee-insights overview. */}
          <Route index element={<ResponsivePage desktopComponent={FeeInsightsPage} mobileComponent={FeeInsightsPageMobile} />} />
          <Route path="dashboard" element={<Navigate to="/dashboard" replace />} />
          <Route path="users" element={<ResponsivePage desktopComponent={UsersPage} mobileComponent={UsersPageMobile} />} />
          <Route path="projects" element={<ResponsivePage desktopComponent={ProjectsPage} mobileComponent={ProjectsPageMobile} />} />
          <Route path="employees" element={<ResponsivePage desktopComponent={EmployeesPage} mobileComponent={EmployeesPageMobile} />} />
          {/* Bảng công moved to the canonical /timesheet route below. */}
          <Route path="settings" element={<ResponsivePage desktopComponent={SettingsPage} mobileComponent={SettingsPageMobile} />} />
          <Route path="audit-log" element={<ResponsivePage desktopComponent={AuditLogPage} mobileComponent={AuditLogPageMobile} />} />
          <Route path="email" element={<EmailPage />} />
          <Route path="projects/:projectId/payrates/:payrateId/edit" element={<ResponsivePage desktopComponent={PayrateEditPage} mobileComponent={PayrateEditPageMobile} />} />
          <Route path="*" element={<NotFound />} />
        </Route>

        {/* Partner Routes — quanly operates; giamdoc + ketoan read and export reports (D28) */}
        <Route path="/partner" element={<ProtectedRoute requiredRole={["quanly", "ketoan", "giamdoc"]}><PartnerLayout /></ProtectedRoute>}>
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<Navigate to="/dashboard" replace />} />
          {/* PRD §5.1: fee-insights overview — ketoan/giamdoc landing (PRD §4). */}
          <Route path="overview" element={<ResponsivePage desktopComponent={FeeInsightsPage} mobileComponent={FeeInsightsPageMobile} />} />
          {/* PRD §5.3: user directory — giamdoc reads /partner/users read-only;
              user-management writes are admin-only (also enforced by casbin). */}
          <Route path="users" element={<ResponsivePage desktopComponent={UsersPage} mobileComponent={UsersPageMobile} />} />
          <Route path="projects" element={<ResponsivePage desktopComponent={PartnerProjectsPage} mobileComponent={PartnerProjectsPageMobile} />} />
          <Route path="employees" element={<ResponsivePage desktopComponent={PartnerEmployeesPage} mobileComponent={PartnerEmployeesPageMobile} />} />
          {/* PRD §5.8: Sổ cái — company transfer ledger + sao kê (ketoan/giamdoc). */}
          <Route path="ledger" element={<ProtectedRoute requiredRole={["ketoan", "giamdoc"]}><ResponsivePage desktopComponent={PartnerLedgerPage} mobileComponent={PartnerLedgerPageMobile} /></ProtectedRoute>} />
          <Route path="projects/:projectId/payrates/:payrateId/edit" element={<ResponsivePage desktopComponent={PayrateEditPage} mobileComponent={PayrateEditPageMobile} />} />
        </Route>

        {/* PRD §5.7: Lịch sử trả lương — payment history under one canonical
            URL shared by quanly/ketoan/giamdoc/admin. PartnerLayout
            carries the role guard internally; this route widens it to admin. */}
        <Route path="/payment-history" element={<PartnerLayout roles={["quanly", "ketoan", "giamdoc", "admin"]} />}>
          <Route index element={<ResponsivePage desktopComponent={PartnerPaymentHistoryPage} mobileComponent={PartnerPaymentHistoryPageMobile} />} />
        </Route>
        {/* Legacy bookmarks redirect to the canonical URL, query preserved. */}
        <Route path="/partner/payment-history" element={<PaymentHistoryRedirect />} />
        <Route path="/admin/payment-history" element={<PaymentHistoryRedirect />} />

        {/* Bảng công — one canonical /timesheet shared by admin +
            partner roles. The shell follows the signed-in role (see
            TimesheetShell); PartnerLayout carries the partner-role guard. */}
        <Route path="/timesheet" element={<TimesheetShell />}>
          <Route index element={<ResponsivePage desktopComponent={TimesheetPage} mobileComponent={TimesheetPageMobile} />} />
        </Route>
        {/* Legacy bookmarks redirect to the canonical URL, query preserved. */}
        <Route path="/admin/timesheet" element={<TimesheetRedirect />} />
        <Route path="/admin/timesheets" element={<TimesheetRedirect />} />
        <Route path="/partner/timesheet" element={<TimesheetRedirect />} />

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
