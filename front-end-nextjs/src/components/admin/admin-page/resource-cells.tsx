"use client";
import {
  useState
} from "react";
import type {
  LucideIcon
} from "lucide-react";
import {
  Bell,
  Check,
  FileText,
  Image as ImageIcon,
  Megaphone,
  Package,
  Radio,
  RefreshCw,
  SlidersHorizontal,
  Star,
  X,
  Zap
} from "lucide-react";
import type {
  AdminListResource,
  AdminListRow
} from "@/lib/types";
import {
  cn
} from "@/lib/utils";
import {
  Tone
} from "./types";
import {
  detectMediaType
} from "./resource-editor";
import {
  fieldText,
  readableValue,
  tonePillClass
} from "./helpers";

export function IconButton({ label, icon: Icon, onClick, danger }: { label: string; icon: LucideIcon; onClick: () => void; danger?: boolean }) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      onClick={onClick}
      className={cn("inline-flex size-8 items-center justify-center rounded-lg transition", danger ? "text-[#d71935] hover:bg-[#fff0f2]" : "text-[#59606c] hover:bg-[#edf0f5]")}
    >
      <Icon className="size-4" />
    </button>
  );
}


export function ResourceRowActions({
  resource,
  row,
  onAction,
}: {
  resource: AdminListResource;
  row: AdminListRow;
  onAction: (action: "approve" | "reject" | "retry" | "resend" | "toggle-active" | "recommend" | "original-incentive" | "test-discord" | "test-email") => void;
}) {
  const actions: Array<{ key: Parameters<typeof onAction>[0]; label: string; icon: LucideIcon; danger?: boolean }> = [];
  if (resource === "content-review") {
    actions.push({ key: "approve", label: "通过", icon: Check }, { key: "reject", label: "驳回", icon: X, danger: true }, { key: "retry", label: "重试", icon: RefreshCw });
  }
  if (resource === "audit") {
    actions.push({ key: "approve", label: "通过", icon: Check }, { key: "reject", label: "驳回", icon: X, danger: true });
  }
  if (resource === "posts") {
    actions.push({ key: "recommend", label: "推荐", icon: Zap }, { key: "original-incentive", label: "原创激励", icon: Star });
  }
  if (resource === "posts-quality") {
    actions.push({ key: "original-incentive", label: "原创激励", icon: Star });
  }
  if (resource === "system-notifications") {
    actions.push({ key: "resend", label: "重发", icon: Bell });
  }
  if (resource === "notification-templates") {
    actions.push({ key: "test-discord", label: "Discord 测试", icon: Radio }, { key: "test-email", label: "邮件测试", icon: Megaphone });
  }
  if (resource === "user-toolbar" && row.id !== undefined) {
    actions.push({ key: "toggle-active", label: "启停", icon: SlidersHorizontal });
  }
  if (!actions.length) return null;
  return (
    <>
      {actions.map((action) => (
        <IconButton key={action.key} label={action.label} icon={action.icon} danger={action.danger} onClick={() => onAction(action.key)} />
      ))}
    </>
  );
}


export function Avatar({ src, label }: { src?: string | null; label: string }) {
  return (
    <span className="flex size-9 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-[#f0f2f6] text-xs font-semibold text-[#6b7180]">
      {src ? <span className="size-full bg-cover bg-center" style={{ backgroundImage: `url(${src})` }} /> : label.slice(0, 1).toUpperCase()}
    </span>
  );
}


export function Thumbnail({ url, className }: { url?: string | null; className?: string }) {
  const [open, setOpen] = useState(false);
  const imageURL = typeof url === "string" && isImageURL(url) ? url : "";
  return (
    <>
      <button
        type="button"
        disabled={!imageURL}
        onClick={() => setOpen(true)}
        className={cn("size-12 shrink-0 overflow-hidden rounded-lg bg-[#edf0f5] text-[#9aa0ad] transition enabled:hover:ring-2 enabled:hover:ring-[#1d4ed8]/40", className)}
        title={imageURL ? "查看图片" : "暂无图片"}
      >
        {imageURL ? <span className="block size-full bg-cover bg-center" style={{ backgroundImage: `url(${imageURL})` }} /> : <ImageIcon className="m-3 size-6" />}
      </button>
      {open ? <ImagePreviewModal url={imageURL} onClose={() => setOpen(false)} /> : null}
    </>
  );
}


export function ImagePreviewModal({ url, onClose }: { url: string; onClose: () => void }) {
  if (!url) return null;
  return (
    <div className="fixed inset-0 z-[70] flex items-center justify-center bg-[#17171d]/72 p-4" onClick={onClose}>
      <button type="button" className="absolute right-4 top-4 inline-flex size-10 items-center justify-center rounded-lg bg-white/92 text-[#252932] shadow-lg" aria-label="关闭图片预览" onClick={onClose}>
        <X className="size-5" />
      </button>
      <div
        role="img"
        aria-label="帖子图片预览"
        className="h-[86vh] max-h-[86vh] w-[92vw] max-w-[92vw] rounded-lg bg-white bg-contain bg-center bg-no-repeat shadow-2xl"
        style={{ backgroundImage: `url(${url})` }}
        onClick={(event) => event.stopPropagation()}
      />
    </div>
  );
}


