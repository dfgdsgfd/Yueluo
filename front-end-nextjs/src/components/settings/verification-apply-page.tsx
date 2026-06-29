"use client";

import Image from "next/image";
import Link from "next/link";
import { useEffect, useMemo, useRef, useState, type ChangeEvent, type FormEvent } from "react";
import {
  ArrowLeft,
  BadgeCheck,
  CheckCircle2,
  Clock3,
  FileText,
  ImagePlus,
  Loader2,
  RotateCcw,
  Send,
  ShieldCheck,
  Trash2,
  UserRound,
  XCircle,
} from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  createVerificationApplication,
  getStoredAccessToken,
  getVerificationApplications,
  uploadImages,
} from "@/lib/api";
import type { VerificationApplication, VerificationType } from "@/lib/types";
import { cn } from "@/lib/utils";

type VerificationDraft = {
  content: string;
  imageUrls: string[];
  type: VerificationType;
  verifiedName: string;
};

const maxVerificationImages = 9;

const verificationTypeOptions = [
  { value: 1, label: "个人认证", icon: UserRound },
  { value: 2, label: "官方认证", icon: BadgeCheck },
] as const satisfies ReadonlyArray<{
  value: VerificationType;
  label: string;
  icon: typeof UserRound;
}>;

const emptyDraft: VerificationDraft = {
  content: "",
  imageUrls: [],
  type: 1,
  verifiedName: "",
};

