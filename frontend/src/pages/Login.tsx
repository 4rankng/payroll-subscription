import { useState, useEffect, useCallback, useRef } from "react";
import { useNavigate, Link } from "react-router-dom";
import { AlertCircle, Eye, EyeOff, Lock01, ShieldTick, User01 } from "@untitledui/icons";
import { authManager } from "@/lib/auth";
import { useAuth } from "@/contexts";
import { useLogin } from "@/hooks/api/useAuth";
import type { ApiError } from "@/services/api/client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { AuthSpinner } from "@/components/auth/AuthSpinner";
import { AuthBrandMark } from "@/components/auth/AuthBrandMark";

const Login = () => {
  const [emailOrUsername, setEmailOrUsername] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [loginSubmitting, setLoginSubmitting] = useState(false);
  const loginMutation = useLogin();
  const loginError = loginMutation.error;
  const resetLogin = loginMutation.reset;
  const navigate = useNavigate();
  const { isAuthenticated, isLoading } = useAuth();
  const loginSubmitInFlightRef = useRef(false);

  // Redirect authenticated users away from /login
  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      const role = authManager.getUserRole();
      if (role === "admin") navigate("/admin", { replace: true });
      else if (role === "quanly") navigate("/partner/dashboard", { replace: true });
      else if (role === "ketoan" || role === "giamdoc") navigate("/partner/overview", { replace: true });
      else if (role === "nhanvien") navigate("/employee", { replace: true });
      else navigate("/", { replace: true });
    }
  }, [isLoading, isAuthenticated, navigate]);

  useEffect(() => {
    if (!authManager.hasSession()) {
      authManager.clearSession();
    }
  }, []);

  useEffect(() => {
    if (loginError) resetLogin();
  }, [emailOrUsername, password]); // eslint-disable-line react-hooks/exhaustive-deps

  const isDisabled = loginSubmitting || loginMutation.isPending;

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      // Guard the entire submit window synchronously so a double click/Enter
      // cannot create two login requests.
      if (loginSubmitInFlightRef.current) return;
      loginSubmitInFlightRef.current = true;
      setLoginSubmitting(true);
      try {
        await loginMutation.mutateAsync({
          username: emailOrUsername,
          password,
        });
      } catch {
        // The mutation owns error presentation; this only prevents an unhandled
        // rejection from mutateAsync while keeping the single-flight guard.
      } finally {
        loginSubmitInFlightRef.current = false;
        setLoginSubmitting(false);
      }
    },
    [emailOrUsername, password, loginMutation]
  );

  const togglePassword = useCallback(() => setShowPassword((v) => !v), []);

  if (isLoading) {
    return (
      <div
        data-admin-ui=""
        data-theme="congtruong"
        className="flex min-h-dvh items-center justify-center bg-muted text-fg-primary"
      >
        <AuthSpinner className="size-7 text-brand-solid" aria-label="Đang tải" role="img" />
      </div>
    );
  }

  if (isAuthenticated) {
    return null;
  }

  return (
    <div
      id="main-content"
      data-admin-ui=""
      data-login-ui=""
      data-theme="congtruong"
      className="relative min-h-dvh w-full overflow-x-hidden bg-muted text-fg-primary"
      style={{ paddingTop: "env(safe-area-inset-top)", paddingBottom: "env(safe-area-inset-bottom)" }}
    >
      <main className="relative mx-auto grid min-h-dvh w-full max-w-[1560px] lg:grid-cols-[minmax(0,1.08fr)_minmax(440px,0.92fr)] lg:gap-3 lg:p-3 xl:gap-4 xl:p-5">
        <section
          aria-labelledby="login-brand-heading"
          className="relative hidden min-h-[calc(100dvh-2.5rem)] overflow-hidden rounded-[2rem] border border-input bg-card shadow-xs lg:flex"
        >
          <img
            src="/login-employee-wallet-app-hero.webp"
            alt="Ứng dụng lương tuần bên cạnh ví và lịch trả lương"
            className="absolute bottom-0 right-0 h-full w-auto max-w-none object-contain object-bottom-right"
          />
          <div className="absolute inset-0 bg-gradient-to-r from-card via-card/95 via-[46%] to-transparent to-[78%]" />
          <div className="absolute inset-0 bg-gradient-to-b from-card/55 via-transparent via-[46%] to-card/10" />

          <div className="relative z-10 flex h-full w-full max-w-none items-start justify-start px-8 py-[clamp(3.5rem,8vh,7rem)] text-left xl:px-12 2xl:px-14">
            <div className="max-w-[540px]">
              <p className="mb-4 text-sm font-semibold text-brand-secondary">
                Lương về đúng nhịp
              </p>
              <h1 id="login-brand-heading" className="font-display text-[clamp(2.65rem,3.8vw,4.1rem)] font-black leading-[0.96] tracking-[-0.05em] text-fg-primary">
                Ứng lương khi cần.
                <span className="mt-2 block text-brand-secondary">Trả lương mỗi tuần.</span>
              </h1>
              <p className="mt-6 max-w-[470px] text-base font-medium leading-7 text-fg-secondary xl:text-lg">
                Dòng tiền linh hoạt cho người lao động. Một chu kỳ lương gọn gàng, dễ kiểm soát cho doanh nghiệp.
              </p>
            </div>
          </div>
        </section>

        <section className="relative flex min-h-dvh flex-col bg-card lg:min-h-[calc(100dvh-2.5rem)] lg:rounded-[2rem] lg:border lg:border-input">
          <div className="relative h-44 min-h-44 overflow-hidden border-b border-input bg-card sm:h-52 sm:min-h-52 lg:hidden">
            <img
              src="/login-employee-wallet-app-hero.webp"
              alt="Ứng dụng lương tuần bên cạnh ví và lịch trả lương"
              className="absolute inset-y-0 right-0 h-full w-[56%] object-cover object-[54%_68%]"
            />
            <div className="absolute inset-y-0 left-0 w-[62%] bg-gradient-to-r from-card via-card/95 to-transparent" />
            <div className="relative z-10 flex h-full w-full max-w-none items-start justify-start px-5 py-5 sm:px-8 sm:py-7">
              <div className="max-w-[56%]">
                <p className="text-sm font-semibold text-brand-secondary">Lương về đúng nhịp</p>
                <p className="mt-1.5 max-w-[280px] font-display text-2xl font-black leading-[1.05] tracking-[-0.035em] sm:text-3xl">
                  Ứng lương khi cần.<br /><span className="text-brand-secondary">Trả lương mỗi tuần.</span>
                </p>
                <p className="mt-2.5 text-xs font-medium leading-[1.45] text-fg-secondary sm:text-xs">
                  Dòng tiền linh hoạt cho người lao động. Một chu kỳ lương gọn gàng, dễ kiểm soát cho doanh nghiệp.
                </p>
              </div>
            </div>
          </div>

          <div className="flex flex-1 items-center justify-center px-4 py-6 sm:px-8 sm:py-8 lg:px-10 xl:px-16">
            <div className="w-full max-w-[470px]">
              <AuthBrandMark className="mb-7" />

              <div className="mb-7">
                <h2 className="font-display text-display-lg font-semibold leading-tight tracking-tight text-fg-primary">Chào mừng trở lại</h2>
              </div>

              {loginMutation.error && (
                <div
                  role="alert"
                  data-testid="error-message"
                  className="mb-5 flex items-start gap-2.5 rounded-lg border border-utility-error-300 bg-utility-error-50 p-3 text-sm motion-safe:animate-fade-in-up"
                >
                  <AlertCircle className="size-5 shrink-0 text-utility-error-700" aria-hidden="true" />
                  <span className="font-semibold leading-5 text-fg-error-primary">
                    {(((loginMutation.error as unknown) as ApiError)?.http_status === 429
                      ? (((loginMutation.error as unknown) as ApiError)?.message || "Quá nhiều lần đăng nhập. Vui lòng thử lại sau ít phút.")
                      : "Thông tin đăng nhập không hợp lệ. Vui lòng thử lại.")}
                  </span>
                </div>
              )}

              <form onSubmit={handleSubmit} className="space-y-4" noValidate>
                <fieldset className="space-y-4">
                  <legend className="sr-only">Thông tin đăng nhập</legend>

                  <div className="space-y-1.5">
                    <Label htmlFor="emailOrUsername">Tên đăng nhập</Label>
                    <div className="relative">
                      <User01 className="pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2 text-fg-tertiary" aria-hidden="true" />
                      <Input
                        id="emailOrUsername"
                        type="text"
                        placeholder="CCCD, số điện thoại hoặc tên đăng nhập"
                        value={emailOrUsername}
                        onChange={(e) => setEmailOrUsername(e.target.value)}
                        className="pl-9 sm:pl-9"
                        required
                        autoComplete="username"
                        autoCapitalize="none"
                        disabled={isDisabled}
                      />
                    </div>
                  </div>

                  <div className="space-y-1.5">
                    <Label htmlFor="password">Mật khẩu</Label>
                    <div className="relative">
                      <Lock01 className="pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2 text-fg-tertiary" aria-hidden="true" />
                      <Input
                        id="password"
                        type={showPassword ? "text" : "password"}
                        autoCapitalize="none"
                        autoCorrect="off"
                        spellCheck={false}
                        placeholder="Nhập mật khẩu của bạn"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        className="pr-12 pl-9 sm:pr-10 sm:pl-9"
                        required
                        autoComplete="current-password"
                        disabled={isDisabled}
                      />
                      <button
                        type="button"
                        onClick={togglePassword}
                        aria-label={showPassword ? "Ẩn mật khẩu" : "Hiện mật khẩu"}
                        className="absolute inset-y-0 right-0 flex h-11 w-11 items-center justify-center rounded-lg text-fg-tertiary outline-brand transition duration-100 ease-linear hover:text-fg-tertiary focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 sm:h-9 sm:w-9"
                      >
                        {showPassword ? <EyeOff className="size-5" aria-hidden="true" /> : <Eye className="size-5" aria-hidden="true" />}
                      </button>
                    </div>
                  </div>

                  <div className="flex justify-end pb-3 pt-3">
                    <Link to="/forgot-password" className="tap-target-44 inline-flex min-h-6 items-center rounded-sm text-xs font-semibold text-brand-secondary underline-offset-2 outline-brand transition duration-100 ease-linear hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2">
                      Quên mật khẩu?
                    </Link>
                  </div>
                </fieldset>

                <Button type="submit" className="w-full" disabled={isDisabled}>
                  {loginMutation.isPending ? (
                    <>
                      <AuthSpinner className="size-4" />
                      Đang đăng nhập...
                    </>
                  ) : (
                    "Đăng nhập"
                  )}
                </Button>
              </form>

              <div className="mt-6 flex items-center justify-center gap-2 text-xs text-fg-tertiary">
                <ShieldTick className="size-4 text-fg-success-primary" aria-hidden="true" />
                <span>Phiên đăng nhập được bảo vệ và mã hóa</span>
              </div>
            </div>
          </div>

          <footer className="px-5 pb-6 text-center text-xs text-fg-tertiary lg:px-10">
            <p>© {new Date().getFullYear()} TingTing · Ứng lương nhanh · Trả lương tuần</p>
            <div className="mt-2 flex items-center justify-center gap-3">
              <span>Điều khoản</span>
              <span aria-hidden="true">·</span>
              <span>Quyền riêng tư</span>
              <span aria-hidden="true">·</span>
              <span>Hỗ trợ</span>
            </div>
          </footer>
        </section>
      </main>
    </div>
  );
};

export default Login;
