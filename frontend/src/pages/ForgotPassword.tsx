import { useState, useCallback } from "react";
import { Link, useNavigate } from "react-router-dom";
import { ArrowLeft, CheckCircle, Mail01 } from "@untitledui/icons";
import { Button, buttonVariants } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useRequestPasswordReset } from "@/hooks/api/usePasswordReset";
import { AuthBrandMark } from "@/components/auth/AuthBrandMark";
import { AuthCard } from "@/components/auth/AuthCard";
import { AuthSpinner } from "@/components/auth/AuthSpinner";
import { FeaturedIcon } from "@/components/foundations/featured-icon/featured-icon";
import { cx } from "@/utils/cx";

/**
 * ForgotPassword — step 1 of the self-service password-reset flow.
 *
 * A single input field accepts an email address and triggers the Resend
 * magic-link flow → "check your inbox" state.
 *
 * Only registered emails get a real send; unknown inputs get the same generic
 * success response (anti-enumeration). The success state is shown regardless
 * of network outcome so a timing/connection difference can't leak which
 * emails are registered.
 */
const ForgotPassword = () => {
  const [identifier, setIdentifier] = useState("");
  const [fieldError, setFieldError] = useState<string | null>(null);
  const [done, setDone] = useState(false);
  const navigate = useNavigate();

  const emailMutation = useRequestPasswordReset();

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      const trimmed = identifier.trim();
      if (!trimmed) {
        setFieldError("Vui lòng nhập email.");
        return;
      }

      // Email → Resend magic link. Stay on page, show "check inbox" state.
      try {
        await emailMutation.mutateAsync(trimmed);
      } catch {
        // Swallow: success state shown either way (anti-enumeration).
      } finally {
        setDone(true);
      }
    },
    [identifier, emailMutation],
  );

  const pending = emailMutation.isPending;

  return (
    <div
      data-admin-ui=""
      data-theme="congtruong"
      className="relative flex min-h-dvh w-full items-center justify-center overflow-x-hidden bg-muted px-4 py-6 text-fg-primary"
      style={{ paddingTop: "env(safe-area-inset-top)", paddingBottom: "env(safe-area-inset-bottom)" }}
    >
      <AuthCard>
        <AuthBrandMark className="mb-6" />

        {!done ? (
          <>
            <div className="mb-5">
              <h2 className="font-display text-display-md font-semibold leading-tight tracking-tight text-fg-primary">
                Quên mật khẩu?
              </h2>
              <p className="mt-2 text-sm text-fg-tertiary">
                Nhập email đã đăng ký để đặt lại mật khẩu.
              </p>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4" noValidate>
              <div className="space-y-1.5">
                <Label htmlFor="identifier">Email</Label>
                <div className="relative">
                  <Mail01 className="pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2 text-fg-tertiary" aria-hidden="true" />
                  <Input
                    id="identifier"
                    type="email"
                    inputMode="email"
                    placeholder="email@cua-ban.vn"
                    value={identifier}
                    onChange={(e) => {
                      setIdentifier(e.target.value);
                      setFieldError(null);
                    }}
                    className="pl-9 sm:pl-9"
                    required
                    autoComplete="email"
                    aria-invalid={!!fieldError}
                    aria-describedby="identifier-error"
                    disabled={pending}
                  />
                </div>
                {fieldError && (
                  <p id="identifier-error" role="alert" className="text-sm text-fg-error-primary">
                    {fieldError}
                  </p>
                )}
                {/* Channel hint — neutral; does NOT confirm the value exists in the DB. */}
                {identifier.trim() && (
                  <p className="text-xs text-fg-tertiary">
                    {"📧 Nếu email tồn tại, liên kết đặt lại sẽ gửi qua email."}
                  </p>
                )}
              </div>

              <Button type="submit" className="mt-2 w-full" disabled={pending || !identifier.trim()}>
                {pending ? (
                  <>
                    <AuthSpinner className="size-4" /> Đang gửi...
                  </>
                ) : (
                  "Gửi yêu cầu đặt lại"
                )}
              </Button>
            </form>

            <Button
              type="button"
              variant="ghost"
              onClick={() => navigate("/login", { replace: true })}
              className="mt-5 text-xs"
            >
              <ArrowLeft className="size-4" aria-hidden="true" />
              Quay lại đăng nhập
            </Button>
          </>
        ) : (
          <div className="py-2" role="status">
            <div className="mb-5 flex items-start gap-3">
              <FeaturedIcon icon={CheckCircle} color="success" theme="light" size="md" />
              <div>
                <p className="text-sm font-bold text-fg-primary">
                  Nếu email tồn tại trong hệ thống
                </p>
                <p className="mt-1 text-sm leading-5 text-fg-secondary">
                  Bạn sẽ nhận được liên kết đặt lại mật khẩu trong vài phút. Vui lòng kiểm tra hộp thư (kể cả thư rác).
                </p>
              </div>
            </div>
            <Link to="/login" className={cx(buttonVariants({ variant: "outline" }), "w-full")}>
              <ArrowLeft className="size-4" aria-hidden="true" />
              Quay lại đăng nhập
            </Link>
          </div>
        )}
      </AuthCard>
    </div>
  );
};

export default ForgotPassword;