export function VerificationApplyPage() {
  const imageInputRef = useRef<HTMLInputElement | null>(null);
  const [authChecked, setAuthChecked] = useState(false);
  const [authToken, setAuthToken] = useState<string | null>(null);
  const [applications, setApplications] = useState<VerificationApplication[]>([]);
  const [draft, setDraft] = useState<VerificationDraft>(emptyDraft);
  const [isLoading, setIsLoading] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isUploadingImages, setIsUploadingImages] = useState(false);

  const latestApplication = applications[0] ?? null;
  const approvedApplication = useMemo(
    () => applications.find((item) => normalizeStatus(item.status) === 1) ?? null,
    [applications],
  );
  const blockingApplication = useMemo(
    () => applications.find((item) => normalizeStatus(item.status) === 0 || normalizeStatus(item.status) === 1),
    [applications],
  );
  const canSubmit =
    Boolean(authToken) &&
    !blockingApplication &&
    draft.verifiedName.trim().length > 0 &&
    draft.content.trim().length > 0 &&
    !isUploadingImages &&
    !isSubmitting;

  useEffect(() => {
    let cancelled = false;
    queueMicrotask(() => {
      if (cancelled) {
        return;
      }

      const token = getStoredAccessToken();
      setAuthToken(token);
      setAuthChecked(true);
      if (token) {
        void loadApplications();
      }
    });

    return () => {
      cancelled = true;
    };
  }, []);

  async function loadApplications() {
    setIsLoading(true);
    try {
      setApplications(await getVerificationApplications());
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "认证状态加载失败");
    } finally {
      setIsLoading(false);
    }
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canSubmit) {
      return;
    }

    setIsSubmitting(true);
    try {
      await createVerificationApplication(draft);
      toast.success("认证申请已提交");
      setDraft({ ...emptyDraft, imageUrls: [] });
      await loadApplications();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "认证申请提交失败");
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleImageUpload(event: ChangeEvent<HTMLInputElement>) {
    const input = event.currentTarget;
    const files = Array.from(input.files ?? []);
    input.value = "";
    if (!files.length) {
      return;
    }

    const imageFiles = files.filter((file) => file.type.startsWith("image/"));
    if (!imageFiles.length) {
      toast.error("请选择图片文件");
      return;
    }

    const capacity = maxVerificationImages - draft.imageUrls.length;
    if (capacity <= 0) {
      toast.error(`最多上传 ${maxVerificationImages} 张认证图片`);
      return;
    }

    const uploadFiles = imageFiles.slice(0, capacity);
    if (uploadFiles.length < imageFiles.length) {
      toast.info(`最多保留 ${maxVerificationImages} 张认证图片`);
    }

    setIsUploadingImages(true);
    try {
      const assets = await uploadImages(uploadFiles);
      const nextURLs = assets.map((asset) => asset.url).filter(Boolean);
      if (!nextURLs.length) {
        toast.error("图片上传失败");
        return;
      }
      setDraft((current) => ({
        ...current,
        imageUrls: Array.from(new Set([...current.imageUrls, ...nextURLs])).slice(0, maxVerificationImages),
      }));
      toast.success(nextURLs.length > 1 ? "认证图片已上传" : "认证图片已添加");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "图片上传失败");
    } finally {
      setIsUploadingImages(false);
    }
  }

  function handleRemoveImage(index: number) {
    setDraft((current) => ({
      ...current,
      imageUrls: current.imageUrls.filter((_, itemIndex) => itemIndex !== index),
    }));
  }

  return (
    <main className="theme-adaptive min-h-dvh bg-[#121212] text-white">
      <div className="mx-auto flex min-h-dvh w-full max-w-[920px] flex-col">
        <header className="sticky top-0 z-20 flex h-14 items-center gap-2 border-b border-white/[0.07] bg-[#121212]/95 px-3 backdrop-blur">
          <Button
            asChild
            variant="ghost"
            size="icon"
            className="size-10 text-white/78 hover:bg-white/[0.06] hover:text-white"
          >
            <Link href="/settings">
              <ArrowLeft className="size-5" />
            </Link>
          </Button>
          <div className="min-w-0 flex-1">
            <h1 className="truncate text-lg font-bold">申请认证</h1>
          </div>
        </header>

        <section className="min-h-0 flex-1 overflow-y-auto px-4 pb-[calc(28px+env(safe-area-inset-bottom))] pt-4">
          {!authChecked ? (
            <LoadingPanel />
          ) : !authToken ? (
            <LoginPanel />
          ) : (
            <div
              className={cn(
                "grid gap-4",
                approvedApplication ? "lg:grid-cols-1" : "lg:grid-cols-[minmax(0,1fr)_minmax(300px,0.82fr)]",
              )}
            >
              <form
                onSubmit={handleSubmit}
                className="min-w-0 rounded-[8px] border border-white/[0.08] bg-white/[0.06] p-4 sm:p-5"
              >
                <div className="flex items-start gap-3">
                  <span className="flex size-11 shrink-0 items-center justify-center rounded-full bg-primary/15 text-primary">
                    <ShieldCheck className="size-5" />
                  </span>
                  <div className="min-w-0 flex-1">
                    <h2 className="text-base font-bold">提交认证资料</h2>
                    <p className="mt-1 text-sm leading-5 text-white/50">
                      已通过或待审核的申请会占用当前认证名额。
                    </p>
                  </div>
                </div>

                {approvedApplication ? (
                  <ApprovedVerificationPanel application={approvedApplication} />
                ) : (
                  <>
                    <div className="mt-5 grid grid-cols-2 gap-2 rounded-[8px] bg-white/[0.06] p-1">
                      {verificationTypeOptions.map(({ value, label, icon: Icon }) => {
                        const active = draft.type === value;
                        return (
                          <button
                            key={value}
                            type="button"
                            aria-pressed={active}
                            disabled={Boolean(blockingApplication)}
                            onClick={() => setDraft((current) => ({ ...current, type: value }))}
                            className={cn(
                              "flex h-10 min-w-0 items-center justify-center gap-2 rounded-[6px] px-3 text-sm font-bold transition-colors disabled:cursor-not-allowed disabled:opacity-55",
                              active
                                ? "bg-white text-[#111827]"
                                : "text-white/58 hover:bg-white/[0.07] hover:text-white",
                            )}
                          >
                            <Icon className="size-4 shrink-0" />
                            <span className="truncate">{label}</span>
                          </button>
                        );
                      })}
                    </div>

                    <label className="mt-5 block">
                      <span className="text-sm font-semibold text-white/76">认证名称</span>
                      <input
                        value={draft.verifiedName}
                        onChange={(event) =>
                          setDraft((current) => ({ ...current, verifiedName: event.target.value }))
                        }
                        disabled={Boolean(blockingApplication)}
                        maxLength={30}
                        placeholder="例如 原创创作者 / 摄影创作者"
                        className="mt-2 h-11 w-full rounded-[8px] border border-white/[0.08] bg-black/24 px-3 text-sm text-white outline-none placeholder:text-white/30 focus:border-primary/60 focus:ring-2 focus:ring-primary/20 disabled:cursor-not-allowed disabled:opacity-60"
                      />
                    </label>

                    <label className="mt-4 block">
                      <span className="text-sm font-semibold text-white/76">认证说明</span>
                      <textarea
                        value={draft.content}
                        onChange={(event) =>
                          setDraft((current) => ({ ...current, content: event.target.value }))
                        }
                        disabled={Boolean(blockingApplication)}
                        maxLength={500}
                        rows={7}
                        placeholder="填写身份、领域、资质或可核验链接。"
                        className="mt-2 w-full resize-none rounded-[8px] border border-white/[0.08] bg-black/24 px-3 py-3 text-sm leading-6 text-white outline-none placeholder:text-white/30 focus:border-primary/60 focus:ring-2 focus:ring-primary/20 disabled:cursor-not-allowed disabled:opacity-60"
                      />
                      <span className="mt-2 block text-right text-xs text-white/36">
                        {draft.content.length}/500
                      </span>
                    </label>

                    <input
                      ref={imageInputRef}
                      type="file"
                      accept="image/*"
                      multiple
                      disabled={Boolean(blockingApplication) || isUploadingImages}
                      className="hidden"
                      onChange={(event) => void handleImageUpload(event)}
                    />
                    <VerificationImageUploader
                      disabled={Boolean(blockingApplication)}
                      imageUrls={draft.imageUrls}
                      uploading={isUploadingImages}
                      onRemove={handleRemoveImage}
                      onUploadClick={() => imageInputRef.current?.click()}
                    />

                    {blockingApplication ? (
                      <div className="mt-4 rounded-[8px] border border-amber-300/20 bg-amber-300/10 px-3 py-2 text-sm text-amber-100">
                        当前已有{statusLabel(blockingApplication.status)}的认证申请，暂不能重复提交。
                      </div>
                    ) : null}

                    <Button
                      type="submit"
                      disabled={!canSubmit}
                      className="mt-5 h-11 w-full rounded-[8px] bg-primary text-base font-bold text-white hover:bg-primary/90 disabled:opacity-50"
                    >
                      {isSubmitting ? <Loader2 className="size-5 animate-spin" /> : <Send className="size-5" />}
                      提交申请
                    </Button>
                  </>
                )}
              </form>

              {approvedApplication ? null : (
                <aside className="min-w-0 space-y-4">
                  <section className="rounded-[8px] border border-white/[0.08] bg-white/[0.06] p-4">
                    <div className="flex items-center gap-3">
                      <span className="flex size-10 shrink-0 items-center justify-center rounded-full bg-white/[0.07] text-white/70">
                        <BadgeCheck className="size-5" />
                      </span>
                      <div className="min-w-0 flex-1">
                        <h2 className="truncate text-base font-bold">最近状态</h2>
                        <p className="mt-1 text-xs text-white/42">以最新一条认证申请为准。</p>
                      </div>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        disabled={isLoading}
                        aria-label="刷新认证状态"
                        onClick={() => void loadApplications()}
                        className="size-9 text-white/62 hover:bg-white/[0.06] hover:text-white"
                      >
                        <RotateCcw className={cn("size-4", isLoading && "animate-spin")} />
                      </Button>
                    </div>

                    {isLoading ? (
                      <div className="mt-4 flex h-28 items-center justify-center text-sm text-white/52">
                        <Loader2 className="mr-2 size-4 animate-spin" />
                        正在加载
                      </div>
                    ) : latestApplication ? (
                      <div className="mt-4 rounded-[8px] bg-black/22 p-4">
                        <StatusPill status={latestApplication.status} />
                        <div className="mt-3 grid gap-2 text-sm">
                          <InfoLine label="类型" value={typeLabel(latestApplication.type)} />
                          <InfoLine
                            label="名称"
                            value={verificationName(latestApplication) || "未填写"}
                          />
                          <InfoLine label="提交时间" value={formatDateTime(latestApplication.created_at)} />
                          {latestApplication.reason ? (
                            <InfoLine label="审核意见" value={latestApplication.reason} />
                          ) : null}
                        </div>
                        <VerificationImagesPreview urls={verificationImages(latestApplication)} />
                      </div>
                    ) : (
                      <div className="mt-4 rounded-[8px] bg-black/22 px-4 py-8 text-center text-sm text-white/50">
                        暂无认证申请
                      </div>
                    )}
                  </section>

                  <section className="rounded-[8px] border border-white/[0.08] bg-white/[0.06] p-4">
                    <div className="flex items-center gap-2 text-base font-bold">
                      <FileText className="size-5 text-white/65" />
                      申请记录
                    </div>
                    <div className="mt-4 space-y-2">
                      {applications.length > 0 ? (
                        applications.map((application) => (
                          <ApplicationRow key={application.id} application={application} />
                        ))
                      ) : (
                        <div className="rounded-[8px] border border-dashed border-white/[0.1] px-3 py-5 text-center text-sm text-white/40">
                          没有历史记录
                        </div>
                      )}
                    </div>
                  </section>
                </aside>
              )}
            </div>
          )}
        </section>
      </div>
    </main>
  );
}

