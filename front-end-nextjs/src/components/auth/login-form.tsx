"use client";

import { useCallback, useEffect, useState } from "react";
import type { FormEvent, ReactNode } from "react";
import {
  AlertCircle,
  CheckCircle2,
  LoaderCircle,
  LogIn,
  Mail,
  RefreshCw,
  UserPlus,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import {
  getAuthCaptcha,
  getAuthConfig,
  loginUser,
  registerUser,
  sendAuthEmailCode,
} from "@/lib/api";
import { Button } from "@/components/ui/button";
import type { AuthConfigPayload, CaptchaPayload } from "@/lib/types";

type AuthMode = "login" | "register";

type RegisterDraft = {
  captchaText: string;
  email: string;
  emailCode: string;
  nickname: string;
  password: string;
  userID: string;
};

const initialRegisterDraft: RegisterDraft = {
  captchaText: "",
  email: "",
  emailCode: "",
  nickname: "",
  password: "",
  userID: "",
};

const inputClassName =
  "h-11 w-full min-w-0 rounded-xl border border-border bg-background px-3 text-sm text-foreground outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/20";

export function LoginForm() {
  const t = useTranslations("login");
  const router = useRouter();
  const [mode, setMode] = useState<AuthMode>("login");
  const [authConfig, setAuthConfig] = useState<AuthConfigPayload | null>(null);
  const [isConfigLoading, setIsConfigLoading] = useState(true);
  const [configFailed, setConfigFailed] = useState(false);
  const [identifier, setIdentifier] = useState("");
  const [loginPassword, setLoginPassword] = useState("");
  const [registerDraft, setRegisterDraft] = useState<RegisterDraft>(initialRegisterDraft);
  const [captcha, setCaptcha] = useState<CaptchaPayload | null>(null);
  const [captchaLoading, setCaptchaLoading] = useState(false);
  const [emailCodeSending, setEmailCodeSending] = useState(false);
  const [emailCodeSent, setEmailCodeSent] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [returnPath] = useState(() =>
    typeof window === "undefined"
      ? "/"
      : normalizeReturnPath(new URLSearchParams(window.location.search).get("next")),
  );
  const emailEnabled = Boolean(authConfig?.emailEnabled);
  const geetestEnabled = Boolean(authConfig?.geetestEnabled);
  const captchaImageSrc = captcha?.captchaSvg
    ? `data:image/svg+xml;utf8,${encodeURIComponent(captcha.captchaSvg)}`
    : null;

  const refreshCaptcha = useCallback(async () => {
    if (geetestEnabled) {
      return;
    }
    setCaptchaLoading(true);
    try {
      setCaptcha(await getAuthCaptcha());
    } catch {
      setFormError(t("errors.captchaLoadFailed"));
    } finally {
      setCaptchaLoading(false);
    }
  }, [geetestEnabled, t]);

  useEffect(() => {
    let mounted = true;

    getAuthConfig()
      .then((config) => {
        if (mounted) {
          setAuthConfig(config);
          setConfigFailed(false);
        }
      })
      .catch(() => {
        if (mounted) {
          setConfigFailed(true);
        }
      })
      .finally(() => {
        if (mounted) {
          setIsConfigLoading(false);
        }
      });

    return () => {
      mounted = false;
    };
  }, []);

  async function handleLoginSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalizedIdentifier = identifier.trim();
    setFormError(null);

    if (!normalizedIdentifier || !loginPassword) {
      setFormError(t("errors.missingLogin"));
      return;
    }

    setSubmitting(true);
    try {
      await loginUser(normalizedIdentifier, loginPassword);
      toast.success(t("success.login"));
      finishAuth(returnPath, router);
    } catch {
      setFormError(t("errors.loginFailed"));
    } finally {
      setSubmitting(false);
    }
  }

  async function handleRegisterSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const draft = {
      ...registerDraft,
      email: registerDraft.email.trim(),
      emailCode: registerDraft.emailCode.trim(),
      nickname: registerDraft.nickname.trim(),
      userID: registerDraft.userID.trim(),
    };
    setFormError(null);

    if (!draft.userID || !draft.nickname || !draft.password) {
      setFormError(t("errors.missingRegister"));
      return;
    }
    if (draft.password.length < 6 || draft.password.length > 20) {
      setFormError(t("errors.passwordLength"));
      return;
    }
    if (!isValidEmail(draft.email)) {
      setFormError(t("errors.emailInvalid"));
      return;
    }
    if (emailEnabled && !draft.emailCode) {
      setFormError(t("errors.emailCodeRequired"));
      return;
    }
    if (geetestEnabled) {
      setFormError(t("errors.geetestUnavailable"));
      return;
    }
    if (!captcha?.captchaId || !draft.captchaText.trim()) {
      setFormError(t("errors.captchaRequired"));
      return;
    }

    setSubmitting(true);
    try {
      await registerUser({
        captchaId: captcha.captchaId,
        captchaText: draft.captchaText.trim(),
        email: draft.email,
        emailCode: emailEnabled ? draft.emailCode : undefined,
        nickname: draft.nickname,
        password: draft.password,
        userID: draft.userID,
      });
      toast.success(t("success.register"));
      finishAuth(returnPath, router);
    } catch {
      setFormError(t("errors.registerFailed"));
      await refreshCaptcha();
    } finally {
      setSubmitting(false);
    }
  }

  async function handleSendEmailCode() {
    const email = registerDraft.email.trim();
    setFormError(null);
    if (!isValidEmail(email)) {
      setFormError(t("errors.emailInvalid"));
      return;
    }

    setEmailCodeSending(true);
    try {
      await sendAuthEmailCode(email);
      setEmailCodeSent(true);
      toast.success(t("emailCodeSent"));
    } catch {
      setFormError(t("errors.emailCodeFailed"));
    } finally {
      setEmailCodeSending(false);
    }
  }

  function updateRegisterDraft<K extends keyof RegisterDraft>(key: K, value: RegisterDraft[K]) {
    setRegisterDraft((current) => ({ ...current, [key]: value }));
  }

  function switchMode(nextMode: AuthMode) {
    setMode(nextMode);
    setFormError(null);
    if (nextMode === "register" && !geetestEnabled && !captcha && !captchaLoading) {
      void refreshCaptcha();
    }
  }

  return (
    <div className="mt-8 space-y-4 text-left">
      <div className="grid grid-cols-2 rounded-full bg-muted p-1">
        <ModeButton active={mode === "login"} label={t("modes.login")} onClick={() => switchMode("login")} />
        <ModeButton active={mode === "register"} label={t("modes.register")} onClick={() => switchMode("register")} />
      </div>

      {formError ? <AlertMessage>{formError}</AlertMessage> : null}

      {configFailed ? <AlertMessage>{t("configLoadFailed")}</AlertMessage> : null}

      {mode === "login" ? (
        <form className="space-y-4" onSubmit={handleLoginSubmit}>
          <Field label={t("fields.identifier")}>
            <input
              autoComplete="username"
              className={inputClassName}
              inputMode="email"
              name="identifier"
              onChange={(event) => setIdentifier(event.target.value)}
              placeholder={t("placeholders.identifier")}
              value={identifier}
            />
          </Field>
          <Field label={t("fields.password")}>
            <input
              autoComplete="current-password"
              className={inputClassName}
              name="password"
              onChange={(event) => setLoginPassword(event.target.value)}
              placeholder={t("placeholders.password")}
              type="password"
              value={loginPassword}
            />
          </Field>
          <Button type="submit" aria-busy={submitting} className="h-11 w-full" disabled={submitting}>
            {submitting ? (
              <LoaderCircle className="size-4 motion-safe:animate-spin" aria-hidden="true" />
            ) : (
              <LogIn className="size-4" aria-hidden="true" />
            )}
            {submitting ? t("actions.loggingIn") : t("actions.login")}
          </Button>
        </form>
      ) : (
        <form className="space-y-4" onSubmit={handleRegisterSubmit}>
          <Field label={t("fields.userID")}>
            <input
              autoComplete="username"
              className={inputClassName}
              maxLength={15}
              name="user_id"
              onChange={(event) => updateRegisterDraft("userID", event.target.value)}
              placeholder={t("placeholders.userID")}
              value={registerDraft.userID}
            />
          </Field>
          <Field label={t("fields.nickname")}>
            <input
              autoComplete="nickname"
              className={inputClassName}
              maxLength={10}
              name="nickname"
              onChange={(event) => updateRegisterDraft("nickname", event.target.value)}
              placeholder={t("placeholders.nickname")}
              value={registerDraft.nickname}
            />
          </Field>
          <Field label={t("fields.email")}>
            <input
              autoComplete="email"
              className={inputClassName}
              inputMode="email"
              name="email"
              onChange={(event) => {
                updateRegisterDraft("email", event.target.value);
                setEmailCodeSent(false);
              }}
              placeholder={t("placeholders.email")}
              type="email"
              value={registerDraft.email}
            />
          </Field>
          {emailEnabled ? (
            <>
              <Field label={t("fields.emailCode")}>
                <div className="flex min-w-0 gap-2">
                  <input
                    autoComplete="one-time-code"
                    className={inputClassName}
                    inputMode="numeric"
                    name="emailCode"
                    onChange={(event) => updateRegisterDraft("emailCode", event.target.value)}
                    placeholder={t("placeholders.emailCode")}
                    value={registerDraft.emailCode}
                  />
                  <Button
                    aria-busy={emailCodeSending}
                    className="h-11 shrink-0 px-3"
                    disabled={emailCodeSending || submitting}
                    onClick={() => void handleSendEmailCode()}
                    type="button"
                    variant="outline"
                  >
                    {emailCodeSending ? (
                      <LoaderCircle className="size-4 motion-safe:animate-spin" aria-hidden="true" />
                    ) : emailCodeSent ? (
                      <CheckCircle2 className="size-4" aria-hidden="true" />
                    ) : (
                      <Mail className="size-4" aria-hidden="true" />
                    )}
                    <span className="hidden sm:inline">
                      {emailCodeSent ? t("actions.resendEmailCode") : t("actions.sendEmailCode")}
                    </span>
                  </Button>
                </div>
              </Field>
            </>
          ) : null}
          <Field label={t("fields.password")}>
            <input
              autoComplete="new-password"
              className={inputClassName}
              name="new-password"
              onChange={(event) => updateRegisterDraft("password", event.target.value)}
              placeholder={t("placeholders.newPassword")}
              type="password"
              value={registerDraft.password}
            />
          </Field>
          {!geetestEnabled ? (
            <Field label={t("fields.captcha")}>
              <div className="grid grid-cols-[minmax(0,1fr)_auto] gap-2">
                <input
                  autoComplete="off"
                  className={inputClassName}
                  name="captchaText"
                  onChange={(event) => updateRegisterDraft("captchaText", event.target.value)}
                  placeholder={t("placeholders.captcha")}
                  value={registerDraft.captchaText}
                />
                <button
                  aria-label={t("actions.refreshCaptcha")}
                  className="flex h-11 w-[7.5rem] shrink-0 items-center justify-center rounded-xl border border-border bg-background text-xs font-semibold text-muted-foreground transition hover:bg-muted"
                  disabled={captchaLoading}
                  onClick={() => void refreshCaptcha()}
                  type="button"
                >
                  {captchaLoading ? (
                    <RefreshCw className="size-4 motion-safe:animate-spin" aria-hidden="true" />
                  ) : captchaImageSrc ? (
                    // eslint-disable-next-line @next/next/no-img-element -- Captcha SVG is generated by the auth API and cannot be optimized by next/image.
                    <img alt={t("captchaAlt")} className="h-[2.1rem] max-w-full" src={captchaImageSrc} />
                  ) : (
                    <RefreshCw className="size-4" aria-hidden="true" />
                  )}
                </button>
              </div>
            </Field>
          ) : (
            <AlertMessage>{t("errors.geetestUnavailable")}</AlertMessage>
          )}
          <Button type="submit" aria-busy={submitting} className="h-11 w-full" disabled={submitting}>
            {submitting ? (
              <LoaderCircle className="size-4 motion-safe:animate-spin" aria-hidden="true" />
            ) : (
              <UserPlus className="size-4" aria-hidden="true" />
            )}
            {submitting ? t("actions.registering") : t("actions.register")}
          </Button>
        </form>
      )}

      {isConfigLoading ? (
        <div
          aria-busy="true"
          aria-live="polite"
          className="rounded-xl border border-border bg-muted px-3 py-3 text-center text-sm text-muted-foreground"
        >
          <span className="inline-flex items-center justify-center gap-2">
            <RefreshCw className="size-3.5 animate-spin" aria-hidden="true" />
            {t("loadingConfig")}
          </span>
        </div>
      ) : null}
    </div>
  );
}

