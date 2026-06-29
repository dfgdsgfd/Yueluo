"use client";

import { useEffect, useRef, useState, type ReactNode } from "react";
import type { Editor } from "@tiptap/core";
import CharacterCount from "@tiptap/extension-character-count";
import Link from "@tiptap/extension-link";
import Placeholder from "@tiptap/extension-placeholder";
import TextAlign from "@tiptap/extension-text-align";
import Underline from "@tiptap/extension-underline";
import { Markdown } from "@tiptap/markdown";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import {
  AlignCenter,
  AlignJustify,
  AlignLeft,
  AlignRight,
  Bold,
  Code,
  Heading1,
  Heading2,
  Highlighter,
  Italic,
  Link2,
  List,
  ListOrdered,
  Minus,
  Pilcrow,
  Quote,
  Redo2,
  RemoveFormatting,
  Strikethrough,
  Underline as UnderlineIcon,
  Undo2,
  type LucideIcon,
} from "lucide-react";
import { useTranslations } from "next-intl";
import { MarkdownContent } from "@/components/markdown-content";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { inferRichTextContentType, insertMarkdownFromPaste, isRichTextHtml } from "@/lib/rich-text";
import { postContentLength } from "@/lib/post-content";
import { cn } from "@/lib/utils";

type RichTextToolbarMode = "simple" | "full";
type RichTextEditingMode = "visual" | "markdown" | "preview";

export type RichTextEditorProps = {
  className?: string;
  contentBefore?: ReactNode;
  hideFooter?: boolean;
  limit: number;
  onChange: (value: string) => void;
  placeholder: string;
  value: string;
};

type ToolbarAction = {
  active?: (editor: Editor) => boolean;
  enabled?: (editor: Editor) => boolean;
  icon: LucideIcon;
  key: string;
  run: (editor: Editor, options: { linkPrompt: string }) => void;
};

const simpleActions: ToolbarAction[] = [
  {
    key: "bold",
    icon: Bold,
    active: (editor) => editor.isActive("bold"),
    run: (editor) => editor.chain().focus().toggleBold().run(),
  },
  {
    key: "italic",
    icon: Italic,
    active: (editor) => editor.isActive("italic"),
    run: (editor) => editor.chain().focus().toggleItalic().run(),
  },
  {
    key: "underline",
    icon: UnderlineIcon,
    active: (editor) => editor.isActive("underline"),
    run: (editor) => editor.chain().focus().toggleUnderline().run(),
  },
  {
    key: "bulletList",
    icon: List,
    active: (editor) => editor.isActive("bulletList"),
    run: (editor) => editor.chain().focus().toggleBulletList().run(),
  },
  {
    key: "orderedList",
    icon: ListOrdered,
    active: (editor) => editor.isActive("orderedList"),
    run: (editor) => editor.chain().focus().toggleOrderedList().run(),
  },
  {
    key: "link",
    icon: Link2,
    active: (editor) => editor.isActive("link"),
    run: (editor, options) => {
      const previous = editor.getAttributes("link").href as string | undefined;
      const url = window.prompt(options.linkPrompt, previous ?? "https://");

      if (url === null) {
        return;
      }

      if (!url.trim()) {
        editor.chain().focus().unsetLink().run();
        return;
      }

      editor.chain().focus().extendMarkRange("link").setLink({ href: url.trim() }).run();
    },
  },
];