function LoadingPanel() {
  return (
    <div className="mx-auto mt-10 flex max-w-[440px] items-center justify-center rounded-[8px] border border-white/[0.08] bg-white/[0.06] px-5 py-8 text-sm text-white/54">
      <Loader2 className="mr-2 size-5 animate-spin" />
      正在检查登录状态
    </div>
  );
}

function LoginPanel() {
  return (
    <div className="mx-auto mt-10 max-w-[440px] rounded-[8px] border border-white/[0.08] bg-white/[0.06] px-5 py-8 text-center">
      <ShieldCheck className="mx-auto size-10 text-primary" />
      <h2 className="mt-4 text-lg font-bold">登录后申请认证</h2>
      <p className="mt-2 text-sm leading-6 text-white/50">认证资料需要绑定到当前账号。</p>
      <Button asChild className="mt-5 h-10 rounded-full bg-primary px-5 text-white">
        <Link href="/login">去登录</Link>
      </Button>
    </div>
  );
}

function ApplicationRow({ application }: { application: VerificationApplication }) {
  return (
    <div className="rounded-[8px] bg-black/20 px-3 py-3">
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0">
          <p className="truncate text-sm font-bold text-white">{verificationName(application) || typeLabel(application.type)}</p>
          <p className="mt-1 text-xs text-white/38">{formatDateTime(application.created_at)}</p>
        </div>
        <StatusPill status={application.status} compact />
      </div>
    </div>
  );
}

