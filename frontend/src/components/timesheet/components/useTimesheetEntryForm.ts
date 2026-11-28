import { useState, useMemo, useEffect, useCallback } from "react";
import { format } from "date-fns";
import { vi } from "date-fns/locale";
import { NewTimesheetEntry, Timesheet } from "@/types/api/timesheet.types";
import {
  useCreateTimesheets,
  useUpdateTimesheet,
  useApproveTimesheet,
  useRejectTimesheet,
  useResetTimesheet,
} from "@/hooks/api/useTimesheets";
import { useCurrentPayRate, isForbiddenError } from "@/hooks/api/usePayRates";
import { useActiveProjectEmployees } from "@/hooks/api/useProjectEmployees";
import { useProject } from "@/hooks/api/useProjects";
import { useAuth } from "@/contexts/AuthContext";
import { canEditTimesheet } from "@/lib/permissions";
import { vietnameseEquals } from "@/utils/vietnameseNormalization";
import {
  calculateTimesheetPreviewAmount,
  findTimesheetRate,
} from "./timesheet-pay-unit";

interface FormData {
  hoursWorked: number;
  hourType: string;
  dayType: "Ngày thường" | "Ngày nghỉ" | "Ngày lễ";
}

interface UseTimesheetEntryFormProps {
  isOpen: boolean;
  onClose: () => void;
  projectId: number;
  employeeId: number;
  date: Date;
  existingEntry?: Timesheet | null;
  onSuccess?: () => void;
  onDelete?: (timesheet: Timesheet) => void;
}