function ModeButton({
  active,
  label,
  onClick,
}: {
  active: boolean;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      aria-pressed={active}
      className={
        active
          ? "h-9 rounded-full bg-background text-sm font-semibold text-foreground shadow-sm"
          : "h-9 rounded-full text-sm font-semibold text-muted-foreground transition hover:text-foreground"
      }
      onClick={onClick}
      type="button"
    >
      {label}
    </button>
  );
}

function Field({ children, label }: { children: ReactNode; label: string }) {
  return (
    <label className="block space-y-1.5">
      <span className="text-xs font-semibold text-muted-foreground">{label}</span>
      {children}
    </label>
  );
}

function AlertMessage({ children }: { children: ReactNode }) {
  return (
    <div
      role="alert"
      className="flex items-start gap-2 rounded-xl border border-destructive/30 bg-destructive/10 px-3 py-3 text-sm text-destructive"
    >
      <AlertCircle className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
      <p className="min-w-0">{children}</p>
    </div>
  );
}

function normalizeReturnPath(value: string | null) {
  const normalized = value?.trim() ?? "";
  if (!normalized || !normalized.startsWith("/") || normalized.startsWith("//")) {
    return "/";
  }
  if (normalized === "/login" || normalized.startsWith("/login?")) {
    return "/";
  }
  return normalized;
}

function isValidEmail(value: string) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/u.test(value.trim());
}

function finishAuth(returnPath: string, router: ReturnType<typeof useRouter>) {
  router.replace(returnPath);
  router.refresh();
}