function VerificationImageUploader({
  disabled,
  imageUrls,
  onRemove,
  onUploadClick,
  uploading,
}: {
  disabled?: boolean;
  imageUrls: string[];
  onRemove: (index: number) => void;
  onUploadClick: () => void;
  uploading: boolean;
}) {
  const uploadDisabled = disabled || uploading || imageUrls.length >= maxVerificationImages;
  return (
    <div className="mt-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <span className="text-sm font-semibold text-white/76">认证图片</span>
          <p className="mt-1 text-xs leading-5 text-white/38">
            可上传原创图片、作品，最多 {maxVerificationImages} 张。
          </p>
        </div>
        <Button
          type="button"
          variant="outline"
          disabled={uploadDisabled}
          onClick={onUploadClick}
          className="h-9 shrink-0 rounded-[8px] border-white/[0.12] bg-transparent px-3 text-white hover:bg-white/[0.07] disabled:opacity-50"
        >
          {uploading ? <Loader2 className="size-4 animate-spin" /> : <ImagePlus className="size-4" />}
          上传
        </Button>
      </div>

      {imageUrls.length > 0 ? (
        <div className="mt-3 grid grid-cols-3 gap-2 sm:grid-cols-4">
          {imageUrls.map((url, index) => (
            <div key={`${url}-${index}`} className="group relative aspect-square overflow-hidden rounded-[8px] bg-black/28">
              <Image
                src={url}
                alt={`认证图片 ${index + 1}`}
                fill
                sizes="(max-width: 640px) 33vw, 160px"
                unoptimized
                className="object-cover"
              />
              <button
                type="button"
                aria-label="移除认证图片"
                title="移除认证图片"
                disabled={disabled || uploading}
                onClick={() => onRemove(index)}
                className="absolute right-1.5 top-1.5 inline-flex size-7 items-center justify-center rounded-full bg-black/62 text-white shadow-sm transition hover:bg-black/78 disabled:cursor-not-allowed disabled:opacity-50"
              >
                <Trash2 className="size-3.5" />
              </button>
            </div>
          ))}
        </div>
      ) : (
        <button
          type="button"
          disabled={uploadDisabled}
          onClick={onUploadClick}
          className="mt-3 flex h-24 w-full items-center justify-center rounded-[8px] border border-dashed border-white/[0.14] bg-black/16 text-sm text-white/42 transition hover:border-primary/42 hover:text-white/70 disabled:cursor-not-allowed disabled:opacity-55"
        >
          {uploading ? <Loader2 className="mr-2 size-4 animate-spin" /> : <ImagePlus className="mr-2 size-4" />}
          上传认证图片
        </button>
      )}
    </div>
  );
}

