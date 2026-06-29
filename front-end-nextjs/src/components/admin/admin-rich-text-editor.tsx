"use client";

import { forwardRef, useCallback, useEffect, useImperativeHandle, useMemo, useRef, useState } from "react";
import type { Editor } from "@tiptap/core";
import Emoji from "@tiptap/extension-emoji";
import FileHandler from "@tiptap/extension-file-handler";
import Highlight from "@tiptap/extension-highlight";
import Image from "@tiptap/extension-image";
import Link from "@tiptap/extension-link";
import Mention from "@tiptap/extension-mention";
import Placeholder from "@tiptap/extension-placeholder";
import { Table, TableCell, TableHeader, TableRow } from "@tiptap/extension-table";
import { TaskItem } from "@tiptap/extension-task-item";
import { TaskList } from "@tiptap/extension-task-list";
import TextAlign from "@tiptap/extension-text-align";
import { BackgroundColor, Color, TextStyle } from "@tiptap/extension-text-style";
import { Typography } from "@tiptap/extension-typography";
import Underline from "@tiptap/extension-underline";
import { Markdown } from "@tiptap/markdown";
import { EditorContent, ReactRenderer, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import type { SuggestionOptions } from "@tiptap/suggestion";
import { useTranslations } from "next-intl";
import {
  AlignCenter,
  AlignLeft,
  AlignRight,
  AtSign,
  Bold,
  CheckSquare,
  FileUp,
  Highlighter,
  ImagePlus,
  Italic,
  List,
  ListOrdered,
  PaintBucket,
  Palette,
  Smile,
  Table as TableIcon,
  Underline as UnderlineIcon,
} from "lucide-react";
import tippy, { type Instance as TippyInstance } from "tippy.js";
import { toast } from "sonner";
import { MarkdownContent } from "@/components/markdown-content";
import { getAdminList, uploadAttachment, uploadImage } from "@/lib/api";
import { inferRichTextContentType, insertMarkdownFromPaste, isRichTextHtml } from "@/lib/rich-text";
import { cn } from "@/lib/utils";

type MentionItem = {
  id: string;
  label: string;
};

type MentionListHandle = {
  onKeyDown: (props: { event: KeyboardEvent }) => boolean;
};

type MentionListProps = {
  command: (item: MentionItem) => void;
  items: MentionItem[];
};

const colors = ["#17171d", "#dc2626", "#d97706", "#16a34a", "#2563eb", "#7c3aed"];
const backgrounds = ["#fff3a3", "#dbeafe", "#dcfce7", "#fee2e2", "#f3e8ff", "#f1f5f9"];

function mentionSuggestion(token: string) {
  const suggestion = {
    char: "@",
    items: async ({ query }: { query: string }) => {
      const keyword = query.trim();
      try {
        const payload = await getAdminList("users", { page: 1, limit: 8, keyword }, token);
        return payload.items.map((row) => ({
          id: String(row.id ?? row.user_id ?? ""),
          label: String(row.nickname ?? row.user_id ?? row.id ?? "用户"),
        })).filter((item) => item.id && item.label);
      } catch {
        return [];
      }
    },
    render: () => {
      let component: ReactRenderer<MentionListHandle, MentionListProps> | null = null;
      let popup: TippyInstance[] | null = null;
      return {
        onStart: (props: Record<string, unknown>) => {
          component = new ReactRenderer(MentionList, {
            props: props as MentionListProps,
            editor: props.editor as Editor,
          });
          const rect = props.clientRect as (() => DOMRect | null) | undefined;
          popup = tippy("body", {
            appendTo: () => document.body,
            content: component.element,
            getReferenceClientRect: () => rect?.() ?? new DOMRect(),
            interactive: true,
            placement: "bottom-start",
            showOnCreate: true,
            trigger: "manual",
          });
        },
        onUpdate(props: Record<string, unknown>) {
          component?.updateProps(props as MentionListProps);
          const rect = props.clientRect as (() => DOMRect | null) | undefined;
          popup?.[0]?.setProps({ getReferenceClientRect: () => rect?.() ?? new DOMRect() });
        },
        onKeyDown(props: { event: KeyboardEvent }) {
          if (props.event.key === "Escape") {
            popup?.[0]?.hide();
            return true;
          }
          return component?.ref?.onKeyDown(props) ?? false;
        },
        onExit() {
          popup?.[0]?.destroy();
          component?.destroy();
        },
      };
    },
  };
  return suggestion as unknown as Omit<SuggestionOptions<MentionItem>, "editor">;
}

export function AdminRichTextEditor({
  disabled,
  onChange,
  placeholder,
  token,
  value,
}: {
  disabled?: boolean;
  onChange: (value: string) => void;
  placeholder?: string;
  token: string;
  value: string;
}) {
  const t = useTranslations("publish.editor.richText");
  const [uploading, setUploading] = useState(false);
  const [editingMode, setEditingMode] = useState<"visual" | "markdown" | "preview">("visual");
  const editorRef = useRef<Editor | null>(null);
  const editor = useEditor({
    immediatelyRender: false,
    extensions: [
      Markdown.configure({
        markedOptions: { breaks: true, gfm: true },
      }),
      StarterKit.configure({
        heading: { levels: [1, 2, 3] },
      }),
      TextStyle,
      BackgroundColor,
      Color,
      Underline,
      Highlight.configure({ multicolor: true }),
      Link.configure({
        autolink: true,
        openOnClick: false,
      }),
      TextAlign.configure({ types: ["heading", "paragraph"] }),
      Placeholder.configure({
        placeholder: placeholder ?? "",
      }),
      Typography,
      Emoji,
      Image.configure({ allowBase64: false }),
      Table.configure({ resizable: true }),
      TableRow,
      TableHeader,
      TableCell,
      TaskList,
      TaskItem.configure({ nested: true }),
      Mention.configure({
        HTMLAttributes: { class: "admin-rich-mention" },
        suggestion: mentionSuggestion(token),
      }),
      FileHandler.configure({
        allowedMimeTypes: ["image/png", "image/jpeg", "image/gif", "image/webp", "application/pdf", "text/plain", "application/zip"],
        onDrop: (currentEditor, files) => {
          void insertFiles(currentEditor, files, token, setUploading);
        },
        onPaste: (currentEditor, files) => {
          void insertFiles(currentEditor, files, token, setUploading);
        },
      }),
    ],
    editable: !disabled,
    content: value || "",
    contentType: inferRichTextContentType(value),
    editorProps: {
      attributes: {
        class: "admin-rich-prosemirror min-h-[220px] px-4 py-3 text-sm leading-7 text-[#252932] outline-none",
      },
      handlePaste: (_view, event) => {
        const currentEditor = editorRef.current;
        return currentEditor ? insertMarkdownFromPaste(currentEditor, event) : false;
      },
    },
    onUpdate: ({ editor: nextEditor }) => onChange(nextEditor.getMarkdown()),
  });

  useEffect(() => {
    editorRef.current = editor;
    return () => {
      if (editorRef.current === editor) {
        editorRef.current = null;
      }
    };
  }, [editor]);

  useEffect(() => {
    if (!editor) return;
    if (editor.getHTML() === (value || "")) {
      if (isRichTextHtml(value)) {
        onChange(editor.getMarkdown());
      }
      return;
    }
    editor.commands.setContent(value || "", { emitUpdate: false, contentType: inferRichTextContentType(value) });
    if (isRichTextHtml(value)) {
      onChange(editor.getMarkdown());
    }
  }, [editor, onChange, value]);

  const canEdit = Boolean(editor && !disabled);

  function changeEditingMode(mode: "visual" | "markdown" | "preview") {
    if (mode === "markdown" && editor && isRichTextHtml(value)) {
      onChange(editor.getMarkdown());
    }
    setEditingMode(mode);
  }

  return (
    <div className={cn("overflow-hidden rounded-lg border border-black/[0.08] bg-white", disabled && "opacity-70")}>
      <div className="flex flex-wrap items-center gap-1 border-b border-black/[0.06] bg-[#fafbfe] p-2">
        <div className="mr-2 grid grid-cols-3 rounded-lg bg-black/[0.04] p-1 text-xs font-semibold">
          {(["visual", "markdown", "preview"] as const).map((mode) => (
            <button
              key={mode}
              type="button"
              onClick={() => changeEditingMode(mode)}
              className={cn("h-7 rounded-md px-2", editingMode === mode ? "bg-white text-[#1d4ed8] shadow-sm" : "text-[#687080]")}
            >
              {t(mode)}
            </button>
          ))}
        </div>
        {editingMode === "visual" ? (
          <>
        <ToolbarButton editor={editor} disabled={!canEdit} label="粗体" active="bold" onClick={(item) => item.chain().focus().toggleBold().run()} icon={Bold} />
        <ToolbarButton editor={editor} disabled={!canEdit} label="斜体" active="italic" onClick={(item) => item.chain().focus().toggleItalic().run()} icon={Italic} />
        <ToolbarButton editor={editor} disabled={!canEdit} label="下划线" active="underline" onClick={(item) => item.chain().focus().toggleUnderline().run()} icon={UnderlineIcon} />
        <ToolbarButton editor={editor} disabled={!canEdit} label="无序列表" active="bulletList" onClick={(item) => item.chain().focus().toggleBulletList().run()} icon={List} />
        <ToolbarButton editor={editor} disabled={!canEdit} label="有序列表" active="orderedList" onClick={(item) => item.chain().focus().toggleOrderedList().run()} icon={ListOrdered} />
        <ToolbarButton editor={editor} disabled={!canEdit} label="任务列表" active="taskList" onClick={(item) => item.chain().focus().toggleTaskList().run()} icon={CheckSquare} />
        <ToolbarButton editor={editor} disabled={!canEdit} label="左对齐" onClick={(item) => item.chain().focus().setTextAlign("left").run()} icon={AlignLeft} />
        <ToolbarButton editor={editor} disabled={!canEdit} label="居中" onClick={(item) => item.chain().focus().setTextAlign("center").run()} icon={AlignCenter} />
        <ToolbarButton editor={editor} disabled={!canEdit} label="右对齐" onClick={(item) => item.chain().focus().setTextAlign("right").run()} icon={AlignRight} />
        <ToolbarButton editor={editor} disabled={!canEdit} label="表格" onClick={(item) => item.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()} icon={TableIcon} />
        <ToolbarButton editor={editor} disabled={!canEdit} label="@提及" onClick={(item) => item.chain().focus().insertContent("@").run()} icon={AtSign} />
        <ToolbarButton editor={editor} disabled={!canEdit} label="表情" onClick={(item) => item.chain().focus().insertContent("🙂").run()} icon={Smile} />
        <label className={buttonClassName(!canEdit || uploading)} title="插入图片">
          <ImagePlus className="size-4" />
          <input className="sr-only" type="file" accept="image/*" disabled={!canEdit || uploading} onChange={(event) => void insertInputFile(editor, event.currentTarget, token, setUploading)} />
        </label>
        <label className={buttonClassName(!canEdit || uploading)} title="插入文件">
          <FileUp className="size-4" />
          <input className="sr-only" type="file" disabled={!canEdit || uploading} onChange={(event) => void insertInputFile(editor, event.currentTarget, token, setUploading)} />
        </label>
        <ColorSwatches icon={Palette} colors={colors} disabled={!canEdit} onPick={(color) => editor?.chain().focus().setColor(color).run()} />
        <ColorSwatches icon={PaintBucket} colors={backgrounds} disabled={!canEdit} onPick={(color) => editor?.chain().focus().setBackgroundColor(color).run()} />
        <ToolbarButton editor={editor} disabled={!canEdit} label="高亮" active="highlight" onClick={(item) => item.chain().focus().toggleHighlight({ color: "#fff3a3" }).run()} icon={Highlighter} />
          </>
        ) : null}
      </div>
      {editingMode === "visual" ? (
        <EditorContent editor={editor} />
      ) : editingMode === "markdown" ? (
        <textarea
          aria-label={t("markdown")}
          className="min-h-[260px] w-full resize-y bg-[#17171d] px-4 py-3 font-mono text-sm leading-6 text-white outline-none"
          disabled={disabled}
          onChange={(event) => onChange(event.target.value)}
          placeholder={placeholder}
          value={value}
        />
      ) : (
        <MarkdownContent className="markdown-editor-preview min-h-[260px] px-4 py-3 text-sm leading-7 text-[#252932]" content={value} />
      )}
      <div className="border-t border-black/[0.06] px-3 py-2 text-xs text-[#7b808c]">
        {uploading ? "正在上传并插入文件..." : "支持拖拽图片/文件、表格、待办、@提及、颜色和高亮。"}
      </div>
    </div>
  );
}

function ToolbarButton({
  active,
  disabled,
  editor,
  icon: Icon,
  label,
  onClick,
}: {
  active?: string;
  disabled?: boolean;
  editor: Editor | null;
  icon: typeof Bold;
  label: string;
  onClick: (editor: Editor) => void;
}) {
  const isActive = active && editor?.isActive(active);
  return (
    <button type="button" aria-label={label} title={label} disabled={disabled} onClick={() => editor && onClick(editor)} className={cn(buttonClassName(disabled), isActive && "bg-[#e8f0ff] text-[#1d4ed8]")}>
      <Icon className="size-4" />
    </button>
  );
}

function ColorSwatches({ colors: values, disabled, icon: Icon, onPick }: { colors: string[]; disabled?: boolean; icon: typeof Palette; onPick: (value: string) => void }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="relative">
      <button type="button" aria-label="颜色" title="颜色" disabled={disabled} onClick={() => setOpen((current) => !current)} className={buttonClassName(disabled)}>
        <Icon className="size-4" />
      </button>
      {open ? (
        <div className="absolute left-0 top-9 z-20 grid grid-cols-3 gap-1 rounded-lg border border-black/[0.08] bg-white p-2 shadow-lg">
          {values.map((color) => (
            <button key={color} type="button" aria-label={color} className="size-6 rounded border border-black/[0.08]" style={{ backgroundColor: color }} onClick={() => { onPick(color); setOpen(false); }} />
          ))}
        </div>
      ) : null}
    </div>
  );
}

