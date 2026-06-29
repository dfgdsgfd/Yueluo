"use client";

import {
  useEffect,
  useState
} from "react";
import {
  Download,
  Eye,
  FileSearch,
  FileText,
  Loader2,
  X
} from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import {
  downloadFileRecycleItem,
  inspectFileRecycleItem,
  previewFileRecycleItem
} from "@/lib/api";
import type {
  AdminListRow,
  FileRecycleInspectPayload,
  FileRecyclePathState
} from "@/lib/types";
import { cn } from "@/lib/utils";
import { errorMessage, fieldBytes } from "./helpers";
import { IconButton, LongTextCell } from "./resource-cells";

type TranslationFn = ReturnType<typeof useTranslations>;

type PreviewBlob = {
  url: string;
  contentType?: string | null;
  filename?: string | null;
  kind?: string | null;
};

export function FileRecycleRowActions({
  row,
  token,
}: {
  row: AdminListRow;
  token: string;
}) {
  const t = useTranslations("adminPortal.fileRecycleBin");
  const id = row.id;
  const [inspection, setInspection] = useState<FileRecycleInspectPayload | null>(null);
  const [preview, setPreview] = useState<PreviewBlob | null>(null);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState<"inspect" | "preview" | "download" | null>(null);

  useEffect(() => {
    return () => {
      if (preview?.url) {
        URL.revokeObjectURL(preview.url);
      }
    };
  }, [preview?.url]);

  async function loadInspection(showToast: boolean) {
    if (id === undefined) return null;
    setLoading("inspect");
    try {
      const payload = await inspectFileRecycleItem(id, token);
      setInspection(payload);
      setOpen(true);
      if (showToast) {
        toast.success(payload.recycled.exists ? t("toasts.recycledExists") : t("toasts.recycledMissing"));
      }
      return payload;
    } catch (error) {
      toast.error(fileRecycleErrorMessage(error, t));
      return null;
    } finally {
      setLoading(null);
    }
  }

  async function openPreview() {
    if (id === undefined) return;
    const current = inspection ?? await loadInspection(false);
    if (!current) return;
    if (!current.previewable) {
      toast.error(t(current.recycled.is_dir ? "toasts.directoryUnsupported" : "toasts.previewUnsupported"));
      return;
    }
    setLoading("preview");
    try {
      const payload = await previewFileRecycleItem(id, token);
      const url = URL.createObjectURL(payload.blob);
      setPreview((previous) => {
        if (previous?.url) {
          URL.revokeObjectURL(previous.url);
        }
        return {
          url,
          contentType: payload.contentType,
          filename: payload.filename ?? current.filename,
          kind: current.preview_kind,
        };
      });
      setOpen(true);
    } catch (error) {
      toast.error(fileRecycleErrorMessage(error, t));
    } finally {
      setLoading(null);
    }
  }

  async function download() {
    if (id === undefined) return;
    setLoading("download");
    try {
      const payload = await downloadFileRecycleItem(id, token);
      saveDownloadBlob(payload.blob, payload.filename ?? inspection?.filename ?? `recycled-file-${String(id)}`);
      toast.success(t("toasts.downloadReady"));
    } catch (error) {
      toast.error(fileRecycleErrorMessage(error, t));
    } finally {
      setLoading(null);
    }
  }

  function closeDialog() {
    setOpen(false);
    setPreview(null);
  }

  return (
    <>
      <IconButton label={t("actions.inspect")} icon={loading === "inspect" ? Loader2 : FileSearch} onClick={() => void loadInspection(true)} />
      <IconButton label={t("actions.preview")} icon={loading === "preview" ? Loader2 : Eye} onClick={() => void openPreview()} />
      <IconButton label={t("actions.download")} icon={loading === "download" ? Loader2 : Download} onClick={() => void download()} />
      {open ? (
        <FileRecyclePreviewDialog
          inspection={inspection}
          preview={preview}
          onClose={closeDialog}
          onDownload={() => void download()}
        />
      ) : null}
    </>
  );
}