const fullOnlyActions: ToolbarAction[] = [
  {
    key: "paragraph",
    icon: Pilcrow,
    active: (editor) => editor.isActive("paragraph"),
    run: (editor) => editor.chain().focus().setParagraph().run(),
  },
  {
    key: "heading1",
    icon: Heading1,
    active: (editor) => editor.isActive("heading", { level: 1 }),
    run: (editor) => editor.chain().focus().toggleHeading({ level: 1 }).run(),
  },
  {
    key: "heading2",
    icon: Heading2,
    active: (editor) => editor.isActive("heading", { level: 2 }),
    run: (editor) => editor.chain().focus().toggleHeading({ level: 2 }).run(),
  },
  {
    key: "strike",
    icon: Strikethrough,
    active: (editor) => editor.isActive("strike"),
    run: (editor) => editor.chain().focus().toggleStrike().run(),
  },
  {
    key: "blockquote",
    icon: Quote,
    active: (editor) => editor.isActive("blockquote"),
    run: (editor) => editor.chain().focus().toggleBlockquote().run(),
  },
  {
    key: "code",
    icon: Code,
    active: (editor) => editor.isActive("code"),
    run: (editor) => editor.chain().focus().toggleCode().run(),
  },
  {
    key: "codeBlock",
    icon: Highlighter,
    active: (editor) => editor.isActive("codeBlock"),
    run: (editor) => editor.chain().focus().toggleCodeBlock().run(),
  },
  {
    key: "horizontalRule",
    icon: Minus,
    run: (editor) => editor.chain().focus().setHorizontalRule().run(),
  },
  {
    key: "alignLeft",
    icon: AlignLeft,
    active: (editor) => editor.isActive({ textAlign: "left" }),
    run: (editor) => editor.chain().focus().setTextAlign("left").run(),
  },
  {
    key: "alignCenter",
    icon: AlignCenter,
    active: (editor) => editor.isActive({ textAlign: "center" }),
    run: (editor) => editor.chain().focus().setTextAlign("center").run(),
  },
  {
    key: "alignRight",
    icon: AlignRight,
    active: (editor) => editor.isActive({ textAlign: "right" }),
    run: (editor) => editor.chain().focus().setTextAlign("right").run(),
  },
  {
    key: "alignJustify",
    icon: AlignJustify,
    active: (editor) => editor.isActive({ textAlign: "justify" }),
    run: (editor) => editor.chain().focus().setTextAlign("justify").run(),
  },
  {
    key: "clear",
    icon: RemoveFormatting,
    run: (editor) => editor.chain().focus().clearNodes().unsetAllMarks().run(),
  },
  {
    key: "undo",
    icon: Undo2,
    enabled: (editor) => editor.can().undo(),
    run: (editor) => editor.chain().focus().undo().run(),
  },
  {
    key: "redo",
    icon: Redo2,
    enabled: (editor) => editor.can().redo(),
    run: (editor) => editor.chain().focus().redo().run(),
  },
];

