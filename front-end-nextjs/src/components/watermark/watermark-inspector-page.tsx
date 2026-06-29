"use client";

import Link from "next/link";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import { useLocale, useTranslations } from "next-intl";
import {
  AlertCircle,
  ArrowLeft,
  CheckCircle2,
  FileSearch,
  ImageUp,
  Loader2,
  type LucideIcon,
  ShieldCheck,
} from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  ApiError,
  extractHiddenWatermark,
  getStoredAccessToken,
} from "@/lib/api";
import type { WatermarkExtractionProgress } from "@/lib/api";
import type { HiddenWatermarkResult } from "@/lib/types";
import { cn } from "@/lib/utils";

const acceptedImageTypes = ["image/jpeg", "image/png", "image/webp", "image/avif"];

export function WatermarkInspectorPage() {
  const t = useTranslations("watermarkInspector");
  const locale = useLocale();
  const router = useRouter();
  const [authChecked, setAuthChecked] = useState(false);
  const [authToken, setAuthToken] = useState<string | null>(null);
  const [file, setFile] = useState<File | null>(null);
  const [referenceFile, setReferenceFile] = useState<File | null>(null);
  const [result, setResult] = useState<HiddenWatermarkResult | null>(null);
  const [isInspecting, setIsInspecting] = useState(false);
  const [extractionProgress, setExtractionProgress] = useState<WatermarkExtractionProgress>({
    stage: "uploading",
    percent: 0,
    elapsedMs: 0,
  });
  const previewUrl = useMemo(() => (file ? URL.createObjectURL(file) : ""), [file]);

  useEffect(() => {
    queueMicrotask(() => {
      setAuthToken(getStoredAccessToken());
      setAuthChecked(true);
    });
  }, []);

  useEffect(() => {
    if (!previewUrl) return;
    return () => URL.revokeObjectURL(previewUrl);
  }, [previewUrl]);

  const resultEntries = useMemo(
    () =>
      result
        ? [
            [t("fields.found"), formatBoolean(result.found, t)],
            [t("fields.valid"), formatBoolean(result.valid, t)],
            [t("fields.version"), formatValue(result.version)],
            [t("fields.traceToken"), formatValue(result.traceToken)],
            [t("fields.traceType"), formatValue(result.traceType)],
            [t("fields.traceResolved"), formatBoolean(result.traceResolved, t)],
            [t("fields.payloadBytes"), formatValue(result.payloadBytes)],
            [t("fields.payloadBits"), formatValue(result.payloadBits)],
            [t("fields.payloadFormat"), formatValue(result.payloadFormat)],
            [t("fields.watermarkEngine"), formatValue(result.watermarkEngine)],
            [t("fields.uid"), formatValue(result.uid)],
            [t("fields.userId"), formatValue(result.userId)],
            [t("fields.username"), formatValue(result.username)],
            [t("fields.uploadedAt"), formatValue(result.uploadedAt)],
            [t("fields.sourceHash"), formatValue(result.sourceHash)],
            [t("fields.customText"), formatValue(result.customText)],
            [t("fields.postId"), formatValue(result.postId)],
            [t("fields.imageId"), formatValue(result.imageId)],
            [t("fields.jobId"), formatValue(result.jobId)],
            [t("fields.includedFields"), formatValue(result.includedFields?.join(", "))],
          ]
        : [],
    [result, t],
  );

  const handleBack = useCallback(() => {
    if (window.history.length > 1) {
      router.back();
      return;
    }
    router.push("/");
  }, [router]);

  function chooseFile(nextFile: File | null) {
    setResult(null);
    if (!nextFile) {
      setFile(null);
      return;
    }
    if (!acceptedImageTypes.includes(nextFile.type)) {
      toast.error(t("errors.unsupportedType"));
      setFile(null);
      return;
    }
    setFile(nextFile);
  }

  function chooseReferenceFile(nextFile: File | null) {
    setResult(null);
    if (!nextFile) {
      setReferenceFile(null);
      return;
    }
    if (!acceptedImageTypes.includes(nextFile.type)) {
      toast.error(t("errors.unsupportedType"));
      setReferenceFile(null);
      return;
    }
    setReferenceFile(nextFile);
  }

  async function inspect() {
    if (!file) {
      toast.error(t("errors.fileRequired"));
      return;
    }
    setIsInspecting(true);
    setExtractionProgress({ stage: "uploading", percent: 0, elapsedMs: 0 });
    try {
      const nextResult = await extractHiddenWatermark(file, referenceFile, {
        onProgress: (progress) => {
          setExtractionProgress((current) => ({
            ...progress,
            percent: Math.max(current.percent, progress.percent),
          }));
        },
      });
      setResult(nextResult);
      toast.success(t("completed"));
    } catch (error) {
      setResult(null);
      toast.error(errorMessage(error, t));
    } finally {
      setIsInspecting(false);
    }
  }

  if (!authChecked) {
    return (
      <Shell title={t("title")} onBack={handleBack}>
        <CenteredState icon={Loader2} label={t("checkingAuth")} spinning />
      </Shell>
    );
  }

  if (!authToken) {
    return (
      <Shell title={t("title")} onBack={handleBack}>
        <section className="mx-auto flex min-h-[420px] w-full max-w-[420px] flex-col items-center justify-center px-5 text-center">
          <span className="flex size-14 items-center justify-center rounded-full bg-emerald-500/10 text-emerald-600">
            <ShieldCheck className="size-7" />
          </span>
          <h1 className="mt-5 text-xl font-black text-[#191b20]">{t("loginRequired.title")}</h1>
          <p className="mt-2 text-sm leading-6 text-[#666c76]">{t("loginRequired.description")}</p>
          <div className="mt-6 grid w-full grid-cols-2 gap-2">
            <Button asChild className="h-11 rounded-lg bg-emerald-600 text-white hover:bg-emerald-700">
              <Link href="/login">{t("loginRequired.login")}</Link>
            </Button>
            <Button asChild variant="outline" className="h-11 rounded-lg">
              <Link href="/">{t("loginRequired.home")}</Link>
            </Button>
          </div>
        </section>
      </Shell>
    );
  }

  return (
    <Shell title={t("title")} onBack={handleBack}>
      <section className="mx-auto grid w-full max-w-[1040px] gap-4 px-4 pb-[calc(24px+env(safe-area-inset-bottom))] pt-4 md:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)] md:px-6 lg:pt-8">
        <div className="min-w-0 rounded-lg border border-black/[0.07] bg-white p-4 shadow-[0_14px_34px_rgba(30,41,59,0.07)] md:p-5">
          <div className="flex min-w-0 items-start gap-3">
            <span className="flex size-11 shrink-0 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-600">
              <ImageUp className="size-5" />
            </span>
            <div className="min-w-0">
              <h1 className="text-lg font-black text-[#191b20] md:text-xl">{t("upload.title")}</h1>
              <p className="mt-1 text-sm leading-6 text-[#68707d]">{t("upload.description")}</p>
            </div>
          </div>

          <label
            htmlFor="watermark-inspector-file"
            className="mt-5 flex min-h-[180px] cursor-pointer flex-col items-center justify-center rounded-lg border border-dashed border-emerald-500/35 bg-emerald-50/60 px-4 py-6 text-center transition-colors hover:bg-emerald-50"
          >
            <FileSearch className="size-8 text-emerald-600" />
            <span className="mt-3 text-sm font-black text-[#23262d]">
              {file ? file.name : t("upload.choose")}
            </span>
            <span className="mt-2 text-xs leading-5 text-[#68707d]">
              {file ? formatFileSize(file.size, locale) : t("upload.accept")}
            </span>
          </label>
          <input
            id="watermark-inspector-file"
            type="file"
            accept={acceptedImageTypes.join(",")}
            className="sr-only"
            onChange={(event) => chooseFile(event.target.files?.[0] ?? null)}
          />

          <label
            htmlFor="watermark-inspector-reference-file"
            className="mt-4 block text-sm font-black text-[#23262d]"
          >
            {t("upload.referenceLabel")}
          </label>
          <input
            id="watermark-inspector-reference-file"
            type="file"
            accept={acceptedImageTypes.join(",")}
            className="mt-2 block w-full min-w-0 text-sm text-[#68707d] file:mr-3 file:rounded-lg file:border-0 file:bg-emerald-50 file:px-3 file:py-2 file:font-bold file:text-emerald-700"
            onChange={(event) => chooseReferenceFile(event.target.files?.[0] ?? null)}
          />
          <p className="mt-2 text-xs leading-5 text-[#747b86]">
            {referenceFile
              ? t("upload.referenceSelected", { name: referenceFile.name })
              : t("upload.referenceDescription")}
          </p>

          {previewUrl ? (
            <div className="mt-4 overflow-hidden rounded-lg border border-black/[0.07] bg-[#f5f7f8]">
              <Image
                src={previewUrl}
                alt={t("upload.previewAlt")}
                width={960}
                height={640}
                unoptimized
                className="max-h-[320px] w-full object-contain"
              />
            </div>
          ) : null}

          <Button
            type="button"
            disabled={!file || isInspecting}
            onClick={() => void inspect()}
            className="mt-4 h-11 w-full rounded-lg bg-emerald-600 text-sm font-black text-white hover:bg-emerald-700"
          >
            {isInspecting ? <Loader2 className="size-4 animate-spin" /> : <FileSearch className="size-4" />}
            <span>{isInspecting ? t("inspecting") : t("inspect")}</span>
          </Button>

          {isInspecting ? (
            <div className="mt-3 rounded-lg border border-emerald-500/15 bg-emerald-50/70 p-3">
              <div className="flex items-center justify-between gap-3 text-xs font-bold text-emerald-800">
                <span>{t(`progress.${extractionProgress.stage}`)}</span>
                <span>{t("progress.elapsed", { seconds: Math.floor(extractionProgress.elapsedMs / 1000) })}</span>
              </div>
              <div className="mt-2 h-2 overflow-hidden rounded-full bg-emerald-900/10">
                <div
                  className="h-full rounded-full bg-emerald-600 transition-[width] duration-300"
                  style={{ width: `${extractionProgress.percent}%` }}
                />
              </div>
              <p className="mt-2 text-xs leading-5 text-emerald-900/60">
                {extractionProgress.total
                  ? t("progress.units", {
                      completed: extractionProgress.completed ?? 0,
                      total: extractionProgress.total,
                    })
                  : extractionProgress.heartbeat
                    ? t(
                        extractionProgress.source === "engine"
                          ? "progress.heartbeatEngine"
                          : "progress.heartbeatGateway",
                      )
                    : t("progress.live")}
              </p>
            </div>
          ) : null}

          <p className="mt-3 text-xs leading-5 text-[#747b86]">{t("upload.notice")}</p>
        </div>

        <div className="min-w-0 rounded-lg border border-black/[0.07] bg-white p-4 shadow-[0_14px_34px_rgba(30,41,59,0.07)] md:p-5">
          <div className="flex min-w-0 items-start gap-3">
            <span
              className={cn(
                "flex size-11 shrink-0 items-center justify-center rounded-lg",
                result?.found ? "bg-emerald-500/10 text-emerald-600" : "bg-[#eef1f3] text-[#7a828c]",
              )}
            >
              {result?.found ? <CheckCircle2 className="size-5" /> : <AlertCircle className="size-5" />}
            </span>
            <div className="min-w-0">
              <h2 className="text-lg font-black text-[#191b20] md:text-xl">{t("result.title")}</h2>
              <p className="mt-1 text-sm leading-6 text-[#68707d]">
                {result ? (result.found ? t("result.found") : t("result.notFound")) : t("result.empty")}
              </p>
            </div>
          </div>

          {result ? (
            <dl className="mt-5 grid gap-2 sm:grid-cols-2">
              {resultEntries.map(([label, value]) => (
                <div key={label} className="min-w-0 rounded-lg bg-[#f6f8f8] px-3 py-2">
                  <dt className="truncate text-xs font-bold text-[#717985]">{label}</dt>
                  <dd className="mt-1 break-words text-sm font-semibold text-[#23262d]">{value}</dd>
                </div>
              ))}
            </dl>
          ) : (
            <CenteredState icon={ShieldCheck} label={t("result.placeholder")} />
          )}
        </div>
      </section>
    </Shell>
  );
}

