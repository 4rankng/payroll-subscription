import { ColumnDef } from "@tanstack/react-table";
import { Project } from "@/types/api/project.types";
import { formatEmployeeCount, formatSalaryPeriodLabel } from "@/utils/projectHelpers";

interface CreateProjectColumnsProps {
  pagination?: {
    page: number;
    pageSize: number;
  };
}

export const createProjectColumns = (
  props: CreateProjectColumnsProps = {},
): ColumnDef<Project>[] => [
  {
    accessorKey: "name",
    header: "Dự án",
    size: 240,
    minSize: 180,
    cell: ({ row }) => (
      <div className="flex flex-col min-w-0 gap-0.5 py-0.5">
        <div
          className="typography-body-medium font-semibold text-foreground leading-snug line-clamp-1 break-words"
          title={row.getValue("name") as string}
        >
          {row.getValue("name")}
        </div>
        {row.original.code && (
          <span className="typography-label-medium text-muted-foreground font-mono">
            {row.original.code}
          </span>
        )}
      </div>
    ),
  },
  {
    accessorKey: "client_name",
    header: "Khách hàng",
    size: 160,
    maxSize: 200,
    cell: ({ row }) => (
      <span
        className="typography-body-medium text-foreground/80 line-clamp-1 break-words"
        title={row.getValue("client_name") as string}
      >
        {row.getValue("client_name") || "—"}
      </span>
    ),
  },
  {
    id: "salary_period",
    header: "Kỳ lương",
    size: 130,
    maxSize: 150,
    cell: ({ row }) => {
      const project = row.original;
      const label = formatSalaryPeriodLabel(project.salary_period_from, project.salary_period_to);
      return (
        <span className="typography-body-medium text-muted-foreground">
          {label || "—"}
        </span>
      );
    },
  },
  {
    id: "employee_summary",
    header: () => (
      <div className="text-center">Nhân viên</div>
    ),
    size: 140,
    maxSize: 160,
    cell: ({ row }) => {
      // The projects list payload carries a single `employee_count` (verified
      // against GET /api/v1/projects); the weekly/monthly split fields are
      // only populated by detail endpoints, so fall back to the list total
      // instead of rendering a false "—" for projects that do have staff.
      const weekly = row.original.weekly_salary_employee_count || 0;
      const monthly = row.original.monthly_salary_employee_count || 0;
      const splitTotal = weekly + monthly;
      const total = splitTotal > 0 ? splitTotal : row.original.employee_count || 0;

      if (total === 0) {
        return (
          <div className="flex justify-center">
            <span className="typography-body-medium text-muted-foreground">—</span>
          </div>
        );
      }

      // Admin list payload carries only the plain employee_count (the split
      // fields come from detail endpoints) — render the total so the column
      // is never blank when the split rows are absent.
      if (weekly === 0 && monthly === 0) {
        return (
          <div className="flex justify-center">
            <span className="typography-body-medium font-semibold text-foreground tabular-nums">
              {formatEmployeeCount(total)}
            </span>
          </div>
        );
      }

      return (
        <div className="flex flex-col items-start gap-0.5">
          {weekly > 0 && (
            <div className="flex items-center gap-1.5">
              <span className="typography-body-medium font-semibold text-foreground tabular-nums">{formatEmployeeCount(weekly)}</span>
              <span className="typography-label-medium text-muted-foreground">Lương tuần</span>
            </div>
          )}
          {monthly > 0 && (
            <div className="flex items-center gap-1.5">
              <span className="typography-body-medium font-semibold text-foreground tabular-nums">{formatEmployeeCount(monthly)}</span>
              <span className="typography-label-medium text-muted-foreground">Lương tháng</span>
            </div>
          )}
        </div>
      );
    },
  },
];
