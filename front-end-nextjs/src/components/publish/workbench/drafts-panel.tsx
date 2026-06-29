"use client";
import {
  FilePenLine,
  FileText,
  Loader2,
  RotateCcw,
  Trash2,
  X
} from "lucide-react";
import {
  Button
} from "@/components/ui/button";
import { useTranslations } from "next-intl";
import { useLocale } from "next-intl";
import {
  richTextToPlainText
} from "@/lib/rich-text";
import type {
  FeedPost
} from "@/lib/types";
import {
  formatDraftDate,
  formatDraftLabel
} from "./post-builders";

export function DraftsPanel({
  actionId,
  drafts,
  loading,
  onClose,
  onDelete,
  onRefresh,
  onRestore,
}: {
  actionId: string | number | null;
  drafts: FeedPost[];
  loading: boolean;
  onClose: () => void;
  onDelete: (draftId: string | number) => void;
  onRefresh: () => void;
  onRestore: (draft: FeedPost) => void;
}) {
  const t = useTranslations("publish.workbench");
  const locale = useLocale();
  return (
    <section className="mb-5 rounded-2xl bg-white p-5 shadow-[0_8px_28px_rgba(0,0,0,0.04)]">
      <div className="flex flex-wrap items-center gap-3">
        <div className="min-w-0 flex-1">
          <h2 className="text-base font-semibold text-[#25252b]">{t("drafts")}</h2>
          <p className="mt-1 text-xs text-[#85858c]">{t("draftsHint")}</p>
        </div>
        <Button
          type="button"
          variant="outline"
          onClick={onRefresh}
          disabled={loading}
          className="h-9 border-[#e3e3e6] bg-white px-3 text-[#55555d]"
        >
          {loading ? <Loader2 className="size-4 animate-spin" /> : <RotateCcw className="size-4" />}
          {t("refresh")}
        </Button>
        <Button
          type="button"
          variant="ghost"
          onClick={onClose}
          className="size-9 p-0 text-[#777780]"
          aria-label={t("closeDrafts")}
        >
          <X className="size-4" />
        </Button>
      </div>

      {loading && drafts.length === 0 ? (
        <div className="mt-5 flex min-h-36 items-center justify-center rounded-2xl bg-[#fafafa] text-sm text-[#777780]">
          <Loader2 className="mr-2 size-4 animate-spin" />
          {t("loadingDrafts")}
        </div>
      ) : drafts.length > 0 ? (
        <div className="mt-5 grid grid-cols-[repeat(auto-fit,minmax(min(100%,260px),1fr))] gap-4">
          {drafts.map((draft) => {
            const busy = actionId === draft.id;
            const summary = richTextToPlainText(draft.content);
            const title = draft.title.trim() || summary || t("untitledDraft");

            return (
              <article
                key={draft.id}
                className="min-w-0 rounded-2xl border border-[#eeeeef] bg-[#fafafa] p-4"
              >
                <div className="flex items-start gap-3">
                  <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-white text-primary shadow-sm">
                    <FilePenLine className="size-5" />
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="shrink-0 rounded-full bg-white px-2 py-0.5 text-[11px] font-semibold text-[#777780]">
                        {t(formatDraftLabel(draft))}
                      </span>
                      <span className="truncate text-[11px] text-[#9a9aa1]">
                        {formatDraftDate(draft.created_at, locale) || t("justNow")}
                      </span>
                    </div>
                    <h3 className="mt-2 line-clamp-2 break-words text-sm font-semibold leading-5 text-[#25252b]">
                      {title}
                    </h3>
                    {summary ? (
                      <p className="mt-2 line-clamp-2 break-words text-xs leading-5 text-[#777780]">
                        {summary}
                      </p>
                    ) : null}
                    {draft.tags?.length ? (
                      <div className="mt-3 flex flex-wrap gap-1.5">
                        {draft.tags.slice(0, 4).map((tag) => (
                          <span
                            key={`${draft.id}-${tag.id}`}
                            className="max-w-full truncate rounded-full border border-[#e8e8eb] bg-white px-2 py-0.5 text-[11px] font-medium text-[#777780]"
                          >
                            #{tag.name}
                          </span>
                        ))}
                      </div>
                    ) : null}
                  </div>
                </div>

                <div className="mt-4 flex items-center gap-2">
                  <Button
                    type="button"
                    onClick={() => onRestore(draft)}
                    className="h-9 flex-1 px-3 text-xs"
                    disabled={busy}
                  >
                    <FilePenLine className="size-3.5" />
                    {t("continueEditing")}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    aria-label={t("deleteDraft")}
                    onClick={() => onDelete(draft.id)}
                    disabled={busy}
                    className="size-9 border-[#e3e3e6] bg-white p-0 text-[#777780] hover:text-primary"
                  >
                    {busy ? <Loader2 className="size-4 animate-spin" /> : <Trash2 className="size-4" />}
                  </Button>
                </div>
              </article>
            );
          })}
        </div>
      ) : (
        <div className="mt-5 flex min-h-36 flex-col items-center justify-center rounded-2xl bg-[#fafafa] px-6 text-center">
          <FileText className="size-8 text-[#c3c3c9]" />
          <p className="mt-3 text-sm font-semibold text-[#3c3c43]">{t("noDrafts")}</p>
          <p className="mt-1 text-xs text-[#85858c]">{t("noDraftsHint")}</p>
        </div>
      )}
    </section>
  );
}
