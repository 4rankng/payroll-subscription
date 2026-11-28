import { useState, useMemo, useCallback, useEffect } from "react";
import { format } from "date-fns";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Unlink, User, Calendar, ChevronLeft, ChevronRight } from "lucide-react";
import { Project } from "@/types/api/project.types";
import { ProjectEmployeeAssignment, ProjectEmployeeListParams } from "@/types/api/project-employee.types";
import { FilterPill } from "@/components/shared/FilterPill";
import { SearchBar } from "@/components/shared/SearchBar";
import { canPartnerManageProject } from "@/lib/permissions";
import { formatPositionLabel } from "@/components/projects/details/shift-pay-summary";
import { cn } from "@/lib/utils";
import { useProjectEmployeeManagement } from "@/hooks/business/useProjectEmployeeManagement";
import { CheckInToggle } from "./CheckInToggle";

interface ProjectEmployeesListProps {
  project: Project;
  onEmployeeRemoved?: () => void;
  onAddEmployees?: () => void;
  onTotalCountChange?: (count: number) => void;
}

export function ProjectEmployeesList({
  project,
  onEmployeeRemoved,
  onAddEmployees,
  onTotalCountChange,
}: ProjectEmployeesListProps) {
  // UI state
  const [removingId, setRemovingId] = useState<number | null>(null);
  const [showRemoveDialog, setShowRemoveDialog] = useState(false);
  const [selectedEmployeeToRemove, setSelectedEmployeeToRemove] =
    useState<ProjectEmployeeAssignment | null>(null);
  const [lastDate, setLastDate] = useState<string>("");
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize] = useState(20);
  const [checkInFilter, setCheckInFilter] = useState<string>("all");
  const [searchTerm, setSearchTerm] = useState("");

  // Build API params — all pagination & filtering handled by backend
  const apiParams = useMemo<ProjectEmployeeListParams>(() => {
    const params: ProjectEmployeeListParams = { page: currentPage, pageSize };
    if (checkInFilter !== "all") {
      params.check_in_enabled = checkInFilter === "on";
    }
    const trimmed = searchTerm.trim();
    if (trimmed) {
      params.search = trimmed;
    }
    return params;
  }, [currentPage, pageSize, checkInFilter, searchTerm]);

  // Business logic
  const {
    employees,
    totalCount: apiTotalCount,
    totalPages: apiTotalPages,
    isLoading,
    isRemoving,
    error,
    errorMessage,
    removeEmployee,
    isEmpty,
    getAssignmentMeta,
  } = useProjectEmployeeManagement({
    project,
    params: apiParams,
    onEmployeeRemoved,
  });

  // Notify parent of total count
  useEffect(() => {
    onTotalCountChange?.(apiTotalCount);
  }, [apiTotalCount, onTotalCountChange]);

  // UI event handlers
  const handleRemoveEmployee = useCallback(
    async (assignment?: ProjectEmployeeAssignment) => {
      const employeeToRemove = assignment || selectedEmployeeToRemove;
      if (!employeeToRemove) return;

      setRemovingId(employeeToRemove.id);
      setShowRemoveDialog(false);

      const success = await removeEmployee(
        employeeToRemove,
        lastDate || undefined,
      );

      if (success) {
        setSelectedEmployeeToRemove(null);
        setLastDate("");
      }

      setRemovingId(null);
    },
    [selectedEmployeeToRemove, lastDate, removeEmployee],
  );

  const openRemoveDialog = useCallback(
    (assignment: ProjectEmployeeAssignment) => {
      setSelectedEmployeeToRemove(assignment);
      setShowRemoveDialog(true);
    },
    [],
  );

  const closeRemoveDialog = useCallback(() => {
    setShowRemoveDialog(false);
    setSelectedEmployeeToRemove(null);
    setLastDate("");
  }, []);

  // Reset to page 1 when filter changes
  const handleCheckInFilterChange = useCallback((value: string) => {
    setCheckInFilter(value);
    setCurrentPage(1);
  }, []);

  // Reset to page 1 when search changes (debounced in SearchBar)
  const handleSearchChange = useCallback((value: string) => {
    setSearchTerm(value);
    setCurrentPage(1);
  }, []);

  // Loading state
  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="typography-body-medium text-muted-foreground">
          Đang tải danh sách nhân viên...
        </div>
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="typography-body-medium text-destructive">
          Lỗi khi tải danh sách nhân viên: {errorMessage}
        </div>
      </div>
    );
  }

  const hasSearch = searchTerm.trim().length > 0;

  const totalPages = apiTotalPages || Math.max(1, Math.ceil(apiTotalCount / pageSize));
  const startIndex = (currentPage - 1) * pageSize;

  // Display-only employment state, derived from the same date semantics the
  // management hook uses (compare plain yyyy-MM-dd keys, no TZ conversion).
  const todayKey = format(new Date(), 'yyyy-MM-dd');
  // Removal is a write action: only quanly/admin see it.
  const canRemoveEmployees = canPartnerManageProject();

  const assignmentStatePill = (
    employee: ProjectEmployeeAssignment & { meta?: { canCancel: boolean } }
  ): { label: string; pill: string } => {
    const startDay = (employee.start_date || '').slice(0, 10);
    const endDay = employee.last_date ? employee.last_date.slice(0, 10) : null;
    if (startDay > todayKey || employee.meta?.canCancel) {
      return { label: 'Chưa bắt đầu', pill: 'bg-info/10 text-info ring-1 ring-inset ring-info/30' };
    }
    if (endDay && endDay < todayKey) {
      return { label: 'Đã kết thúc', pill: 'bg-muted text-muted-foreground ring-1 ring-inset ring-border' };
    }
    return { label: 'Đang làm', pill: 'bg-success/10 text-success ring-1 ring-inset ring-success/30' };
  };

  return (
    <div className="h-full flex flex-col overflow-hidden">
      {/* Filter row */}
      <div className="flex items-center gap-2 pt-2 pb-1">
        <SearchBar
          searchTerm={searchTerm}
          onSearchChange={handleSearchChange}
          placeholder="Tìm theo tên, CCCD hoặc mã nhân viên"
          className="flex-1 sm:max-w-xs"
        />
        {project.is_flexible && (
          <FilterPill
            value={checkInFilter}
            onChange={handleCheckInFilterChange}
            placeholder="Điểm danh"
            options={[
              { value: "on", label: "Bật" },
              { value: "off", label: "Tắt" },
            ]}
            contentClassName="bg-white border-gray-200 shadow-lg"
          />
        )}
      </div>

      {/* Employee dense table — spans the full drawer width */}
      <div className="flex-1 overflow-auto pt-2">
        {isEmpty ? (
          <div className="flex h-full flex-col items-center justify-center space-y-4">
            <User className="w-12 h-12 text-muted-foreground" />
            <div className="text-center space-y-2">
              <h3 className="font-medium">
                {hasSearch
                  ? "Không tìm thấy nhân viên phù hợp"
                  : "Chưa có nhân viên nào được giao"}
              </h3>
              <p className="typography-body-medium text-muted-foreground">
                {hasSearch
                  ? `Không có kết quả nào cho "${searchTerm.trim()}"`
                  : project.status === "active"
                    ? 'Nhấn nút "Thêm nhân viên" ở cuối trang để giao nhân viên vào dự án này'
                    : "Chỉ có thể thêm nhân viên vào dự án đang hoạt động"}
              </p>
            </div>
          </div>
        ) : employees.length === 0 ? (
          <div className="flex items-center justify-center h-32 text-muted-foreground typography-body-medium">
            Không tìm thấy nhân viên
          </div>
        ) : (
          <div className="overflow-x-auto rounded-lg border border-border/60">
            <table className="w-full min-w-[460px] border-collapse text-left">
              <thead>
                <tr className="border-b border-border bg-muted/50">
                  <th className="h-8 px-3 text-[11px] font-semibold leading-[1.35] tracking-[0.03em] text-muted-foreground">Họ tên</th>
                  <th className="h-8 px-3 text-[11px] font-semibold leading-[1.35] tracking-[0.03em] text-muted-foreground">Vị trí</th>
                  <th className="h-8 px-3 text-[11px] font-semibold leading-[1.35] tracking-[0.03em] text-muted-foreground">Trạng thái</th>
                  <th className="h-8 px-3 text-right text-[11px] font-semibold leading-[1.35] tracking-[0.03em] text-muted-foreground">Thao tác</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border/40">
                {employees.map((employee) => {
                  const state = assignmentStatePill(employee);
                  const removable =
                    canRemoveEmployees &&
                    (employee.meta.canRemove || employee.meta.canCancel);
                  return (
                    <tr key={employee.id} className="align-middle">
                      <td className="max-w-[180px] px-3 py-2">
                        <p className="truncate text-[12px] font-semibold text-foreground">
                          {employee.employee_name || employee.employee_code || "-"}
                        </p>
                        <p className="truncate font-mono text-[11px] text-muted-foreground">
                          {employee.employee_code
                            ? `MSNV: ${employee.employee_code}`
                            : `CCCD: ${employee.employee_cccd || "-"}`}
                        </p>
                      </td>
                      <td className="whitespace-nowrap px-3 py-2 text-[12px] text-foreground/85">
                        {employee.position ? formatPositionLabel(employee.position) : "—"}
                      </td>
                      <td className="px-3 py-2">
                        <span
                          className={cn(
                            'inline-flex shrink-0 items-center whitespace-nowrap rounded-full px-2 py-0.5 text-[11px] font-semibold',
                            state.pill,
                          )}
                        >
                          {state.label}
                        </span>
                      </td>
                      <td className="px-3 py-2">
                        <div className="flex items-center justify-end gap-1.5">
                          {project.is_flexible && (
                            <CheckInToggle
                              assignment={employee}
                              disabled={project.status !== "active"}
                            />
                          )}
                          {removable ? (
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-7 px-2 text-[11px] font-semibold text-destructive hover:bg-destructive/10 hover:text-destructive"
                              onClick={() => openRemoveDialog(employee)}
                              disabled={removingId === employee.id}
                            >
                              <Unlink className="h-3 w-3" aria-hidden="true" />
                              Gỡ
                            </Button>
                          ) : null}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between px-4 py-2 border-t border-border/40">
          <span className="text-xs text-muted-foreground">
            {startIndex + 1}–{Math.min(startIndex + pageSize, apiTotalCount)}/{apiTotalCount}
          </span>
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
              disabled={currentPage === 1}
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
              let page: number;
              if (totalPages <= 5) {
                page = i + 1;
              } else if (currentPage <= 3) {
                page = i + 1;
              } else if (currentPage >= totalPages - 2) {
                page = totalPages - 4 + i;
              } else {
                page = currentPage - 2 + i;
              }
              return (
                <Button
                  key={page}
                  variant={page === currentPage ? "outline" : "ghost"}
                  size="icon"
                  className="h-7 w-7 text-xs"
                  onClick={() => setCurrentPage(page)}
                >
                  {page}
                </Button>
              );
            })}
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
              disabled={currentPage === totalPages}
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}

      {/* Remove Employee Dialog */}
      <Dialog open={showRemoveDialog} onOpenChange={setShowRemoveDialog}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Unlink className="h-5 w-5 text-destructive" />
              Xác nhận gỡ nhân viên
            </DialogTitle>
            <DialogDescription>
              Gỡ{" "}
              <strong>
                {selectedEmployeeToRemove?.employee_name ||
                  selectedEmployeeToRemove?.employee_code ||
                  "nhân viên này"}
              </strong>{" "}
              khỏi dự án <strong>{project.name}</strong>
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label
                htmlFor="last-date"
                className="text-sm font-medium flex items-center gap-2"
              >
                <Calendar className="h-4 w-4" />
                Ngày kết thúc
              </Label>
              <Input
                id="last-date"
                type="date"
                value={lastDate}
                onChange={(e) => setLastDate(e.target.value)}
                className="w-full"
              />
              <p className="text-xs text-muted-foreground">
                Để trống để kết thúc ngay lập tức
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={closeRemoveDialog}
              disabled={removingId === selectedEmployeeToRemove?.id}
            >
              Đóng
            </Button>
            <Button
              type="button"
              variant="destructive"
              onClick={() => handleRemoveEmployee()}
              disabled={removingId === selectedEmployeeToRemove?.id}
            >
              {removingId === selectedEmployeeToRemove?.id
                ? "Đang gỡ..."
                : "Gỡ khỏi dự án"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