const MentionList = forwardRef<MentionListHandle, MentionListProps>(function MentionList({ command, items }, ref) {
  const [selectedIndex, setSelectedIndex] = useState(0);
  const activeIndex = items.length ? Math.min(selectedIndex, items.length - 1) : 0;
  const selected = useMemo(() => items[activeIndex], [items, activeIndex]);
  const selectItem = useCallback((index: number) => {
    const item = items[index];
    if (item) {
      command(item);
    }
  }, [command, items]);

  useImperativeHandle(ref, () => ({
    onKeyDown: ({ event }) => {
      if (!items.length) return false;
      if (event.key === "ArrowUp") {
        setSelectedIndex((current) => (current + items.length - 1) % items.length);
        return true;
      }
      if (event.key === "ArrowDown") {
        setSelectedIndex((current) => (current + 1) % items.length);
        return true;
      }
      if (event.key === "Enter") {
        selectItem(activeIndex);
        return true;
      }
      return false;
    },
  }), [activeIndex, items, selectItem]);

  return (
    <div className="min-w-[180px] overflow-hidden rounded-lg border border-black/[0.08] bg-white p-1 text-sm shadow-lg">
      {items.length ? items.map((item, index) => (
        <button key={item.id} type="button" className={cn("block w-full rounded-md px-2 py-1.5 text-left", index === activeIndex ? "bg-[#eef6ff] text-[#1d4ed8]" : "text-[#343944]")} onMouseEnter={() => setSelectedIndex(index)} onClick={() => command(item)}>
          @{item.label}
        </button>
      )) : <p className="px-2 py-1.5 text-[#8b919e]">没有匹配用户</p>}
      <span className="sr-only">{selected?.label}</span>
    </div>
  );
});