export function RichTextEditor({
  className,
  contentBefore,
  hideFooter,
  limit,
  onChange,
  placeholder,
  value,
}: RichTextEditorProps) {
  const t = useTranslations();
  const [toolbarMode, setToolbarMode] = useState<RichTextToolbarMode>("simple");
  const [editingMode, setEditingMode] = useState<RichTextEditingMode>("visual");
  const [initialContent] = useState(() => value);
  const [initialContentType] = useState(() => inferRichTextContentType(value));
  const editorRef = useRef<Editor | null>(null);

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
      Underline,
      Link.configure({
        autolink: true,
        openOnClick: false,
      }),
      TextAlign.configure({
        types: ["heading", "paragraph"],
      }),
      Placeholder.configure({
        placeholder,
      }),
      CharacterCount,
    ],
    content: initialContent,
    contentType: initialContentType,
    editorProps: {
      attributes: {
        class:
          "min-h-[178px] px-4 py-3 text-[15px] leading-7 text-[#25252b] outline-none",
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

    if (isRichTextHtml(value)) {
      if (editor.getHTML() === value) {
        onChange(editor.getMarkdown());
        return;
      }
      editor.commands.setContent(value, { emitUpdate: false, contentType: "html" });
      onChange(editor.getMarkdown());
      return;
    }

    if (editor.getText() === value) {
      return;
    }
    editor.commands.setContent(value, { emitUpdate: false, contentType: "markdown" });
  }, [editor, onChange, value]);

  const actions = toolbarMode === "simple" ? simpleActions : [...simpleActions, ...fullOnlyActions];
  const characterCount = postContentLength(value);

  function changeEditingMode(mode: RichTextEditingMode) {
    if (mode === "markdown" && editor && isRichTextHtml(value)) {
      onChange(editor.getMarkdown());
    }
    setEditingMode(mode);
  }

  return (
    <div className={cn(
      "rounded-xl border border-[#e8e8eb] bg-white transition focus-within:border-primary",
      className,
    )}>
      <div className="flex flex-wrap items-center gap-2 border-b border-[#eeeeef] p-2">
        <div className="grid grid-cols-3 rounded-full bg-[#f6f6f7] p-1 text-xs font-semibold">
          {(["visual", "markdown", "preview"] as const).map((mode) => (
            <button
              key={mode}
              type="button"
              onClick={() => changeEditingMode(mode)}
              className={cn(
                "h-8 rounded-full px-3 transition-colors",
                editingMode === mode
                  ? "bg-white text-primary shadow-sm"
                  : "text-[#777780] hover:text-[#25252b]",
              )}
            >
              {t(`publish.editor.richText.${mode}`)}
            </button>
          ))}
        </div>

        {editingMode === "visual" ? (
          <>
            <div className="grid grid-cols-2 rounded-full bg-[#f6f6f7] p-1 text-xs font-semibold">
              {(["simple", "full"] as const).map((mode) => (
                <button
                  key={mode}
                  type="button"
                  onClick={() => setToolbarMode(mode)}
                  className={cn(
                    "h-8 rounded-full px-3 transition-colors",
                    toolbarMode === mode
                      ? "bg-white text-primary shadow-sm"
                      : "text-[#777780] hover:text-[#25252b]",
                  )}
                >
                  {t(`publish.editor.richText.${mode}`)}
                </button>
              ))}
            </div>
            <TooltipProvider>
              <div className="flex min-w-0 flex-1 flex-wrap items-center gap-1">
                {actions.map((action) => (
                  <ToolbarButton
                    key={action.key}
                    action={action}
                    editor={editor}
                    label={t(`publish.editor.richText.actions.${action.key}`)}
                    linkPrompt={t("publish.editor.richText.linkPrompt")}
                  />
                ))}
              </div>
            </TooltipProvider>
          </>
        ) : null}
      </div>

      {contentBefore}

      <div>
        <div className="min-w-0">
          {editingMode === "visual" ? (
            <EditorContent className="publish-rich-text-content" editor={editor} />
          ) : editingMode === "markdown" ? (
            <textarea
              aria-label={t("publish.editor.richText.markdown")}
              className="min-h-[240px] w-full resize-y bg-[#17171d] px-4 py-3 font-mono text-sm leading-6 text-[#f3f4f6] outline-none lg:min-h-[360px]"
              onChange={(event) => onChange(event.target.value)}
              placeholder={placeholder}
              value={value}
            />
          ) : (
            <MarkdownContent
              className="markdown-editor-preview min-h-[240px] px-4 py-3 text-[15px] leading-7 text-[#25252b]"
              content={value}
            />
          )}
        </div>
      </div>

      {hideFooter ? null : (
        <div className="flex items-center justify-between gap-3 border-t border-[#eeeeef] px-4 py-2 text-xs text-[#8a8a91]">
          <span>{t(`publish.editor.richText.${editingMode === "visual" ? `${toolbarMode}Hint` : `${editingMode}Hint`}`)}</span>
          <span className={cn(characterCount > limit && "text-primary")}>
            {characterCount}/{limit}
          </span>
        </div>
      )}
    </div>
  );
}

function ToolbarButton({
  action,
  editor,
  label,
  linkPrompt,
}: {
  action: ToolbarAction;
  editor: Editor | null;
  label: string;
  linkPrompt: string;
}) {
  const Icon = action.icon;
  const active = editor ? action.active?.(editor) : false;
  const enabled = editor ? (action.enabled ? action.enabled(editor) : true) : false;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-label={label}
          disabled={!enabled}
          onClick={() => {
            if (editor) {
              action.run(editor, {
                linkPrompt,
              });
            }
          }}
          className={cn(
            "flex size-8 items-center justify-center rounded-lg text-[#67676f] transition-colors hover:bg-[#f4f4f5] hover:text-[#25252b] disabled:cursor-not-allowed disabled:opacity-40",
            active && "bg-[#fff1f3] text-primary hover:bg-[#fff1f3] hover:text-primary",
          )}
        >
          <Icon className="size-4" />
        </button>
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}
