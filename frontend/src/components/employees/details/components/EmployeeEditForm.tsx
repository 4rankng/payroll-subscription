import { useState } from "react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { formatVietnameseName } from "@/lib/validation";
import { cn } from "@/lib/utils";
import { User, AlertCircle } from "lucide-react";
import type { EmployeeFormProps } from "../types";

function FieldGroup({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-lg border bg-card p-4 space-y-3">
      {children}
    </div>
  );
}

function SectionTitle({ icon: Icon, label }: { icon: React.ElementType; label: string }) {
  return (
    <div className="flex items-center gap-2 mb-1">
      <div className="h-6 w-6 rounded-lg bg-primary/10 flex items-center justify-center">
        <Icon className="h-3.5 w-3.5 text-primary" />
      </div>
      <span className="text-xs font-semibold text-foreground">{label}</span>
    </div>
  );
}

function Field({ id, label, children, warning }: { id?: string; label: string; children: React.ReactNode; warning?: string }) {
  return (
    <div className="space-y-1">
      <Label htmlFor={id} className="text-xs text-muted-foreground font-medium">{label}</Label>
      {children}
      {warning && (
        <p className="flex items-center gap-1 text-xs text-warning ">
          <AlertCircle className="h-3 w-3 shrink-0" />
          {warning}
        </p>
      )}
    </div>
  );
}

/** Returns true if the value contains characters that formatVietnameseName would strip */
function hasInvalidNameChars(value: string): boolean {
  if (!value) return false;
  return formatVietnameseName(value) !== value.trim().replace(/\s+/g, ' ');
}

export function EmployeeEditForm({
  formData,
  onInputChange,
  focusMobile = false,
}: EmployeeFormProps) {
  const [nameWarning, setNameWarning] = useState(false);

  return (
    <div className="space-y-3">
      {/* Personal info card */}
      <FieldGroup>
        <SectionTitle icon={User} label="Thông tin cá nhân" />

        <div className="grid grid-cols-1 min-[420px]:grid-cols-2 gap-3">
          <Field
            id="fullname"
            label="Họ và tên"
            warning={nameWarning ? "Họ tên không được chứa số hoặc ký tự đặc biệt" : undefined}
          >
            <Input
              id="fullname"
              className={`h-11 text-sm sm:h-9 ${nameWarning ? "border-warning focus-visible:ring-warning" : ""}`}
              value={formData.fullname || ''}
              onChange={(e) => {
                onInputChange("fullname", e.target.value);
                setNameWarning(hasInvalidNameChars(e.target.value));
              }}
              onBlur={(e) => {
                const formatted = formatVietnameseName(e.target.value);
                if (formatted !== e.target.value) {
                  onInputChange("fullname", formatted);
                }
                setNameWarning(false);
              }}
            />
          </Field>
          <Field id="cccd" label="CCCD">
            <Input
              id="cccd"
              className="h-11 text-sm sm:h-9 font-mono tracking-wide"
              value={formData.cccd || ''}
              onChange={(e) => onInputChange("cccd", e.target.value)}
            />
          </Field>
        </div>

        <div className="grid grid-cols-1 min-[420px]:grid-cols-2 gap-3">
          <Field id="mobile" label="Số điện thoại">
            <Input
              id="mobile"
              autoFocus={focusMobile}
              className={cn(
                "h-11 text-sm sm:h-9",
                focusMobile && "ring-2 ring-primary ring-offset-1"
              )}
              value={formData.mobile || ''}
              onChange={(e) => onInputChange("mobile", e.target.value)}
            />
          </Field>
          <Field id="date_of_birth" label="Ngày sinh">
            <Input
              id="date_of_birth"
              type="date"
              className="h-11 text-sm sm:h-9"
              value={formData.date_of_birth || ''}
              onChange={(e) => onInputChange("date_of_birth", e.target.value)}
            />
          </Field>
        </div>

        <Field id="email" label="Email">
          <Input
            id="email"
            type="email"
            className="h-11 text-sm sm:h-9"
            placeholder="example@email.com"
            value={formData.email || ''}
            onChange={(e) => onInputChange("email", e.target.value)}
          />
        </Field>

        <Field id="address" label="Địa chỉ">
          <Input
            id="address"
            className="h-11 text-sm sm:h-9"
            value={formData.address || ''}
            onChange={(e) => onInputChange("address", e.target.value)}
          />
        </Field>
      </FieldGroup>
    </div>
  );
}