function FileRecyclePreviewDialog({
  inspection,
  preview,
  onClose,
  onDownload,
}: {
  inspection: FileRecycleInspectPayload | null;
  preview: PreviewBlob | null;
  onClose: () => void;
  onDownload: () => void;
}) {
  const t = useTranslations("adminPortal.fileRecycleBin");
  return (
    <div className="fixed inset-0 z-[80] flex items-center justify-center bg-[#17171d]/72 p-4" onClick={onClose}>
      <section
        className="max-h-[88vh] w-full max-w-4xl overflow-hidden rounded-lg bg-white shadow-2xl"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex items-center justify-between gap-3 border-b border-black/[0.08] px-4 py-3">
          <div className="min-w-0">
            <h2 className="line-clamp-1 text-base font-semibold text-[#252932]">{t("preview.title")}</h2>
            <p className="mt-1 line-clamp-1 text-xs text-[#8b919e]">{inspection?.filename ?? "-"}</p>
          </div>
          <div className="flex shrink-0 items-center gap-1">
            {inspection?.downloadable ? <IconButton label={t("actions.download")} icon={Download} onClick={onDownload} /> : null}
            <IconButton label={t("actions.close")} icon={X} onClick={onClose} />
          </div>
        </div>
        <div className="grid max-h-[calc(88vh-64px)] min-h-[360px] min-w-0 gap-0 overflow-auto lg:grid-cols-[minmax(0,1fr)_280px]">
          <div className="flex min-h-[320px] min-w-0 items-center justify-center bg-[#f4f6fa] p-3">
            {preview ? <PreviewSurface preview={preview} /> : <EmptyPreview />}
          </div>
          <aside className="min-w-0 border-t border-black/[0.08] p-4 lg:border-l lg:border-t-0">
            {inspection ? (
              <div className="space-y-4">
                <PathStateBlock title={t("preview.original")} state={inspection.original} />
                <PathStateBlock title={t("preview.recycled")} state={inspection.recycled} />
                {inspection.files?.length ? (
                  <div>
                    <p className="mb-2 text-xs font-semibold uppercase text-[#8b919e]">{t("preview.files")}</p>
                    <div className="max-h-36 space-y-1 overflow-auto rounded-lg border border-black/[0.06] p-2">
                      {inspection.files.map((file) => (
                        <div key={file.name} className="flex min-w-0 items-center justify-between gap-2 text-xs text-[#59606c]">
                          <span className="line-clamp-1 min-w-0 break-all">{file.name}</span>
                          <span className="shrink-0 text-[#8b919e]">{fieldBytes(file, "size_bytes")}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : null}
                <LongTextCell value={inspection.item.recycled_path || inspection.item.original_path || inspection.item.original_url} lines={3} muted />
              </div>
            ) : (
              <p className="text-sm text-[#8b919e]">{t("preview.noInspection")}</p>
            )}
          </aside>
        </div>
      </section>
    </div>
  );
}

function PreviewSurface({ preview }: { preview: PreviewBlob }) {
  const t = useTranslations("adminPortal.fileRecycleBin");
  const kind = preview.kind ?? previewKindFromContentType(preview.contentType);
  if (kind === "image") {
    return (
      <object
        data={preview.url}
        type={preview.contentType ?? "image/*"}
        aria-label={t("preview.imageAlt")}
        className="max-h-[70vh] max-w-full rounded-lg object-contain shadow-sm"
      >
        <EmptyPreview />
      </object>
    );
  }
  if (kind === "video") {
    return <video src={preview.url} className="max-h-[70vh] max-w-full rounded-lg bg-black" controls preload="metadata" />;
  }
  if (kind === "audio") {
    return <audio src={preview.url} className="w-full max-w-xl" controls />;
  }
  if (kind === "pdf" || kind === "text") {
    return <iframe src={preview.url} title={preview.filename ?? t("preview.title")} className="h-[70vh] w-full rounded-lg border border-black/[0.08] bg-white" />;
  }
  return <EmptyPreview />;
}

function EmptyPreview() {
  const t = useTranslations("adminPortal.fileRecycleBin");
  return (
    <div className="flex flex-col items-center gap-2 text-[#8b919e]">
      <FileText className="size-10" />
      <p className="text-sm">{t("preview.empty")}</p>
    </div>
  );
}

function PathStateBlock({ title, state }: { title: string; state: FileRecyclePathState }) {
  const t = useTranslations("adminPortal.fileRecycleBin");
  const tone = state.unsafe ? "danger" : state.exists ? "ok" : state.configured ? "warn" : "muted";
  return (
    <div className="min-w-0">
      <div className="mb-2 flex items-center justify-between gap-2">
        <p className="text-xs font-semibold uppercase text-[#8b919e]">{title}</p>
        <span
          className={cn(
            "rounded-full px-2 py-1 text-xs font-semibold",
            tone === "ok" && "bg-[#ecfdf5] text-[#047857]",
            tone === "warn" && "bg-[#fff7ed] text-[#c2410c]",
            tone === "danger" && "bg-[#fef2f2] text-[#b91c1c]",
            tone === "muted" && "bg-[#f3f4f6] text-[#6b7280]",
          )}
        >
          {state.unsafe ? t("status.unsafe") : state.exists ? t("status.exists") : state.configured ? t("status.missing") : t("status.notConfigured")}
        </span>
      </div>
      <div className="space-y-1 text-xs text-[#59606c]">
        <LongTextCell value={state.path ?? "-"} lines={2} muted />
        {state.exists ? (
          <p>
            {state.is_dir ? t("status.directory") : t("status.file")} / {fieldBytes(state, "size_bytes")}
          </p>
        ) : null}
      </div>
    </div>
  );
}

function previewKindFromContentType(contentType?: string | null) {
  const raw = contentType ?? "";
  if (raw.startsWith("image/")) return "image";
  if (raw.startsWith("video/")) return "video";
  if (raw.startsWith("audio/")) return "audio";
  if (raw.startsWith("text/")) return "text";
  if (raw.includes("pdf")) return "pdf";
  return "other";
}

function saveDownloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}

function fileRecycleErrorMessage(error: unknown, t: TranslationFn) {
  const message = error instanceof Error ? error.message : "";
  const prefix = "admin.fileRecycleBin.";
  if (message.startsWith(prefix)) {
    const key = `errors.${message.slice(prefix.length)}`;
    if (t.has(key)) {
      return t(key);
    }
  }
  return errorMessage(error);
}
