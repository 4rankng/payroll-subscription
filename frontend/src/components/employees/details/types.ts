import type { Employee } from "@/types/api/employee.types";

export interface EmployeeDetailsProps {
  employee: Employee;
}

export interface EmployeeFormProps {
  employee: Employee;
  formData: Partial<Employee>;
  onInputChange: (field: keyof Employee, value: string) => void;
  /** DASH-06: autofocus + highlight the phone field when ?focus=mobile. */
  focusMobile?: boolean;
}

export interface EmployeeActionsProps {
  onEdit?: () => void;
  onSave?: () => void;
  onCancel?: () => void;
  onDelete?: () => void;
  onResetPassword?: () => void;
  onClose?: () => void;
  isDirty?: boolean;
}