function Shell({
  children,
  onBack,
  title,
}: {
  children: ReactNode;
  onBack: () => void;
  title: string;
}) {
  const t = useTranslations("watermarkInspector");

  return (
    <main className="min-h-dvh bg-[#f5f7f8] text-[#22252b]">
      <header className="sticky top-0 z-30 border-b border-black/[0.06] bg-white/94 px-3 pt-[env(safe-area-inset-top)] backdrop-blur">
        <div className="mx-auto grid h-14 w-full max-w-[1040px] grid-cols-[44px_minmax(0,1fr)_44px] items-center">
          <button
            type="button"
            aria-label={t("back")}
            onClick={onBack}
            className="flex size-10 items-center justify-center rounded-lg text-[#333842] transition-colors active:bg-black/[0.05]"
          >
            <ArrowLeft className="size-5" />
          </button>
          <h1 className="truncate text-center text-base font-black text-[#191b20] md:text-lg">{title}</h1>
          <span aria-hidden="true" />
        </div>
      </header>
      {children}
    </main>
  );
}

function CenteredState({
  icon: Icon,
  label,
  spinning,
}: {
  icon: LucideIcon;
  label: string;
  spinning?: boolean;
}) {
  return (
    <div className="flex min-h-[260px] flex-col items-center justify-center rounded-lg border border-black/[0.06] bg-white/70 px-4 text-center text-sm font-bold text-[#747b86]">
      <Icon className={cn("mb-3 size-6 text-[#9aa2ad]", spinning && "animate-spin")} />
      {label}
    </div>
  );
}

