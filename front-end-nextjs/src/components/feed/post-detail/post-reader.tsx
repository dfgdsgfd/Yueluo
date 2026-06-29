"use client";

import { AlignJustify, BookOpen, ListTree, Type, X } from "lucide-react";
import { useTranslations } from "next-intl";
import { useEffect, useMemo, useRef, useState, type CSSProperties, type RefObject } from "react";
import type { MarkdownHeading } from "@/lib/markdown";
import { richTextToPlainText } from "@/lib/rich-text";
import { cn } from "@/lib/utils";

const readerPreferencesKey = "yuem_post_reader_preferences_v1";
const readerProgressKey = "yuem_post_reader_progress_v1";
const readerAutoThreshold = 1500;
const fontSizes = ["0.95rem", "1.05rem", "1.16rem"] as const;
const lineHeights = [1.72, 1.9, 2.08] as const;

type ReaderPreferences = { fontIndex: number; lineIndex: number };
type ReaderProgressEntry = { progress: number; updatedAt: number };

export function usePostReader<T extends HTMLElement = HTMLElement>({
  autoEnter = true,
  content,
  postId,
  scrollRef,
}: {
  autoEnter?: boolean;
  content: string | null;
  postId: string;
  scrollRef: RefObject<T | null>;
}) {
  const visibleLength = useMemo(
    () => Array.from(richTextToPlainText(content)).length,
    [content],
  );
  const [headings, setHeadings] = useState<MarkdownHeading[]>([]);
  const [readingOverride, setReadingOverride] = useState<{ postId: string; value: boolean } | null>(null);
  const reading = readingOverride?.postId === postId
    ? readingOverride.value
    : autoEnter && visibleLength >= readerAutoThreshold;
  const [progress, setProgress] = useState(0);
  const [preferences, setPreferences] = useState<ReaderPreferences>({ fontIndex: 1, lineIndex: 1 });
  const restoredPostRef = useRef<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    const source = content?.trim();
    if (!source) {
      queueMicrotask(() => {
        if (!cancelled) {
          setHeadings([]);
        }
      });
      return () => {
        cancelled = true;
      };
    }

    import("@/lib/markdown")
      .then(({ renderRichTextDocument }) => {
        if (!cancelled) {
          setHeadings(renderRichTextDocument(source).headings.filter((heading) => heading.level <= 3));
        }
      })
      .catch(() => {
        if (!cancelled) {
          setHeadings([]);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [content]);

  useEffect(() => {
    let cancelled = false;
    try {
      const saved = JSON.parse(window.localStorage.getItem(readerPreferencesKey) ?? "null") as Partial<ReaderPreferences> | null;
      if (saved) {
        queueMicrotask(() => {
          if (!cancelled) {
            setPreferences({
              fontIndex: clampIndex(saved.fontIndex, fontSizes.length, 1),
              lineIndex: clampIndex(saved.lineIndex, lineHeights.length, 1),
            });
          }
        });
      }
    } catch {
      // Invalid legacy preferences fall back to the comfortable defaults.
    }
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!reading) return;
    const element = scrollRef.current;
    if (!element) return;
    let frame = 0;
    const update = () => {
      frame = 0;
      const max = Math.max(0, element.scrollHeight - element.clientHeight);
      const nextProgress = max > 0 ? Math.min(1, Math.max(0, element.scrollTop / max)) : 0;
      setProgress(nextProgress);
      saveReaderProgress(postId, nextProgress);
    };
    const onScroll = () => {
      if (!frame) frame = window.requestAnimationFrame(update);
    };
    element.addEventListener("scroll", onScroll, { passive: true });
    if (restoredPostRef.current !== postId) {
      restoredPostRef.current = postId;
      window.requestAnimationFrame(() => {
        const saved = readReaderProgress(postId);
        const max = Math.max(0, element.scrollHeight - element.clientHeight);
        if (saved > 0.02 && saved < 0.98 && max > 0) {
          element.scrollTop = Math.round(max * saved);
        }
        update();
      });
    } else {
      update();
    }
    return () => {
      element.removeEventListener("scroll", onScroll);
      if (frame) window.cancelAnimationFrame(frame);
    };
  }, [postId, reading, scrollRef]);

  function updatePreferences(next: ReaderPreferences) {
    setPreferences(next);
    window.localStorage.setItem(readerPreferencesKey, JSON.stringify(next));
  }

  return {
    canRead: Boolean(content?.trim()),
    headings,
    progress,
    reading,
    readerStyle: {
      "--post-reader-font-size": fontSizes[preferences.fontIndex],
      "--post-reader-line-height": lineHeights[preferences.lineIndex],
    } as CSSProperties,
    setFontIndex: (fontIndex: number) => updatePreferences({ ...preferences, fontIndex }),
    setLineIndex: (lineIndex: number) => updatePreferences({ ...preferences, lineIndex }),
    setReading: (value: boolean) => setReadingOverride({ postId, value }),
    preferences,
  };
}

export function PostReaderToolbar({
  headings,
  onExit,
  onFontIndexChange,
  onLineIndexChange,
  preferences,
  progress,
}: {
  headings: MarkdownHeading[];
  onExit: () => void;
  onFontIndexChange: (index: number) => void;
  onLineIndexChange: (index: number) => void;
  preferences: ReaderPreferences;
  progress: number;
}) {
  const t = useTranslations("drawer.reader");
  const [tocOpen, setTocOpen] = useState(false);

  function goToHeading(id: string) {
    const destination = document.getElementById(id);
    destination?.scrollIntoView({ behavior: "smooth", block: "start" });
    setTocOpen(false);
  }

  return (
    <>
      <div className="post-reader-toolbar sticky top-0 z-30 mx-auto flex min-h-12 w-full max-w-[760px] items-center gap-2 border-b border-white/10 bg-[#121212]/92 px-3 py-2 backdrop-blur md:rounded-t-2xl">
        <BookOpen className="size-4 shrink-0 text-primary" />
        <span className="min-w-0 flex-1 truncate text-xs font-semibold text-white/72">
          {t("title")}
        </span>
        {headings.length > 0 ? (
          <button type="button" onClick={() => setTocOpen(true)} className="reader-control-button" aria-label={t("toc")}>
            <ListTree className="size-4" />
          </button>
        ) : null}
        <button
          type="button"
          onClick={() => onFontIndexChange((preferences.fontIndex + 1) % fontSizes.length)}
          className="reader-control-button"
          aria-label={t("fontSize")}
        >
          <Type className="size-4" />
        </button>
        <button
          type="button"
          onClick={() => onLineIndexChange((preferences.lineIndex + 1) % lineHeights.length)}
          className="reader-control-button"
          aria-label={t("lineHeight")}
        >
          <AlignJustify className="size-4" />
        </button>
        <button type="button" onClick={onExit} className="reader-control-button" aria-label={t("exit")}>
          <X className="size-4" />
        </button>
        <span className="post-reader-progress-track absolute inset-x-0 bottom-0 h-0.5">
          <span className="block h-full bg-primary transition-[width] duration-150" style={{ width: `${Math.round(progress * 100)}%` }} />
        </span>
      </div>

      {tocOpen ? (
        <div className="fixed inset-0 z-[90] flex items-end justify-center md:items-center" onClick={() => setTocOpen(false)}>
          <div className="absolute inset-0 bg-black/60" />
          <section className="relative max-h-[72dvh] w-full max-w-[430px] overflow-hidden rounded-t-3xl border border-white/10 bg-[#18181c] p-5 pb-[calc(1.25rem+env(safe-area-inset-bottom))] text-white shadow-2xl md:rounded-3xl">
            <div className="flex items-center justify-between">
              <h2 className="font-black">{t("toc")}</h2>
              <button type="button" onClick={() => setTocOpen(false)} className="reader-control-button" aria-label={t("closeToc")}>
                <X className="size-4" />
              </button>
            </div>
            <nav className="mt-4 max-h-[56dvh] space-y-1 overflow-y-auto">
              {headings.map((heading) => (
                <button
                  key={heading.id}
                  type="button"
                  onClick={() => goToHeading(heading.id)}
                  className={cn(
                    "block w-full rounded-xl px-3 py-2 text-left text-sm text-white/72 hover:bg-white/8 hover:text-white",
                    heading.level === 2 && "pl-6",
                    heading.level === 3 && "pl-9 text-xs",
                  )}
                >
                  {heading.text}
                </button>
              ))}
            </nav>
          </section>
        </div>
      ) : null}
    </>
  );
}

function clampIndex(value: unknown, length: number, fallback: number) {
  const parsed = Number(value);
  return Number.isInteger(parsed) && parsed >= 0 && parsed < length ? parsed : fallback;
}

function readReaderProgress(postId: string) {
  try {
    const entries = JSON.parse(window.localStorage.getItem(readerProgressKey) ?? "{}") as Record<string, ReaderProgressEntry>;
    return Number(entries[postId]?.progress) || 0;
  } catch {
    return 0;
  }
}

function saveReaderProgress(postId: string, progress: number) {
  try {
    const entries = JSON.parse(window.localStorage.getItem(readerProgressKey) ?? "{}") as Record<string, ReaderProgressEntry>;
    entries[postId] = { progress, updatedAt: Date.now() };
    const limited = Object.fromEntries(
      Object.entries(entries)
        .sort(([, left], [, right]) => right.updatedAt - left.updatedAt)
        .slice(0, 50),
    );
    window.localStorage.setItem(readerProgressKey, JSON.stringify(limited));
  } catch {
    // Storage can be unavailable in private browsing; reading still works.
  }
}