function ApprovedVerificationPanel({ application }: { application: VerificationApplication }) {
  const images = verificationImages(application);
  return (
    <div className="mt-5 rounded-[8px] border border-emerald-300/20 bg-emerald-300/10 p-4">
      <div className="flex items-start gap-3">
        <span className="flex size-11 shrink-0 items-center justify-center rounded-full bg-emerald-300/15 text-emerald-100">
          <BadgeCheck className="size-5" />
        </span>
        <div className="min-w-0 flex-1">
          <h3 className="text-base font-bold text-emerald-50">认证信息</h3>
          <p className="mt-1 text-sm leading-5 text-emerald-100/72">
            当前账号已通过认证，暂不能重复提交。
          </p>
        </div>
        <StatusPill status={application.status} />
      </div>

      <div className="mt-4 space-y-3 rounded-[8px] bg-black/16 p-4 text-sm">
        <InfoLine label="认证名称" value={verificationName(application) || typeLabel(application.type)} />
        <InfoLine label="认证详情" value={verificationContent(application) || "未填写"} />
        <InfoLine
          label="认证时间"
          value={formatDateTime(application.audit_time ?? application.created_at)}
        />
        <VerificationImagesPreview urls={images} />
      </div>
    </div>
  );
}

function VerificationImagesPreview({ urls }: { urls: string[] }) {
  if (!urls.length) {
    return null;
  }
  return (
    <div className="mt-3">
      <p className="mb-2 text-xs font-semibold text-white/42">认证图片</p>
      <div className="grid grid-cols-3 gap-2 sm:grid-cols-4">
        {urls.slice(0, maxVerificationImages).map((url, index) => (
          <a
            key={`${url}-${index}`}
            href={url}
            target="_blank"
            rel="noreferrer"
            className="relative aspect-square overflow-hidden rounded-[8px] bg-black/28 transition hover:ring-2 hover:ring-white/24"
          >
            <Image
              src={url}
              alt={`认证图片 ${index + 1}`}
              fill
              sizes="(max-width: 640px) 33vw, 160px"
              unoptimized
              className="object-cover"
            />
          </a>
        ))}
      </div>
    </div>
  );
}

function InfoLine({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid grid-cols-[72px_minmax(0,1fr)] gap-2">
      <span className="text-white/38">{label}</span>
      <span className="min-w-0 break-words text-white/78">{value}</span>
    </div>
  );
}

function StatusPill({ compact, status }: { compact?: boolean; status?: number | null }) {
  const normalized = normalizeStatus(status);
  const Icon = normalized === 1 ? CheckCircle2 : normalized === 2 ? XCircle : Clock3;
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-bold",
        statusClassName(normalized),
        compact && "px-2 py-0.5",
      )}
    >
      <Icon className="size-3.5" />
      {statusLabel(status)}
    </span>
  );
}

function statusClassName(status: number) {
  if (status === 1) {
    return "border-emerald-300/20 bg-emerald-300/12 text-emerald-100";
  }
  if (status === 2) {
    return "border-rose-300/20 bg-rose-300/12 text-rose-100";
  }
  return "border-amber-300/20 bg-amber-300/12 text-amber-100";
}

function normalizeStatus(status: number | null | undefined) {
  return typeof status === "number" ? status : 0;
}

function statusLabel(status: number | null | undefined) {
  const normalized = normalizeStatus(status);
  if (normalized === 1) {
    return "已通过";
  }
  if (normalized === 2) {
    return "未通过";
  }
  return "审核中";
}

function typeLabel(type: number) {
  return type === 2 ? "官方认证" : "个人认证";
}

function verificationName(application: VerificationApplication) {
  const value =
    application.audit_result?.verifiedName ??
    application.audit_result?.verified_name;
  return typeof value === "string" ? value : "";
}

function verificationContent(application: VerificationApplication) {
  return typeof application.content === "string" ? application.content.trim() : "";
}

function verificationImages(application: VerificationApplication) {
  return normalizeImageUrls(
    application.audit_result?.imageUrls ??
    application.audit_result?.image_urls ??
    application.audit_result?.images,
  );
}

function normalizeImageUrls(value: unknown): string[] {
  if (Array.isArray(value)) {
    return Array.from(new Set(value.flatMap((item) => normalizeImageUrls(item)))).slice(0, maxVerificationImages);
  }
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (!trimmed) {
      return [];
    }
    if (trimmed.startsWith("[") || trimmed.startsWith("{")) {
      try {
        return normalizeImageUrls(JSON.parse(trimmed));
      } catch {
        return [];
      }
    }
    return trimmed.split(/\r?\n|,|，/).map((item) => item.trim()).filter(Boolean).slice(0, maxVerificationImages);
  }
  if (value && typeof value === "object") {
    const record = value as Record<string, unknown>;
    return normalizeImageUrls(record.url ?? record.src ?? record.image_url);
  }
  return [];
}

function formatDateTime(value?: string | null) {
  if (!value) {
    return "未知时间";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat("zh-CN", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}