function formatBoolean(value: unknown, t: ReturnType<typeof useTranslations>) {
  return value ? t("yes") : t("no");
}

function formatValue(value: unknown) {
  if (value === undefined || value === null || value === "") {
    return "-";
  }
  if (typeof value === "boolean") {
    return value ? "true" : "false";
  }
  return String(value);
}

function formatFileSize(value: number, locale: string) {
  if (!Number.isFinite(value) || value <= 0) {
    return "0 B";
  }
  const units = ["B", "KB", "MB", "GB"];
  let size = value;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }
  return `${new Intl.NumberFormat(locale, { maximumFractionDigits: unitIndex === 0 ? 0 : 1 }).format(size)} ${units[unitIndex]}`;
}

function errorMessage(error: unknown, t: ReturnType<typeof useTranslations>) {
  if (error instanceof ApiError) {
    if (error.status === 403) return t("errors.forbidden");
    if (error.message === "error.image_file_required") return t("errors.fileRequired");
    if (error.message === "error.image_file_too_large") return t("errors.fileTooLarge");
    if (error.message === "error.image_invalid") return t("errors.invalidImage");
    if (error.message === "error.screenshot_recovery_failed") return t("errors.recoveryFailed");
    if (error.message === "error.hidden_watermark_remote_timeout" || error.status === 504 || error.status === 554) {
      return t("errors.remoteTimeout");
    }
    if (
      error.message === "error.watermark_stream_incomplete" ||
      error.message === "error.watermark_stream_disconnected"
    ) {
      return t("errors.remoteDisconnected");
    }
  }
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return t("errors.failed");
}
