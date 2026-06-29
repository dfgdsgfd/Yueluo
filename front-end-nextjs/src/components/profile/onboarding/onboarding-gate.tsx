"use client";

import {
  type ChangeEvent,
  type FormEvent,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import Image from "next/image";
import Link from "next/link";
import {
  ArrowRight,
  Camera,
  CheckCircle2,
  ImageIcon,
  Info,
  Loader2,
  Sparkles,
  X,
} from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import {
  getCurrentUser,
  getOnboardingConfig,
  getStoredUser,
  storeAuthenticatedUser,
  submitOnboarding,
  uploadImage,
} from "@/lib/api";
import type {
  AuthUser,
  OnboardingConfigPayload,
  OnboardingFieldRule,
  OnboardingProfileTask,
  OnboardingSubmitInput,
} from "@/lib/types";
import { Button } from "@/components/ui/button";
import {
  Avatar,
  AvatarFallback,
  AvatarImage,
} from "@/components/ui/avatar";

type Draft = {
  avatar: string;
  background: string;
  bio: string;
  interests: string[];
  nickname: string;
};

const taskKeys = ["set_avatar", "set_background", "set_name", "set_signature"] as const;

type CompletionItem = {
  completed: boolean;
  key: "avatar" | "background" | "name" | "signature" | "interests";
  label: string;
  required: boolean;
};

export function OnboardingGate({ onClose }: { onClose: () => void }) {
  const t = useTranslations("onboarding");
  const [config, setConfig] = useState<OnboardingConfigPayload | null>(null);
  const [user, setUser] = useState<AuthUser | null>(() => getStoredUser());
  const [draft, setDraft] = useState<Draft>(() => draftFromUser(getStoredUser()));
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [uploading, setUploading] = useState<"avatar" | "background" | null>(null);
  const [resultOpen, setResultOpen] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [earnedPoints, setEarnedPoints] = useState(0);
  const avatarInputRef = useRef<HTMLInputElement | null>(null);
  const backgroundInputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      try {
        const [nextConfig, userResult] = await Promise.all([
          getOnboardingConfig(),
          getCurrentUser()
            .then((value) => ({ fromDatabase: true, user: value }))
            .catch(() => ({ fromDatabase: false, user: getStoredUser() })),
        ]);
        const nextUser = userResult.user;
        if (cancelled) {
          return;
        }
        setConfig(nextConfig);
        setUser(nextUser);
        setDraft(draftFromUser(nextUser));
        if (nextUser) {
          storeAuthenticatedUser(nextUser);
        }
        if (!nextConfig.enabled || (userResult.fromDatabase && nextUser?.profile_completed === true)) {
          onClose();
        }
      } catch {
        if (!cancelled) {
          onClose();
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, [onClose]);

  const tasks = useMemo(() => normalizeTasks(config?.profile_tasks), [config?.profile_tasks]);
  const fieldRules = useMemo(() => normalizeFieldRules(config?.fields), [config?.fields]);
  const interestOptions = useMemo(() => normalizeInterestOptions(config?.interest_options), [config?.interest_options]);
  const availableFieldRules = useMemo(
    () => normalizeAvailableFieldRules(fieldRules, interestOptions),
    [fieldRules, interestOptions],
  );
  const completionItems = useMemo(
    () => onboardingCompletionItems(draft, availableFieldRules, t),
    [availableFieldRules, draft, t],
  );
  const potentialPoints = useMemo(() => {
    return taskKeys.reduce((total, taskType) => {
      const task = tasks[taskType];
      if (!task?.is_active) {
        return total;
      }
      if (!taskEnabledForRule(taskType, availableFieldRules)) {
        return total;
      }
      const shouldCount =
        (taskType === "set_avatar" && Boolean(draft.avatar.trim()) && draft.avatar.trim() !== (user?.avatar ?? "")) ||
        (taskType === "set_background" && Boolean(draft.background.trim()) && draft.background.trim() !== (user?.background ?? "")) ||
        (taskType === "set_name" && Boolean(draft.nickname.trim()) && draft.nickname.trim() !== (user?.nickname ?? "")) ||
        (taskType === "set_signature" && Boolean(draft.bio.trim()) && draft.bio.trim() !== (user?.bio ?? ""));
      return shouldCount ? total + task.points : total;
    }, 0);
  }, [availableFieldRules, draft, tasks, user]);
  const totalAvailablePoints = useMemo(() => {
    return taskKeys.reduce((total, taskType) => {
      const task = tasks[taskType];
      return task?.is_active && taskEnabledForRule(taskType, availableFieldRules) ? total + task.points : total;
    }, 0);
  }, [availableFieldRules, tasks]);
  const validationError = validateDraft(draft, availableFieldRules, t);

  if (loading || !config || !config.enabled) {
    return null;
  }

  const intro = config.points_intro ?? {};

  async function handleImageUpload(kind: "avatar" | "background", event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0] ?? null;
    event.target.value = "";
    if (!file) {
      return;
    }
    setUploading(kind);
    try {
      const asset = await uploadImage(file, { purpose: kind });
      setDraft((current) => ({ ...current, [kind]: asset.signedUrl || asset.url }));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("uploadFailed"));
    } finally {
      setUploading(null);
    }
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (validationError) {
      toast.error(validationError);
      return;
    }
    setSubmitting(true);
    try {
      const updated = await submitOnboarding(onboardingSubmitInputFromDraft(draft, availableFieldRules));
      storeAuthenticatedUser(updated);
      setUser(updated);
      setEarnedPoints(sumAwards(updated.points_awards));
      setResultOpen(true);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("saveFailed"));
    } finally {
      setSubmitting(false);
    }
  }

  async function handleSkip() {
    setSubmitting(true);
    try {
      const updated = await submitOnboarding({ skipped: true });
      storeAuthenticatedUser(updated);
      onClose();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("skipFailed"));
    } finally {
      setSubmitting(false);
    }
  }

  function finish() {
    setResultOpen(false);
    onClose();
  }

  return (
    <div className="fixed inset-0 z-[9998] flex min-h-0 items-end justify-center overflow-hidden bg-[#101116]/68 backdrop-blur-sm md:items-center md:p-6">
      <form
        onSubmit={handleSubmit}
        className="flex h-[calc(100dvh-env(safe-area-inset-top)-0.5rem)] max-h-[calc(100vh-0.5rem)] w-full max-w-[960px] flex-col overflow-hidden rounded-t-[8px] border border-white/15 bg-[#f8fafc] text-[#171923] shadow-2xl md:h-auto md:max-h-[min(88vh,760px)] md:rounded-[8px]"
      >
        <div className="flex min-h-14 shrink-0 items-center justify-between gap-3 border-b border-black/[0.06] px-4 sm:px-5">
          <div className="min-w-0">
            <h2 className="truncate text-[clamp(1rem,4.3vw,1.25rem)] font-black tracking-normal">{t("title")}</h2>
            <p className="hidden text-sm font-medium text-[#6b7280] sm:block">{t("subtitle")}</p>
          </div>
          {config.allow_skip ? (
            <Button type="button" variant="ghost" size="icon" disabled={submitting} onClick={handleSkip} aria-label={t("skip")} className="size-9 rounded-lg">
              {submitting ? <Loader2 className="size-4 animate-spin" /> : <X className="size-4" />}
            </Button>
          ) : null}
        </div>

        <div className="grid min-h-0 flex-1 touch-pan-y gap-0 overflow-y-auto overscroll-contain md:grid-cols-[minmax(0,1fr)_minmax(16rem,20rem)]">
          <div className="min-w-0 px-4 py-4 sm:px-5 md:py-5">
            <div className="relative min-h-[148px] overflow-hidden rounded-[8px] bg-[#dbe4f0]">
              {availableFieldRules.background.enabled && draft.background ? (
                <Image src={draft.background} alt={t("backgroundPreview")} fill sizes="(max-width: 768px) 100vw, 640px" className="object-cover" unoptimized />
              ) : (
                <div className="flex h-full min-h-[148px] items-center justify-center bg-[linear-gradient(135deg,#e0f2fe,#f4f4f5_48%,#dcfce7)]">
                  <ImageIcon className="size-8 text-[#64748b]" />
                </div>
              )}
              {availableFieldRules.background.enabled ? (
                <button
                  type="button"
                  onClick={() => backgroundInputRef.current?.click()}
                  disabled={Boolean(uploading) || submitting}
                  className="absolute bottom-3 right-3 inline-flex h-9 items-center gap-1.5 rounded-lg bg-[#111827]/88 px-2.5 text-xs font-bold text-white shadow-lg transition hover:bg-[#111827] sm:gap-2 sm:px-3 sm:text-sm"
                >
                  {uploading === "background" ? <Loader2 className="size-4 animate-spin" /> : <ImageIcon className="size-4" />}
                  <span>{t("changeBackground")}</span>
                </button>
              ) : null}
            </div>

            <div className="-mt-5 flex items-end gap-3 px-3 sm:-mt-8">
              {availableFieldRules.avatar.enabled ? (
              <div className="relative">
                <Avatar className="size-[clamp(4.25rem,18vw,5.5rem)] border-4 border-[#f8fafc] bg-white shadow-lg">
                  <AvatarImage src={draft.avatar || undefined} alt={draft.nickname || t("avatarPreview")} />
                  <AvatarFallback>{(draft.nickname || t("avatarFallback")).slice(0, 1).toUpperCase()}</AvatarFallback>
                </Avatar>
                <button
                  type="button"
                  onClick={() => avatarInputRef.current?.click()}
                  disabled={Boolean(uploading) || submitting}
                  aria-label={t("changeAvatar")}
                  className="absolute bottom-1 right-1 grid size-8 place-items-center rounded-full bg-[#ef4444] text-white shadow-lg"
                >
                  {uploading === "avatar" ? <Loader2 className="size-4 animate-spin" /> : <Camera className="size-4" />}
                </button>
              </div>
              ) : null}
              <div className="min-w-0 pb-2">
                <p className="truncate text-lg font-black">{draft.nickname || t("namePlaceholder")}</p>
                <p className="line-clamp-2 text-sm font-medium text-[#6b7280]">{draft.bio || t("bioPlaceholder")}</p>
              </div>
            </div>

            <div className="mt-5 grid gap-3">
              {availableFieldRules.name.enabled ? (
              <label className="grid gap-1.5">
                <span className="text-sm font-black text-[#30343b]">{t("nickname")}</span>
                <input
                  value={draft.nickname}
                  onChange={(event) => setDraft((current) => ({ ...current, nickname: event.target.value }))}
                  maxLength={20}
                  required={availableFieldRules.name.required}
                  className="h-11 w-full rounded-lg border border-black/[0.08] bg-white px-3 text-base font-semibold outline-none transition focus:border-[#ef4444] focus:ring-2 focus:ring-[#ef4444]/15"
                  placeholder={t("nicknamePlaceholder")}
                />
              </label>
              ) : null}
              {availableFieldRules.signature.enabled ? (
              <label className="grid gap-1.5">
                <span className="text-sm font-black text-[#30343b]">{t("bio")}</span>
                <textarea
                  value={draft.bio}
                  onChange={(event) => setDraft((current) => ({ ...current, bio: event.target.value }))}
                  maxLength={160}
                  className="min-h-[92px] w-full resize-none rounded-lg border border-black/[0.08] bg-white px-3 py-2 text-sm font-medium leading-6 outline-none transition focus:border-[#ef4444] focus:ring-2 focus:ring-[#ef4444]/15"
                  placeholder={t("bioPlaceholder")}
                />
              </label>
              ) : null}
              {availableFieldRules.interests.enabled ? (
                <div className="grid gap-2">
                  <span className="text-sm font-black text-[#30343b]">{t("interests")}</span>
                  <div className="flex flex-wrap gap-2">
                    {interestOptions.map((interest) => {
                      const selected = draft.interests.includes(interest);
                      return (
                        <button
                          key={interest}
                          type="button"
                          onClick={() => setDraft((current) => ({
                            ...current,
                            interests: selected ? current.interests.filter((item) => item !== interest) : [...current.interests, interest],
                          }))}
                          className={selected ? "rounded-lg bg-[#ef4444] px-3 py-2 text-sm font-bold text-white" : "rounded-lg border border-black/[0.08] bg-white px-3 py-2 text-sm font-bold text-[#4b5563]"}
                        >
                          {interest}
                        </button>
                      );
                    })}
                  </div>
                </div>
              ) : null}
            </div>
          </div>

          <aside className="min-w-0 border-t border-black/[0.06] bg-white px-4 py-4 sm:px-5 md:border-l md:border-t-0">
            <div className="grid gap-3">
              <CompletionPanel items={completionItems} t={t} />
              <div className="rounded-[8px] border border-[#fee2e2] bg-[#fff7f7] p-4">
                <div className="flex items-center gap-2">
                  <Sparkles className="size-5 text-[#ef4444]" />
                  <h3 className="text-base font-black">{t("pointsTitle")}</h3>
                </div>
                <p className="mt-2 text-[clamp(0.8125rem,3.5vw,0.9375rem)] font-medium leading-6 text-[#6b7280]">
                  {t("pointsEstimate", { count: formatPoints(potentialPoints), total: formatPoints(totalAvailablePoints) })}
                </p>
                <div className="mt-3 grid gap-2">
                  {taskKeys.map((taskType) => (
                    taskEnabledForRule(taskType, availableFieldRules) ? <TaskRow key={taskType} task={tasks[taskType]} label={t(`tasks.${taskType}`)} /> : null
                  ))}
                </div>
              </div>
            </div>

          </aside>
        </div>

        <div className="flex shrink-0 flex-col gap-2 border-t border-black/[0.06] bg-white px-4 pb-[max(0.75rem,env(safe-area-inset-bottom))] pt-3 sm:flex-row sm:items-center sm:justify-between sm:px-5 sm:pb-3">
          <p className="text-xs font-semibold text-[#7b8190]">{config.allow_skip ? t("skipHint") : t("requiredHint")}</p>
          <div className="flex gap-2">
            {config.allow_skip ? (
              <Button type="button" variant="outline" disabled={submitting} onClick={handleSkip} className="h-10 flex-1 rounded-lg border-black/[0.08] bg-white px-4 sm:flex-none">
                {t("skip")}
              </Button>
            ) : null}
            <Button type="submit" disabled={submitting || Boolean(uploading) || Boolean(validationError)} className="h-10 flex-1 rounded-lg bg-[#ef4444] px-5 font-black hover:bg-[#dc2626] sm:flex-none">
              {submitting ? <Loader2 className="size-4 animate-spin" /> : <CheckCircle2 className="size-4" />}
              <span>{submitting ? t("saving") : t("complete")}</span>
            </Button>
          </div>
        </div>

        <input ref={avatarInputRef} className="hidden" type="file" accept="image/*" onChange={(event) => void handleImageUpload("avatar", event)} />
        <input ref={backgroundInputRef} className="hidden" type="file" accept="image/*" onChange={(event) => void handleImageUpload("background", event)} />
      </form>

      {resultOpen ? (
        <PointsResultDialog
          detail={intro.detail || t("defaultIntroDetail")}
          earnedPoints={earnedPoints}
          onClose={finish}
          onDetail={() => setDetailOpen(true)}
          savedText={intro.saved_text || t("result.saved")}
          summary={intro.summary || t("defaultIntroSummary")}
          title={intro.title || t("defaultIntroTitle")}
          walletLabel={intro.wallet_label || t("detail.openWallet")}
          walletUrl={intro.wallet_url || "/wallet"}
          resultTitle={intro.result_title || t("result.title")}
        />
      ) : null}

      {detailOpen ? (
        <PointsDetailDialog
          detail={intro.detail || t("defaultIntroDetail")}
          onClose={() => setDetailOpen(false)}
          title={intro.title || t("defaultIntroTitle")}
          walletLabel={intro.wallet_label || t("detail.openWallet")}
          walletUrl={intro.wallet_url || "/wallet"}
        />
      ) : null}
    </div>
  );
}

