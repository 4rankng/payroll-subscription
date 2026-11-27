/**
 * Thẻ Phí subscription (PRD §5.9): admin edits
 * subscription_fee_per_employee_per_ky_vnd — step 1.000 VND, max 999.999.999.
 * Reads/writes the settings row by key; the current value also reaches the
 * dashboards via the billing payloads, so no other role needs settings read.
 */
import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useSettingByKey, useUpdateSetting } from "@/hooks/api/useSettings";
import { toast } from "@/components/ui/sonner";

const FEE_KEY = "subscription_fee_per_employee_per_ky_vnd";
const FEE_MAX = 999_999_999;

const formatVnd = (n: number) => new Intl.NumberFormat("vi-VN").format(n);

export function FeeSettingsCard() {
  const queryClient = useQueryClient();
  const { data, isLoading } = useSettingByKey(FEE_KEY);
  const updateSetting = useUpdateSetting();

  const setting = data;
  const [feeText, setFeeText] = useState("");

  useEffect(() => {
    if (setting?.value !== undefined && setting?.value !== null) {
      setFeeText(String(setting.value));
    }
  }, [setting?.value]);

  const parsed = Number(feeText);
  const invalid =
    feeText === "" ||
    !Number.isInteger(parsed) ||
    parsed < 0 ||
    parsed > FEE_MAX ||
    parsed % 1000 !== 0;

  const handleSave = () => {
    if (!setting || invalid) return;
    updateSetting.mutate(
      { id: setting.id, data: { value: String(parsed), value_type: "number" } },
      {
        onSuccess: () => {
          toast({ title: "Đã lưu phí subscription", description: `${formatVnd(parsed)} VND / nhân viên / Kỳ` });
          queryClient.invalidateQueries({ queryKey: ["settings"] });
        },
        onError: (err: Error) => toast({ title: "Không lưu được phí", description: err.message, variant: "destructive" }),
      },
    );
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Phí subscription</CardTitle>
        <CardDescription>
          Phí cố định tính theo nhân viên distinct × Kỳ (VND/NV/Kỳ). Bước nhảy 1.000đ, tối đa {formatVnd(FEE_MAX)}đ.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-3 sm:flex-row sm:items-end">
        <div className="w-full max-w-xs space-y-1.5">
          <Label htmlFor="subscription-fee">Phí (VND/NV/kỳ)</Label>
          <Input
            id="subscription-fee"
            type="number"
            inputMode="numeric"
            step={1000}
            min={0}
            max={FEE_MAX}
            value={feeText}
            disabled={isLoading || updateSetting.isPending}
            onChange={(e) => setFeeText(e.target.value)}
            aria-invalid={invalid || undefined}
          />
          {invalid && feeText !== "" && (
            <p className="text-xs text-destructive">
              Phí phải là bội số của 1.000, từ 0 đến {formatVnd(FEE_MAX)}.
            </p>
          )}
          {!invalid && feeText !== "" && (
            <p className="text-xs text-muted-foreground">{formatVnd(parsed)} VND / nhân viên / Kỳ</p>
          )}
        </div>
        <Button onClick={handleSave} disabled={invalid || isLoading || updateSetting.isPending}>
          {updateSetting.isPending ? "Đang lưu…" : "Lưu"}
        </Button>
      </CardContent>
    </Card>
  );
}

export default FeeSettingsCard;
