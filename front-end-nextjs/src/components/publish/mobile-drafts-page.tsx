"use client";

import { useEffect, useMemo, useState } from "react";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { ChevronLeft, FileText, ImageIcon, Paperclip, Play, Trash2 } from "lucide-react";
import { Masonry } from "react-plock";
import { toast } from "sonner";
import {
  deleteLocalDraft,
  formatDraftTime,
  getDraftTitle,
  readLocalDrafts,
  translateCategoryName,
  writePendingRestoreDraft,
  type MobilePublishDraft,
} from "./mobile-drafts";

export function MobileDraftsPage() {
  const router = useRouter();
  const t = useTranslations("publish.mobile");
  const [drafts, setDrafts] = useState<MobilePublishDraft[]>([]);

  useEffect(() => {
    let cancelled = false;
    void readLocalDrafts()
      .then((list) => {
        if (!cancelled) {
          setDrafts(list);
        }
      })
      .catch(() => {
        if (!cancelled) {
          toast.error(t("draftReadFailed"));
        }
      });

    return () => {
      cancelled = true;
    };
  }, [t]);

  function restoreDraft(draft: MobilePublishDraft) {
    writePendingRestoreDraft(draft);
    router.push("/publish/mobile");
  }

  async function deleteDraft(id: string) {
    await deleteLocalDraft(id);
    setDrafts((currentDrafts) => currentDrafts.filter((draft) => draft.id !== id));
    toast.success(t("draftDeleted"));
  }

  return (
    <main className="mobile-publish-page min-h-dvh bg-[var(--mobile-publish-bg)] text-[var(--mobile-publish-text)]">
      <div className="mx-auto flex min-h-dvh w-full max-w-[430px] flex-col bg-[var(--mobile-publish-surface)] max-[430px]:max-w-none">
        <header className="sticky top-0 z-20 flex h-18 shrink-0 items-center gap-3 border-b border-[var(--mobile-publish-border-soft)] bg-[var(--mobile-publish-surface)] px-4 pt-[env(safe-area-inset-top)]">
          <Link
            href="/publish/mobile"
            aria-label={t("backToPublish")}
            className="flex size-11 shrink-0 items-center justify-center rounded-full text-[var(--mobile-publish-text)] transition-colors active:bg-[var(--mobile-publish-accent-soft)]"
          >
            <ChevronLeft className="size-8" strokeWidth={2.2} />
          </Link>
          <h1 className="min-w-0 flex-1 truncate text-center text-[24px] font-black leading-none text-[var(--mobile-publish-heading)]">
            {t("drafts")}
          </h1>
          <span aria-hidden="true" className="size-11 shrink-0" />
        </header>

        <section className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 pb-[calc(28px+env(safe-area-inset-bottom))] pt-4">
          {drafts.length > 0 ? (
            <Masonry
              items={drafts}
              config={{
                columns: [2],
                gap: [10],
                media: [320],
                useBalancedLayout: true,
              }}
              render={(draft) => (
                <DraftCard
                  key={draft.id}
                  draft={draft}
                  onDelete={() => void deleteDraft(draft.id)}
                  onRestore={() => restoreDraft(draft)}
                />
              )}
            />
          ) : (
            <div className="flex min-h-[60dvh] flex-col items-center justify-center px-8 text-center">
              <span className="flex size-16 items-center justify-center rounded-full bg-[var(--mobile-publish-card)] text-[var(--mobile-publish-accent-strong)] shadow-[var(--mobile-publish-shadow)]">
                <FileText className="size-8" />
              </span>
              <p className="mt-4 text-[17px] font-bold text-[var(--mobile-publish-heading)]">{t("noDrafts")}</p>
              <p className="mt-2 text-[14px] leading-6 text-[var(--mobile-publish-muted)]">
                {t("noDraftsHint")}
              </p>
            </div>
          )}
        </section>
      </div>
    </main>
  );
}

