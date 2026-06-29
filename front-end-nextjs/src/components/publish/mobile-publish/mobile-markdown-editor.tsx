"use client";

import { postContentLength } from "@/lib/post-content";
import { insertMarkdownFromPaste } from "@/lib/rich-text";
import { cn } from "@/lib/utils";
import type { Editor } from "@tiptap/core";
import { Markdown } from "@tiptap/markdown";
import Placeholder from "@tiptap/extension-placeholder";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import {
  AtSign,
  Eye,
  FileCode2,
  Paperclip,
  Smile,
  Sparkles,
  type LucideIcon,
} from "lucide-react";
import type { useTranslations } from "next-intl";
import { useEffect, useLayoutEffect, useRef } from "react";
import type { MobileMarkdownEditorMode } from "./mobile-publish-config";

type Translate = ReturnType<typeof useTranslations>;

type MobileMarkdownEditorProps = {
  aiDisabled: boolean;
  limit: number;
  mode: MobileMarkdownEditorMode;
  onChange: (value: string) => void;
  onModeChange: (mode: MobileMarkdownEditorMode) => void;
  onOpenAIFormat: () => void;
  onOpenAttachment: () => void;
  onOpenEmoji: () => void;
  onOpenMention: () => void;
  placeholder: string;
  t: Translate;
  value: string;
};

type ToolbarButtonProps = {
  active?: boolean;
  disabled?: boolean;
  icon: LucideIcon;
  label: string;
  onClick: () => void;
  testId?: string;
};

export function MobileMarkdownEditor({
  aiDisabled,
  limit,
  mode,
  onChange,
  onModeChange,
  onOpenAIFormat,
  onOpenAttachment,
  onOpenEmoji,
  onOpenMention,
  placeholder,
  t,
  value,
}: MobileMarkdownEditorProps) {
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const editorRef = useRef<Editor | null>(null);
  const characterCount = postContentLength(value);
  const modeToggleLabel = t(
    mode === "live"
      ? "publish.mobile.markdownToolbar.switchToSource"
      : "publish.mobile.markdownToolbar.switchToLive",
  );
  const editor = useEditor({
    immediatelyRender: false,
    extensions: [
      Markdown.configure({
        markedOptions: { breaks: true, gfm: true },
      }),
      StarterKit.configure({
        heading: {
          levels: [1, 2, 3],
        },
      }),
      Placeholder.configure({
        placeholder,
      }),
    ],
    content: value,
    contentType: "markdown",
    editorProps: {
      attributes: {
        class:
          "min-h-[222px] px-4 py-4 text-[16px] leading-7 text-[var(--mobile-publish-input)] outline-none",
      },
      handlePaste: (_view, event) => {
        const currentEditor = editorRef.current;
        return currentEditor ? insertMarkdownFromPaste(currentEditor, event) : false;
      },
    },
    onUpdate: ({ editor: nextEditor }) => {
      const nextText = nextEditor.getText();

      onChange(nextText.trim() ? nextEditor.getMarkdown() : "");
    },
  });

  useLayoutEffect(() => {
    if (mode !== "source") {
      return;
    }

    const textarea = textareaRef.current;
    if (!textarea) {
      return;
    }

    textarea.focus();
    const cursor = textarea.value.length;
    textarea.setSelectionRange(cursor, cursor);
  }, [mode]);

  useEffect(() => {
    editorRef.current = editor;
    return () => {
      if (editorRef.current === editor) {
        editorRef.current = null;
      }
    };
  }, [editor]);

  useEffect(() => {
    if (!editor) {
      return;
    }

    const currentMarkdown = editor.getText().trim() ? editor.getMarkdown() : "";
    if (currentMarkdown === value) {
      return;
    }

    editor.commands.setContent(value, { emitUpdate: false, contentType: "markdown" });
  }, [editor, value]);

  return (
    <div className="mobile-markdown-editor mt-4 flex min-h-[248px] flex-col overflow-hidden rounded-[14px] bg-[var(--mobile-publish-card)] shadow-[var(--mobile-publish-shadow)]">
      {mode === "live" ? (
        <EditorContent
          aria-label={t("publish.mobile.markdownToolbar.liveInput")}
          className="mobile-markdown-live-input min-h-[222px] flex-1 overflow-x-hidden overflow-y-auto"
          data-testid="mobile-markdown-live-input"
          editor={editor}
        />
      ) : (
        <textarea
          ref={textareaRef}
          aria-label={t("publish.mobile.markdownToolbar.sourceInput")}
          data-testid="mobile-markdown-source-input"
          value={value}
          onChange={(event) => onChange(event.target.value)}
          placeholder={placeholder}
          className="min-h-[222px] w-full flex-1 resize-none bg-transparent px-4 py-4 font-mono text-[15px] leading-7 text-[var(--mobile-publish-input)] outline-none placeholder:text-[var(--mobile-publish-subtle)]"
        />
      )}

      <div className="mobile-markdown-toolbar flex h-[50px] shrink-0 items-center gap-[clamp(0.45rem,2.7vw,1rem)] overflow-x-auto overscroll-x-contain px-4 text-[var(--mobile-publish-muted)]">
        <ToolbarButton
          active={mode === "source"}
          icon={mode === "live" ? FileCode2 : Eye}
          label={modeToggleLabel}
          onClick={() => onModeChange(mode === "live" ? "source" : "live")}
          testId="mobile-markdown-mode-toggle"
        />
        <ToolbarButton
          disabled={aiDisabled}
          icon={Sparkles}
          label={t("publish.aiFormat.trigger")}
          onClick={onOpenAIFormat}
        />
        <ToolbarButton
          icon={Smile}
          label={t("publish.mobile.addEmoji")}
          onClick={onOpenEmoji}
        />
        <ToolbarButton
          icon={AtSign}
          label={t("publish.mobile.mentionUser")}
          onClick={onOpenMention}
        />
        <ToolbarButton
          icon={Paperclip}
          label={t("publish.mobile.addAttachment")}
          onClick={onOpenAttachment}
        />
        <span
          className={cn(
            "ml-auto shrink-0 text-[clamp(0.82rem,3.8vw,1rem)] font-semibold text-[var(--mobile-publish-muted)]",
            characterCount > limit && "text-[var(--mobile-publish-accent-strong)]",
          )}
        >
          {characterCount}/{limit}
        </span>
      </div>
    </div>
  );
}

function ToolbarButton({
  active,
  disabled,
  icon: Icon,
  label,
  onClick,
  testId,
}: ToolbarButtonProps) {
  return (
    <button
      type="button"
      aria-label={label}
      data-testid={testId}
      disabled={disabled}
      title={label}
      onClick={onClick}
      className={cn(
        "flex size-8 shrink-0 items-center justify-center rounded-full transition-colors active:bg-[var(--mobile-publish-accent-soft)] disabled:opacity-40",
        active && "bg-[var(--mobile-publish-accent-soft)] text-[var(--mobile-publish-accent-strong)]",
      )}
    >
      <Icon className="size-[clamp(1rem,5vw,1.45rem)]" />
    </button>
  );
}