function CompletionPanel({ items, t }: { items: CompletionItem[]; t: ReturnType<typeof useTranslations> }) {
  if (!items.length) {
    return null;
  }
  const completedCount = items.filter((item) => item.completed).length;
  const progress = Math.round((completedCount / items.length) * 100);
  const missingRequired = items.filter((item) => item.required && !item.completed);
  const missingText = missingRequired.length
    ? t("missingSummary", { items: new Intl.ListFormat(undefined, { style: "short", type: "conjunction" }).format(missingRequired.map((item) => item.label)) })
    : t("allRequiredComplete");

  return (
    <div className="rounded-[8px] border border-[#dbeafe] bg-[#eff6ff] p-4">
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0">
          <h3 className="truncate text-base font-black text-[#172033]">{t("completionTitle")}</h3>
          <p className="mt-1 text-xs font-semibold leading-5 text-[#5f6f86]">{t("completionSummary", { completed: completedCount, total: items.length })}</p>
        </div>
        <span className="shrink-0 rounded-full bg-white px-2.5 py-1 text-sm font-black text-[#1d4ed8]">{progress}%</span>
      </div>
      <div className="mt-3 h-2 overflow-hidden rounded-full bg-white">
        <div className="h-full rounded-full bg-[#2563eb] transition-[width]" style={{ width: `${progress}%` }} />
      </div>
      <p className="mt-3 text-sm font-semibold leading-6 text-[#40506a]">{missingText}</p>
      <div className="mt-3 grid gap-2">
        {items.map((item) => (
          <div key={item.key} className="flex min-h-10 items-center gap-2 rounded-lg bg-white px-3 py-2 text-sm">
            <span className={item.completed ? "grid size-7 shrink-0 place-items-center rounded-full bg-[#dcfce7] text-[#16a34a]" : item.required ? "grid size-7 shrink-0 place-items-center rounded-full bg-[#fee2e2] text-[#dc2626]" : "grid size-7 shrink-0 place-items-center rounded-full bg-[#f3f4f6] text-[#6b7280]"}>
              {item.completed ? <CheckCircle2 className="size-4" /> : <Info className="size-4" />}
            </span>
            <span className="min-w-0 flex-1 truncate font-bold text-[#293142]">{item.label}</span>
            <span className={item.completed ? "shrink-0 text-xs font-black text-[#16a34a]" : item.required ? "shrink-0 text-xs font-black text-[#dc2626]" : "shrink-0 text-xs font-black text-[#6b7280]"}>
              {item.completed ? t("statusComplete") : item.required ? t("statusMissingRequired") : t("statusOptional")}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

function TaskRow({ label, task }: { label: string; task?: OnboardingProfileTask }) {
  return (
    <div className="flex min-h-9 items-center justify-between gap-3 rounded-lg bg-white px-3 py-2 text-sm">
      <span className="min-w-0 truncate font-bold text-[#333842]">{label}</span>
      <span className="shrink-0 font-black text-[#16a34a]">+{formatPoints(task?.is_active ? task.points : 0)}</span>
    </div>
  );
}

function PointsResultDialog({
  detail,
  earnedPoints,
  onClose,
  onDetail,
  resultTitle,
  savedText,
  summary,
  title,
  walletLabel,
  walletUrl,
}: {
  detail: string;
  earnedPoints: number;
  onClose: () => void;
  onDetail: () => void;
  resultTitle: string;
  savedText: string;
  summary: string;
  title: string;
  walletLabel: string;
  walletUrl: string;
}) {
  const t = useTranslations("onboarding.result");
  return (
    <div className="fixed inset-0 z-[10000] grid place-items-center overflow-y-auto bg-[#111217]/62 px-4 py-[max(1rem,env(safe-area-inset-top))]">
      <div className="max-h-[calc(100vh-2rem)] w-full max-w-[420px] touch-pan-y overflow-y-auto overscroll-contain rounded-[8px] bg-white p-5 text-[#181b22] shadow-2xl supports-[height:100dvh]:max-h-[calc(100dvh-2rem)]">
        <div className="grid place-items-center text-center">
          <div className="grid size-14 place-items-center rounded-full bg-[#dcfce7] text-[#16a34a]">
            <CheckCircle2 className="size-8" />
          </div>
          <h3 className="mt-3 text-xl font-black">{resultTitle}</h3>
          <p className="mt-2 text-sm font-semibold leading-6 text-[#626a78]">
            {earnedPoints > 0 ? t("earned", { count: formatPoints(earnedPoints) }) : savedText}
          </p>
        </div>
        <Link
          href={walletUrl}
          onClick={onClose}
          className="mt-4 block rounded-[8px] border border-[#bfdbfe] bg-[#eff6ff] p-3 text-left transition hover:border-[#60a5fa] hover:bg-[#dbeafe] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/30"
        >
          <span className="flex items-center gap-2 font-black text-[#1f2937]">
            <Info className="size-4 text-[#2563eb]" />
            <span className="min-w-0 flex-1 truncate">{title}</span>
            <ArrowRight className="size-4 shrink-0 text-[#2563eb]" />
          </span>
          <span className="mt-1 line-clamp-3 block text-sm font-medium leading-6 text-[#4b5563]">{summary || detail}</span>
          <span className="mt-2 inline-flex items-center gap-1 text-sm font-black text-[#2563eb]">
            {walletLabel}
            <ArrowRight className="size-3.5" />
          </span>
        </Link>
        <div className="mt-4 flex gap-2">
          <Button type="button" variant="outline" onClick={onClose} className="h-10 flex-1 rounded-lg border-black/[0.08] bg-white">
            {t("later")}
          </Button>
          <Button type="button" onClick={onDetail} className="h-10 flex-1 rounded-lg bg-[#2563eb] hover:bg-[#1d4ed8]">
            {t("details")}
          </Button>
        </div>
      </div>
    </div>
  );
}

function PointsDetailDialog({
  detail,
  onClose,
  title,
  walletLabel,
  walletUrl,
}: {
  detail: string;
  onClose: () => void;
  title: string;
  walletLabel: string;
  walletUrl: string;
}) {
  const t = useTranslations("onboarding.detail");
  return (
    <div className="fixed inset-0 z-[10001] grid place-items-center bg-[#111217]/68 px-4">
      <div className="flex max-h-[82dvh] w-full max-w-[560px] flex-col overflow-hidden rounded-[8px] bg-white text-[#181b22] shadow-2xl">
        <div className="flex h-14 shrink-0 items-center justify-between gap-4 border-b border-black/[0.06] px-5">
          <h3 className="min-w-0 truncate text-lg font-black">{title}</h3>
          <Button type="button" variant="ghost" size="icon" onClick={onClose} aria-label={t("close")} className="size-9 rounded-lg">
            <X className="size-4" />
          </Button>
        </div>
        <div className="min-h-0 overflow-y-auto px-5 py-4">
          <p className="whitespace-pre-line text-sm font-medium leading-7 text-[#4b5563]">{detail}</p>
          <Link href={walletUrl} onClick={onClose} className="mt-4 inline-flex h-10 items-center justify-center rounded-lg bg-[#ef4444] px-4 text-sm font-black text-white transition hover:bg-[#dc2626]">
            {walletLabel || t("openWallet")}
          </Link>
        </div>
      </div>
    </div>
  );
}

function onboardingCompletionItems(
  draft: Draft,
  rules: ReturnType<typeof normalizeFieldRules>,
  t: ReturnType<typeof useTranslations>,
) {
  const items: CompletionItem[] = [];
  const add = (
    key: CompletionItem["key"],
    rule: OnboardingFieldRule,
    label: string,
    completed: boolean,
  ) => {
    if (!rule.enabled) {
      return;
    }
    items.push({
      completed,
      key,
      label,
      required: Boolean(rule.required),
    });
  };

  add("avatar", rules.avatar, t("tasks.set_avatar"), Boolean(draft.avatar.trim()));
  add("background", rules.background, t("tasks.set_background"), Boolean(draft.background.trim()));
  add("name", rules.name, t("tasks.set_name"), Boolean(draft.nickname.trim()));
  add("signature", rules.signature, t("tasks.set_signature"), Boolean(draft.bio.trim()));
  add("interests", rules.interests, t("interests"), draft.interests.length >= rules.interests.min);
  return items;
}

function draftFromUser(user: AuthUser | null): Draft {
  return {
    avatar: user?.avatar ?? "",
    background: user?.background ?? "",
    bio: user?.bio ?? "",
    interests: Array.isArray(user?.interests) ? user.interests.filter((item): item is string => typeof item === "string") : [],
    nickname: user?.nickname ?? "",
  };
}

function normalizeTasks(tasks?: OnboardingProfileTask[]) {
  return Object.fromEntries((tasks ?? []).map((task) => [task.task_type, task])) as Record<string, OnboardingProfileTask | undefined>;
}

function normalizeFieldRules(fields?: OnboardingConfigPayload["fields"]) {
  return {
    avatar: normalizeFieldRule(fields?.avatar, true, true),
    background: normalizeFieldRule(fields?.background, true, true),
    name: normalizeFieldRule(fields?.name, true, true),
    signature: normalizeFieldRule(fields?.signature, true, false),
    interests: normalizeFieldRule(fields?.interests, false, false, 1),
  };
}

function normalizeInterestOptions(options?: string[]) {
  const seen = new Set<string>();
  return (options ?? []).reduce<string[]>((items, option) => {
    const text = option.trim();
    if (text && !seen.has(text)) {
      seen.add(text);
      items.push(text);
    }
    return items;
  }, []);
}

function normalizeAvailableFieldRules(
  rules: ReturnType<typeof normalizeFieldRules>,
  interestOptions: string[],
) {
  if (!rules.interests.enabled || interestOptions.length === 0) {
    return {
      ...rules,
      interests: { ...rules.interests, enabled: false, required: false },
    };
  }
  return {
    ...rules,
    interests: {
      ...rules.interests,
      min: Math.min(rules.interests.min, interestOptions.length),
    },
  };
}

function normalizeFieldRule(rule: OnboardingFieldRule | undefined, enabled: boolean, required: boolean, min = 1) {
  return {
    enabled: rule?.enabled ?? enabled,
    min: Math.max(1, Number(rule?.min ?? min) || min),
    required: rule?.required ?? required,
  };
}

function taskEnabledForRule(taskType: string, rules: ReturnType<typeof normalizeFieldRules>) {
  if (taskType === "set_avatar") return rules.avatar.enabled;
  if (taskType === "set_background") return rules.background.enabled;
  if (taskType === "set_name") return rules.name.enabled;
  if (taskType === "set_signature") return rules.signature.enabled;
  return true;
}

function validateDraft(draft: Draft, rules: ReturnType<typeof normalizeFieldRules>, t: ReturnType<typeof useTranslations>) {
  if (rules.name.enabled && rules.name.required && !draft.nickname.trim()) return t("nicknameRequired");
  if (rules.avatar.enabled && rules.avatar.required && !draft.avatar.trim()) return t("avatarRequired");
  if (rules.background.enabled && rules.background.required && !draft.background.trim()) return t("backgroundRequired");
  if (rules.signature.enabled && rules.signature.required && !draft.bio.trim()) return t("bioRequired");
  if (rules.interests.enabled && rules.interests.required && draft.interests.length < rules.interests.min) {
    return t("interestsRequired", { count: rules.interests.min });
  }
  return "";
}

function onboardingSubmitInputFromDraft(draft: Draft, rules: ReturnType<typeof normalizeFieldRules>) {
  const input: OnboardingSubmitInput = {};
  if (rules.avatar.enabled) input.avatar = draft.avatar.trim();
  if (rules.background.enabled) input.background = draft.background.trim();
  if (rules.name.enabled) input.nickname = draft.nickname.trim();
  if (rules.signature.enabled) input.bio = draft.bio.trim();
  if (rules.interests.enabled) input.interests = draft.interests;
  return input;
}

function sumAwards(value: unknown) {
  if (!Array.isArray(value)) {
    return 0;
  }
  return value.reduce((total, item) => {
    if (!item || typeof item !== "object") {
      return total;
    }
    const amount = "amount" in item ? Number(item.amount) : 0;
    return Number.isFinite(amount) ? total + amount : total;
  }, 0);
}

function formatPoints(value: number) {
  return new Intl.NumberFormat(undefined, {
    maximumFractionDigits: Number.isInteger(value) ? 0 : 2,
  }).format(value);
}