export function useTimesheetEntryForm({
  isOpen,
  onClose,
  projectId,
  employeeId,
  date,
  existingEntry,
  onSuccess,
  onDelete,
}: UseTimesheetEntryFormProps) {
  const [formData, setFormData] = useState<FormData>({
    hoursWorked: 8,
    hourType: "",
    dayType: "Ngày thường",
  });

  const createTimesheetsMutation = useCreateTimesheets();
  const updateTimesheetMutation = useUpdateTimesheet();
  const approveTimesheetMutation = useApproveTimesheet();
  const rejectTimesheetMutation = useRejectTimesheet();
  const resetTimesheetMutation = useResetTimesheet();
  const { user } = useAuth();

  // Edit-capable payroll roles: only admin/quanly may call the payrate config
  // endpoints; ketoan/giamdoc get 403 — their modal reads from the row data.
  const hasPayrollAccess = user?.role === "admin" || user?.role === "quanly";

  const { data: payrateData, isLoading: isPayrateLoading, isError: isPayrateError, error: payrateError } =
    useCurrentPayRate(projectId, isOpen && !!projectId && hasPayrollAccess);
  // 403 = the user's role cannot access this project's payrate — show a
  // permission message instead of the "project has no payrate config" setup flow.
  const payrateForbidden = isForbiddenError(payrateError);
  const {
    data: projectData,
    isLoading: isPayUnitLoading,
    isError: isPayUnitError,
  } = useProject(projectId, isOpen && !!projectId && hasPayrollAccess);
  const { data: projectEmployeesData } = useActiveProjectEmployees(projectId, isOpen && !!projectId && hasPayrollAccess);
  const projectPayUnit = projectData
    ? projectData.is_flexible
      ? "shift"
      : "hour"
    : null;
  // Without payroll access (ketoan/giamdoc) the pay unit is unknown — the rate
  // unit then falls back to the flag stored on the timesheet row itself.
  const isFlexibleProject = projectPayUnit === "shift" || !!existingEntry?.projectIsFlexible;

  const isEditing = !!existingEntry;
  const isReadOnly = existingEntry ? !canEditTimesheet(existingEntry, user?.role) : false;

  const displayProjectName = existingEntry?.projectName ?? "";
  const displayEmployeeName = existingEntry?.employeeName ?? "";
  const displayEmployeeCode = existingEntry?.employeeCode ?? existingEntry?.employeeCCCD ?? "";

  const { availableHourTypes, availableDayTypes } = useMemo(() => {
    if (!payrateData?.rates) return { availableHourTypes: [], availableDayTypes: [] };

    const hourTypesSet = new Set<string>();
    const dayTypesSet = new Set<string>();

    Object.values(payrateData.rates).forEach((positionRates) => {
      if (typeof positionRates === "object" && positionRates !== null) {
        Object.entries(positionRates).forEach(([dayType, dayRates]) => {
          dayTypesSet.add(dayType);
          if (typeof dayRates === "object" && dayRates !== null) {
            Object.keys(dayRates).forEach((hourType) => hourTypesSet.add(hourType));
          }
        });
      }
    });

    if (existingEntry?.hour_type) {
      const existsInConfig = Array.from(hourTypesSet).some((ht) =>
        vietnameseEquals(ht, existingEntry.hour_type!),
      );
      if (!existsInConfig) hourTypesSet.add(existingEntry.hour_type);
    }

    return { availableHourTypes: Array.from(hourTypesSet), availableDayTypes: Array.from(dayTypesSet) };
  }, [payrateData, existingEntry]);

  const isHourTypeValid = useMemo(() => {
    if (!formData.hourType) return false;
    if (availableHourTypes.length === 0) return false;
    return availableHourTypes.some((ht) => vietnameseEquals(ht, formData.hourType));
  }, [formData.hourType, availableHourTypes]);

  const dateValidation = useMemo(() => {
    if (!employeeId || !projectEmployeesData?.data) return { isValid: true, errorMessage: "" };

    const assignment = projectEmployeesData.data.find((emp) => emp.employee_id === employeeId);
    if (!assignment?.start_date) return { isValid: true, errorMessage: "" };

    const entryDate = new Date(date);
    const startDate = new Date(assignment.start_date);

    if (entryDate < startDate) {
      return {
        isValid: false,
        errorMessage: `Ngày chấm công không thể trước ngày bắt đầu làm việc (${format(startDate, "dd/MM/yyyy", { locale: vi })})`,
      };
    }
    return { isValid: true, errorMessage: "" };
  }, [employeeId, date, projectEmployeesData]);

  // Initialize form data when modal opens or entry changes
  useEffect(() => {
    if (isOpen) {
      if (existingEntry) {
        const matchedHourType = availableHourTypes.find((ht) =>
          vietnameseEquals(ht, existingEntry.hour_type),
        );
        setFormData({
          hoursWorked: existingEntry.hours_worked,
          hourType: matchedHourType || existingEntry.hour_type,
          dayType: existingEntry.day_type as FormData["dayType"],
        });
      } else {
        const normalDayType = availableDayTypes.find((dt) =>
          dt.toLowerCase().includes("ngày thường"),
        );
        setFormData({
          hoursWorked: 8,
          hourType: availableHourTypes[0] || "",
          dayType: (normalDayType || availableDayTypes[0] || "") as FormData["dayType"],
        });
      }
    }
  }, [isOpen, existingEntry, availableHourTypes, availableDayTypes]);

  // Save logic
  const performSave = useCallback(async () => {
    if (isEditing && existingEntry) {
      const position = existingEntry.paytype.split(".")[0] || "";
      const fullPaytype = `${position}.${formData.dayType}.${formData.hourType}`;
      await updateTimesheetMutation.mutateAsync({
        id: existingEntry.id,
        data: { hours_worked: formData.hoursWorked, paytype: fullPaytype },
      });
    } else {
      const entries: NewTimesheetEntry[] = [{
        projectId, employeeId,
        date: format(date, "yyyy-MM-dd"),
        hoursWorked: formData.hoursWorked,
        hourType: formData.hourType,
        dayType: formData.dayType,
      }];
      await createTimesheetsMutation.mutateAsync(entries);
    }
  }, [isEditing, existingEntry, formData, projectId, employeeId, date, updateTimesheetMutation, createTimesheetsMutation]);

  const handleSave = useCallback(async () => {
    try {
      await performSave();
      await new Promise((r) => setTimeout(r, 50));
      if (onSuccess) await onSuccess();
      onClose();
    } catch (error) { console.error("Timesheet operation failed:", error); }
  }, [performSave, onSuccess, onClose]);

  const canSaveAndApprove =
    (user?.role === "admin" || user?.role === "quanly" || user?.role === "ketoan") &&
    existingEntry?.status === "pending_approval";

  const handleSaveAndApprove = useCallback(async () => {
    if (!existingEntry) return;
    try {
      await performSave();
      await approveTimesheetMutation.mutateAsync(existingEntry.id);
      await new Promise((r) => setTimeout(r, 50));
      if (onSuccess) await onSuccess();
      onClose();
    } catch (error) { console.error("Save and approve failed:", error); }
  }, [existingEntry, performSave, approveTimesheetMutation, onSuccess, onClose]);

  const isLoading =
    createTimesheetsMutation.isPending ||
    updateTimesheetMutation.isPending ||
    approveTimesheetMutation.isPending ||
    rejectTimesheetMutation.isPending ||
    resetTimesheetMutation.isPending;

  const handleDelete = useCallback(async () => {
    if (!existingEntry || !onDelete) return;
    try { await onDelete(existingEntry); onClose(); } catch { /* parent handles */ }
  }, [existingEntry, onDelete, onClose]);

  const handleFormChange = useCallback((field: keyof FormData, value: string | number) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
  }, []);

  // Accept both `,` and `.` decimal separators — Vietnamese keyboards type
  // "8,5" for eight and a half hours.
  const handleHoursChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = e.target.value;
    if (raw === "" || /^\d*[.,]?\d*$/.test(raw)) {
      const parsed = parseFloat(raw.replace(",", "."));
      setFormData((prev) => ({ ...prev, hoursWorked: raw === "" ? 0 : parsed || 0 }));
    }
  }, []);

  const handleApprove = useCallback(async () => {
    if (!existingEntry) return;
    try {
      await approveTimesheetMutation.mutateAsync(existingEntry.id);
      await new Promise((r) => setTimeout(r, 50));
      if (onSuccess) await onSuccess();
      onClose();
    } catch (error) { console.error("Approve failed:", error); }
  }, [existingEntry, approveTimesheetMutation, onSuccess, onClose]);

  const handleReject = useCallback(async () => {
    if (!existingEntry) return;
    try {
      await rejectTimesheetMutation.mutateAsync({ id: existingEntry.id, data: { rejection_reason: "Loại từ modal chấm công" } });
      await new Promise((r) => setTimeout(r, 50));
      if (onSuccess) await onSuccess();
      onClose();
    } catch (error) { console.error("Reject failed:", error); }
  }, [existingEntry, rejectTimesheetMutation, onSuccess, onClose]);

  const handleReset = useCallback(async () => {
    if (!existingEntry) return;
    try {
      await resetTimesheetMutation.mutateAsync({
        id: existingEntry.id,
        employee_id: existingEntry.employee_id,
        project_id: existingEntry.project_id,
      });
      if (onSuccess) await onSuccess();
      onClose();
    } catch (error) { console.error("Reset failed:", error); }
  }, [existingEntry, resetTimesheetMutation, onSuccess, onClose]);

  // Live payrate calculation
  const liveAmount = useMemo(() => {
    if (
      !payrateData?.rates ||
      !formData.hourType ||
      !formData.dayType ||
      projectPayUnit === null
    ) return null;

    const assignmentPosition = projectEmployeesData?.data.find(
      (employee) => employee.employee_id === employeeId,
    )?.position;
    const position =
      existingEntry?.paytype.split(".")[0] || assignmentPosition || "";
    const rate = findTimesheetRate(
      payrateData.rates,
      position,
      formData.dayType,
      formData.hourType,
    );
    if (rate === null) return null;

    return calculateTimesheetPreviewAmount(
      rate,
      formData.hoursWorked,
      isFlexibleProject,
    );
  }, [
    payrateData,
    formData.hourType,
    formData.dayType,
    formData.hoursWorked,
    projectPayUnit,
    isFlexibleProject,
    projectEmployeesData,
    employeeId,
    existingEntry,
  ]);

  const statusBadge = existingEntry ? (() => {
    const cfg: Record<string, { label: string; cls: string }> = {
      pending_approval: { label: "Chờ duyệt", cls: "bg-amber-500/20 text-amber-200 border border-amber-400/30" },
      approved: { label: "Đã duyệt", cls: "bg-emerald-500/20 text-emerald-200 border border-emerald-400/30" },
      rejected: { label: "Đã loại", cls: "bg-red-500/20 text-red-200 border border-red-400/30" },
    };
    const s = cfg[existingEntry.status];
    return s ? { label: s.label, cls: s.cls } : null;
  })() : null;

  return {
    // Form state
    formData,
    // Loading
    isLoading,
    isPayrateLoading,
    isPayrateError,
    payrateForbidden,
    isPayUnitLoading,
    isPayUnitError,
    // Computed
    availableHourTypes,
    availableDayTypes,
    isHourTypeValid,
    dateValidation,
    liveAmount,
    displayAmount:
      hasPayrollAccess
        ? projectPayUnit === null
          ? null
          : liveAmount ?? existingEntry?.amount ?? null
        : existingEntry?.amount ?? null,
    displayPayrate: existingEntry?.payrate ?? null,
    isFlexibleProject,
    displayProjectName,
    displayEmployeeName,
    displayEmployeeCode,
    isReadOnly,
    isEditing,
    canSaveAndApprove,
    statusBadge,
    user,
    // Mutations (for pending states in JSX)
    rejectTimesheetMutation,
    resetTimesheetMutation,
    // Handlers
    handleSave,
    handleSaveAndApprove,
    handleDelete,
    handleFormChange,
    handleHoursChange,
    handleApprove,
    handleReject,
    handleReset,
  };
}