function DraftCard({
  draft,
  onDelete,
  onRestore,
}: {
  draft: MobilePublishDraft;
  onDelete: () => void;
  onRestore: () => void;
}) {
  const t = useTranslations("publish.mobile");
  const coverUrl = useMemo(() => getDraftCoverUrl(draft), [draft]);
  const hasVideo = draft.mediaAssets.some((item) => item.kind === "video");

  useEffect(() => {
    return () => {
      if (coverUrl?.startsWith("blob:")) {
        URL.revokeObjectURL(coverUrl);
      }
    };
  }, [coverUrl]);

  return (
    <article className="overflow-hidden rounded-[14px] bg-[var(--mobile-publish-card)] p-3 shadow-[var(--mobile-publish-shadow)]">
      <button type="button" onClick={onRestore} className="block w-full text-left">
        {coverUrl ? (
          <span className="relative mb-3 block aspect-square overflow-hidden rounded-[12px] bg-[var(--mobile-publish-card-strong)]">
            <Image src={coverUrl} alt={getDraftTitle(draft, t("untitledDraft"))} fill unoptimized sizes="180px" className="object-cover" />
            {hasVideo ? (
              <span className="absolute inset-0 flex items-center justify-center bg-black/20 text-white">
                <Play className="size-8 fill-white" />
              </span>
            ) : null}
          </span>
        ) : null}
        <p className="line-clamp-2 break-words text-[15px] font-bold leading-5 text-[var(--mobile-publish-input)]">
          {getDraftTitle(draft, t("untitledDraft"))}
        </p>
        {draft.body.trim() ? (
          <p className="mt-2 line-clamp-5 break-words text-[13px] leading-5 text-[var(--mobile-publish-muted)]">
            {draft.body.trim()}
          </p>
        ) : null}
        <div className="mt-3 flex flex-wrap gap-1.5">
          {draft.board.trim() ? (
            <span className="max-w-full truncate rounded-full bg-[var(--mobile-publish-accent-soft)] px-2 py-1 text-[11px] font-semibold text-[var(--mobile-publish-accent-strong)]">
              {translateCategoryName(draft.board)}
            </span>
          ) : null}
          {draft.topic.trim() ? (
            <span className="max-w-full truncate rounded-full border border-[var(--mobile-publish-border)] px-2 py-1 text-[11px] font-semibold text-[var(--mobile-publish-muted)]">
              #{draft.topic.trim().replace(/^#/, "")}
            </span>
          ) : null}
          {draft.mediaAssets.length > 0 ? (
            <span className="inline-flex max-w-full items-center gap-1 rounded-full border border-[var(--mobile-publish-border)] px-2 py-1 text-[11px] font-semibold text-[var(--mobile-publish-muted)]">
              <ImageIcon className="size-3" />
              {draft.mediaAssets.length}
            </span>
          ) : null}
          {draft.attachment ? (
            <span className="inline-flex max-w-full items-center gap-1 rounded-full border border-[var(--mobile-publish-border)] px-2 py-1 text-[11px] font-semibold text-[var(--mobile-publish-muted)]">
              <Paperclip className="size-3" />
              {t("attachment")}
            </span>
          ) : null}
        </div>
        <p className="mt-3 text-[11px] font-medium leading-4 text-[var(--mobile-publish-subtle)]">
          {formatDraftTime(draft.savedAt)}
        </p>
      </button>
      <button
        type="button"
        aria-label={t("deleteDraft")}
        onClick={onDelete}
        className="mt-3 flex h-8 w-full items-center justify-center gap-1.5 rounded-full border border-[var(--mobile-publish-border-soft)] text-[12px] font-semibold text-[var(--mobile-publish-muted)] active:bg-[var(--mobile-publish-accent-soft)]"
      >
        <Trash2 className="size-3.5" />
        {t("delete")}
      </button>
    </article>
  );
}

function getDraftCoverUrl(draft: MobilePublishDraft) {
  const coverMedia = draft.mediaAssets.find((item) => item.previewDataUrl || item.kind === "image");
  if (!coverMedia) {
    return null;
  }

  return coverMedia.previewDataUrl || URL.createObjectURL(coverMedia.file);
}
