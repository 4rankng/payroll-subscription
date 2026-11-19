import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { PageHeader } from "@/components/shared/PageHeader";
import { getConsoleKicker } from "@/lib/console";
import { useLocation } from "react-router-dom";
import { TimesheetMonthSelector } from "@/components/timesheet/TimesheetMonthSelector";
import { TransferUploadPanel } from "@/components/billing/TransferUploadPanel";
import { UploadHistoryPanel, EmployeePayCycleSummary } from "@/components/billing/BillingSummaryPanels";
import { ReportPanel } from "@/components/billing/ReportPanel";
import { billingKeys } from "@/hooks/api/useBilling";
import { useSettingByKey } from "@/hooks/api/useSettings";
import { authManager } from "@/lib/auth";

/**
 * Thanh toán (PAYSUB billing, quanly portal): bank-transfer result uploads,
 * per-Kỳ employee billing summary and the month-end report. The fee comes
 * from the admin-managed subscription setting.
 */
const BillingPage = () => {
  const location = useLocation();
  const queryClient = useQueryClient();
  const now = new Date();
  const [workMonth, setWorkMonth] = useState(
    `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`,
  );

  const { data: feeSetting } = useSettingByKey("subscription_fee_per_employee_per_ky_vnd");
  const feeVnd = feeSetting?.value ? parseFloat(feeSetting.value) : 0;
  const canGenerate = authManager.getUserRole() === "giamdoc" || authManager.getUserRole() === "admin";

  const handleUploaded = () => {
    void queryClient.invalidateQueries({ queryKey: billingKeys.all });
  };

  return (
    <div className="mx-auto max-w-[1280px] space-y-3 p-3 sm:p-4">
      <PageHeader title="Thanh toán" kicker={getConsoleKicker(location.pathname)}>
        <TimesheetMonthSelector
          value={workMonth}
          onChange={setWorkMonth}
          className="lg:w-auto lg:shrink-0"
        />
      </PageHeader>

      <div className="grid grid-cols-1 items-start gap-3 xl:grid-cols-2">
        <TransferUploadPanel workMonth={workMonth} onUploaded={handleUploaded} />
        <UploadHistoryPanel workMonth={workMonth} />
        <EmployeePayCycleSummary workMonth={workMonth} feeVnd={feeVnd} />
        <ReportPanel workMonth={workMonth} canGenerate={canGenerate} />
      </div>
    </div>
  );
};

export default BillingPage;