MentionList.displayName = "MentionList";

async function insertInputFile(editor: Editor | null, input: HTMLInputElement, token: string, setUploading: (value: boolean) => void) {
  const files = Array.from(input.files ?? []);
  input.value = "";
  if (!editor || !files.length) return;
  await insertFiles(editor, files, token, setUploading);
}

async function insertFiles(editor: Editor, files: File[], token: string, setUploading: (value: boolean) => void) {
  if (!files.length) return;
  setUploading(true);
  try {
    for (const file of files) {
      if (file.type.startsWith("image/")) {
        const asset = await uploadImage(file, { auth: true, context: { token } });
        editor.chain().focus().setImage({ src: asset.url || asset.signedUrl || "", alt: file.name }).run();
      } else {
        const asset = await uploadAttachment(file, { auth: true, context: { token } });
        const href = asset.url || asset.signedUrl || "";
        editor.chain().focus().insertContent(`<p><a href="${escapeAttribute(href)}" target="_blank" rel="noopener noreferrer">${escapeHtml(asset.originalname || file.name)}</a></p>`).run();
      }
    }
  } catch (error) {
    toast.error(error instanceof Error ? error.message : "上传失败");
  } finally {
    setUploading(false);
  }
}

function buttonClassName(disabled?: boolean) {
  return cn("flex size-8 items-center justify-center rounded-lg text-[#5f6674] transition hover:bg-white hover:text-[#17171d]", disabled && "cursor-not-allowed opacity-45");
}

function escapeHtml(value: string) {
  return value.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#039;");
}

function escapeAttribute(value: string) {
  return escapeHtml(value).replace(/`/g, "&#096;");
}
