"use client";
import type {
  FormEvent
} from "react";
import {
  useEffect,
  useState
} from "react";
import {
  useRouter
} from "next/navigation";
import {
  Loader2,
  Lock,
  ShieldCheck
} from "lucide-react";
import {
  useTranslations
} from "next-intl";
import {
  toast
} from "sonner";
import {
  Button
} from "@/components/ui/button";
import {
  ApiError,
  getCurrentAdmin,
  getStoredAdminAccessToken,
  loginAdmin,
} from "@/lib/api";
import { AdminPublicShell } from "./admin-public-shell";

export function AdminPage({
  backendApiEntryPath = "/backend-api",
}: {
  backendApiEntryPath?: string;
}) {
  void backendApiEntryPath;
  const adminT = useTranslations("adminPortal");
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [loginError, setLoginError] = useState<string | null>(null);
  const [isLoggingIn, setIsLoggingIn] = useState(false);
  const [isBooting, setIsBooting] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function bootstrapAdminSession() {
      const storedToken = getStoredAdminAccessToken();
      if (storedToken) {
        try {
          await getCurrentAdmin(storedToken);
          if (!cancelled) {
            router.replace("/admin/console");
          }
          return;
        } catch {
          // getCurrentAdmin clears the stored admin session only for 401 responses.
        }
      }

      if (!cancelled) {
        setIsBooting(false);
      }
    }

    void bootstrapAdminSession();

    return () => {
      cancelled = true;
    };
  }, [router]);

  async function handleLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoginError(null);
    setIsLoggingIn(true);
    try {
      await loginAdmin(username.trim(), password);
      setPassword("");
      toast.success(adminT("login.success"));
      router.replace("/admin/console");
    } catch (error) {
      const message = adminLoginErrorMessage(error, {
        failed: adminT("login.errors.failed"),
        invalidCredentials: adminT("login.errors.invalidCredentials"),
        locked: (minutes) => adminT("login.errors.locked", { minutes }),
        missingParams: adminT("login.errors.missingParams"),
      });
      setLoginError(message);
      toast.error(message);
    } finally {
      setIsLoggingIn(false);
    }
  }

  if (isBooting) {
    return (
      <AdminPublicShell>
        <AdminPanelLoading label={adminT("loading")} />
      </AdminPublicShell>
    );
  }

  return (
    <AdminPublicShell>
      <section className="mx-auto flex min-h-[72dvh] w-full max-w-[440px] items-center px-4">
          <form
            onSubmit={handleLogin}
            className="w-full rounded-lg border border-black/[0.06] bg-white/90 p-5 shadow-[0_20px_60px_rgba(20,20,30,0.10)] backdrop-blur-xl sm:p-6"
          >
            <div className="mb-6 flex items-center gap-3">
              <span className="flex size-11 items-center justify-center rounded-lg bg-[#1d4ed8] text-white shadow-lg shadow-[#1d4ed8]/20">
                <Lock className="size-5" />
              </span>
              <div className="min-w-0">
                <h1 className="text-lg font-semibold text-[#17171d]">{adminT("login.title")}</h1>
                <p className="text-xs text-[#7a7d87]">{adminT("login.subtitle")}</p>
              </div>
            </div>
            <label className="mb-3 block">
              <span className="mb-1.5 block text-xs font-semibold text-[#5f636d]">{adminT("login.username")}</span>
              <input
                value={username}
                onChange={(event) => setUsername(event.target.value)}
                className="h-11 w-full rounded-lg border border-black/[0.08] bg-white px-3 text-sm text-[#17171d] outline-none transition focus:border-[#1d4ed8] focus:ring-4 focus:ring-[#1d4ed8]/10"
                placeholder={adminT("login.usernamePlaceholder")}
                autoComplete="username"
                required
              />
            </label>
            <label className="mb-4 block">
              <span className="mb-1.5 block text-xs font-semibold text-[#5f636d]">{adminT("login.password")}</span>
              <input
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                className="h-11 w-full rounded-lg border border-black/[0.08] bg-white px-3 text-sm text-[#17171d] outline-none transition focus:border-[#1d4ed8] focus:ring-4 focus:ring-[#1d4ed8]/10"
                placeholder="******"
                type="password"
                autoComplete="current-password"
                required
              />
            </label>
            {loginError ? (
              <p className="mb-4 rounded-lg border border-[#dc2626]/20 bg-[#fef2f2] px-3 py-2 text-sm text-[#b91c1c]">
                {loginError}
              </p>
            ) : null}
            <Button type="submit" disabled={isLoggingIn} className="h-11 w-full rounded-lg bg-[#1d4ed8] hover:bg-[#1e40af]">
              <span className="inline-flex size-4 items-center justify-center" aria-hidden="true">
                {isLoggingIn ? <Loader2 className="size-4 animate-spin" /> : <ShieldCheck className="size-4" />}
              </span>
              <span>{adminT("login.submit")}</span>
            </Button>
          </form>
      </section>
    </AdminPublicShell>
  );
}

function AdminPanelLoading({ label }: { label?: string }) {
  return (
    <div className="flex min-h-48 items-center justify-center rounded-xl border border-black/[0.06] bg-white/80">
      <Loader2 className="size-5 animate-spin text-[#1d4ed8]" aria-hidden="true" />
      {label ? <span className="ml-2 text-sm text-[#6b7280]">{label}</span> : null}
    </div>
  );
}

function adminLoginErrorMessage(
  error: unknown,
  labels: {
    failed: string;
    invalidCredentials: string;
    locked: (minutes: number) => string;
    missingParams: string;
  },
) {
  if (!(error instanceof ApiError)) {
    return error instanceof Error ? error.message : labels.failed;
  }
  if (error.message === "error.admin_invalid_credentials") {
    return labels.invalidCredentials;
  }
  if (error.message === "error.missing_params") {
    return labels.missingParams;
  }
  if (error.message === "error.admin_login_locked") {
    const details = error.details as { data?: { retry_after_seconds?: unknown } } | null;
    const seconds = Number(details?.data?.retry_after_seconds);
    const minutes = Number.isFinite(seconds) ? Math.max(1, Math.ceil(seconds / 60)) : 30;
    return labels.locked(minutes);
  }
  return error.message || labels.failed;
}