export function IdentityCell({ row }: { row: AdminListRow }) {
  return (
    <div className="flex min-w-0 items-start gap-3">
      <Avatar src={typeof row.avatar === "string" ? row.avatar : null} label={fieldText(row, "nickname")} />
      <div className="min-w-0 flex-1">
        <p className="line-clamp-2 break-all font-semibold leading-5 text-[#252932]" title={fieldText(row, "nickname")}>{fieldText(row, "nickname")}</p>
        <p className="mt-1 line-clamp-2 break-all text-xs leading-4 text-[#8b919e]" title={fieldText(row, "user_id")}>{fieldText(row, "user_id")}</p>
      </div>
    </div>
  );
}


export function LongTextCell({ value, lines = 2, muted: isMuted = false }: { value: unknown; lines?: 1 | 2 | 3; muted?: boolean }) {
  const text = readableValue(value);
  return (
    <span
      className={cn(
        "block min-w-0 break-all leading-5",
        lines === 1 ? "line-clamp-1" : lines === 2 ? "line-clamp-2" : "line-clamp-3",
        isMuted ? "text-[#7b808c]" : "text-[#252932]",
      )}
      title={text}
    >
      {text}
    </span>
  );
}


export function PostCell({ row }: { row: AdminListRow }) {
  const image = firstImageURL(row);
  const title = fieldText(row, "title") !== "-" ? fieldText(row, "title") : `帖子 ${fieldText(row, "post_id")}`;
  const subtitle = fieldText(row, "content") !== "-" ? fieldText(row, "content") : fieldText(row, "nickname");
  return (
    <div className="flex min-w-0 items-center gap-3">
      <Thumbnail url={image} />
      <div className="min-w-0">
        <p className="line-clamp-1 font-semibold text-[#252932]">{title}</p>
        <p className="line-clamp-1 text-xs text-[#8b919e]">{subtitle}</p>
      </div>
    </div>
  );
}


export function MediaLibraryCell({ row }: { row: AdminListRow }) {
  const url = typeof row.url === "string" ? row.url : "";
  const type = String(row.type || detectMediaType(undefined, url));
  const title = fieldText(row, "title") !== "-" ? fieldText(row, "title") : fieldText(row, "filename");
  return (
    <div className="flex min-w-0 items-center gap-3">
      <MediaPreview url={url} type={type} />
      <div className="min-w-0">
        <p className="line-clamp-1 font-semibold text-[#252932]">{title}</p>
        <a href={url || "#"} target="_blank" rel="noreferrer" className="line-clamp-1 text-xs text-[#8b919e] hover:text-[#1d4ed8]">
          {url || "-"}
        </a>
      </div>
    </div>
  );
}


export function MediaPreview({ url, type }: { url: string; type: string }) {
  const [open, setOpen] = useState(false);
  if (type === "image" && url) {
    return (
      <>
        <button
          type="button"
          onClick={() => setOpen(true)}
          className="size-14 shrink-0 overflow-hidden rounded-lg bg-[#edf0f5] bg-cover bg-center transition hover:ring-2 hover:ring-[#1d4ed8]/40"
          style={{ backgroundImage: `url(${url})` }}
          title="查看图片"
        />
        {open ? <ImagePreviewModal url={url} onClose={() => setOpen(false)} /> : null}
      </>
    );
  }
  if (type === "video" && url) {
    return <video src={url} className="h-14 w-20 shrink-0 rounded-lg bg-[#edf0f5] object-cover" muted preload="metadata" controls />;
  }
  const Icon = type === "apk" ? Package : FileText;
  return (
    <a href={url || "#"} target="_blank" rel="noreferrer" className="flex size-14 shrink-0 items-center justify-center rounded-lg bg-[#edf0f5] text-[#687080] transition hover:bg-[#e5e8ef]">
      <Icon className="size-5" />
    </a>
  );
}


export function firstImageURL(row: AdminListRow) {
  const candidates = [row.cover_url, row.first_image_url, row.image_url, row.post_cover, row.cover];
  for (const candidate of candidates) {
    if (typeof candidate === "string" && isImageURL(candidate)) return candidate;
  }
  if (Array.isArray(row.images)) {
    for (const item of row.images) {
      if (typeof item === "string" && isImageURL(item)) return item;
      if (item && typeof item === "object") {
        const record = item as Record<string, unknown>;
        for (const key of ["image_url", "url", "src", "preview", "cover_url"]) {
          const value = record[key];
          if (typeof value === "string" && isImageURL(value)) return value;
        }
      }
    }
  }
  return null;
}


export function isImageURL(value: string) {
  const trimmed = value.trim();
  if (!trimmed) return false;
  if (trimmed.startsWith("data:image/")) return true;
  return /\.(avif|bmp|gif|jpe?g|png|webp)(\?.*)?$/i.test(trimmed) || /\/api\/file\/(images|covers|thumbnails|media)\//i.test(trimmed);
}


export function mediaTypeLabel(value: unknown) {
  const raw = String(value ?? "");
  if (raw === "image") return "图片";
  if (raw === "video") return "视频";
  if (raw === "apk") return "APK";
  if (raw === "file") return "文件";
  return raw || "-";
}


export function mediaTypeTone(value: unknown): Tone {
  const raw = String(value ?? "");
  if (raw === "image") return "purple";
  if (raw === "video") return "blue";
  if (raw === "apk") return "amber";
  return "slate";
}


export function StatusPill({ value, tone }: { value: string; tone: Tone }) {
  return <span className={cn("inline-flex max-w-full items-center rounded-full px-2 py-1 text-xs font-semibold", tonePillClass(tone))}>{value}</span>;
}
