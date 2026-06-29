"use client";
import type {
  FormEvent,
  ReactNode
} from "react";
import {
  useState
} from "react";
import type {
  LucideIcon
} from "lucide-react";
import {
  ChevronLeft,
  ChevronRight,
  Database,
  Filter,
  Loader2,
  RefreshCw,
  Save,
  Upload,
  X
} from "lucide-react";
import {
  useTranslations
} from "next-intl";
import {
  toast
} from "sonner";
import {
  Button
} from "@/components/ui/button";
import {
  AdminRichTextEditor
} from "@/components/admin/admin-rich-text-editor";
import {
  uploadAdminMedia,
  uploadAdminApk,
  uploadAttachment,
  uploadImage,
  uploadImages,
  uploadVideo
} from "@/lib/api";
import type {
  AdminListResource,
  AdminListRow
} from "@/lib/types";
import {
  cn
} from "@/lib/utils";
import {
  FieldConfig,
  FilterConfig,
  UploadKind,
  UploadedAsset
} from "./types";
import {
  AdminObjectPicker,
  pickerSelectionFromField
} from "./object-picker";
import {
  Thumbnail,
  isImageURL
} from "./resource-cells";
import {
  InfoTile
} from "./operations-widgets";
import {
  columnLabel,
  errorMessage,
  formatCompact,
  parseJsonLike,
  readableValue,
  renderReadableValue,
  truthy
} from "./helpers";
import {
  ChoiceSelect,
  ToggleSwitch,
  castSelectValue,
  selectOptionsWithCurrent
} from "./form-fields";

export function FilterControl({ filter, value, onChange }: { filter: FilterConfig; value: string; onChange: (value: string) => void }) {
  if (filter.type === "select" || filter.type === "boolean") {
    return (
      <label className="inline-flex h-10 items-center gap-2 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-2 text-xs text-[#666c78]">
        <Filter className="size-3.5" />
        <select value={value} onChange={(event) => onChange(event.target.value)} className="h-8 max-w-[180px] bg-transparent text-sm text-[#343944] outline-none">
          <option value="">{filter.label}</option>
          {selectOptionsWithCurrent(filter.options ?? [], value).map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
        </select>
      </label>
    );
  }
  return (
    <input
      value={value}
      onChange={(event) => onChange(event.target.value)}
      className="h-10 w-[min(190px,100%)] rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]"
      placeholder={filter.label}
    />
  );
}


export function EditorDrawer({
  mode,
  title,
  fields,
  row,
  draft,
  token,
  saving,
  readOnly,
  restoreAction,
  formAddon,
  onDraftChange,
  onFieldUpload,
  onClose,
  onSubmit,
}: {
  mode: "create" | "edit" | "detail" | null;
  title: string;
  fields: FieldConfig[];
  row: AdminListRow | null;
  draft: Record<string, unknown>;
  token: string;
  saving: boolean;
  readOnly: boolean;
  restoreAction?: { label: string; loading?: boolean; onClick: () => void };
  formAddon?: ReactNode;
  onDraftChange: (key: string, value: unknown) => void;
  onFieldUpload?: (field: FieldConfig, assets: UploadedAsset[], files: File[]) => void;
  onClose: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  if (!mode) return null;
  const visibleFields = fields.filter((field) => !(mode === "create" && field.editOnly) && !(mode === "edit" && field.createOnly));

  return (
    <div className="fixed inset-0 z-50">
      <button type="button" aria-label="关闭抽屉遮罩" className="absolute inset-0 bg-[#17171d]/28" onClick={onClose} />
      <aside className="absolute inset-y-0 right-0 flex w-full max-w-[560px] flex-col bg-white shadow-2xl">
        <div className="flex h-16 shrink-0 items-center gap-3 border-b border-black/[0.06] px-4">
          <h2 className="min-w-0 flex-1 truncate text-base font-semibold text-[#17171d]">{title}</h2>
          <Button type="button" size="icon" variant="ghost" onClick={onClose} className="size-10 rounded-lg">
            <X className="size-5" />
          </Button>
        </div>
        <form onSubmit={onSubmit} className="flex min-h-0 flex-1 flex-col">
          <div className="min-h-0 flex-1 overflow-y-auto p-4">
            {mode === "detail" ? (
              <DetailGrid row={row} fields={fields} />
            ) : (
              <div className="grid gap-3">
                {restoreAction ? (
                  <button
                    type="button"
                    onClick={restoreAction.onClick}
                    disabled={restoreAction.loading}
                    className="flex items-center justify-between rounded-lg border border-[#1d4ed8]/15 bg-[#eff6ff] px-3 py-2 text-left text-sm font-semibold text-[#1e3a8a] transition hover:bg-[#dbeafe] disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    <span>{restoreAction.label}</span>
                    {restoreAction.loading ? <Loader2 className="size-4 animate-spin" /> : <RefreshCw className="size-4" />}
                  </button>
                ) : null}
                {formAddon}
                {visibleFields.map((field) => (
                  <FieldInput
                    key={field.key}
                    field={field}
                    value={draft[field.key]}
                    row={row}
                    token={token}
                    readOnly={readOnly}
                    onChange={(value) => onDraftChange(field.key, value)}
                    onUpload={(assets, files) => onFieldUpload?.(field, assets, files)}
                  />
                ))}
              </div>
            )}
          </div>
          <div className="flex shrink-0 justify-end gap-2 border-t border-black/[0.06] p-4">
            <Button type="button" variant="outline" onClick={onClose} className="rounded-lg border-black/[0.08] bg-white hover:bg-[#f6f7fb]">
              取消
            </Button>
            {!readOnly ? (
              <Button type="submit" disabled={saving} className="rounded-lg bg-[#1d4ed8] hover:bg-[#1e40af]">
                <span className="inline-flex size-4 items-center justify-center" aria-hidden="true">
                  {saving ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
                </span>
                <span>保存</span>
              </Button>
            ) : null}
          </div>
        </form>
      </aside>
    </div>
  );
}


export function FieldInput({
  field,
  value,
  row,
  token,
  readOnly,
  onChange,
  onUpload,
}: {
  field: FieldConfig;
  value: unknown;
  row: AdminListRow | null;
  token: string;
  readOnly: boolean;
  onChange: (value: unknown) => void;
  onUpload?: (assets: UploadedAsset[], files: File[]) => void;
}) {
  const adminT = useTranslations("adminPortal");
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState("");
  const common = "w-full rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm text-[#17171d] outline-none transition focus:border-[#1d4ed8] focus:bg-white focus:ring-4 focus:ring-[#1d4ed8]/10 disabled:bg-[#f1f2f5] disabled:text-[#8a8f9d]";
  const valueText = field.uploadTarget === "json-list" ? listValueText(value) : String(value ?? "");
  const isImageUpload = field.upload === "image";

  if (field.picker) {
    const selected = pickerSelectionFromField(field, value, row);
    return (
      <AdminObjectPicker
        token={token}
        resource={field.picker}
        label={`${field.label}${field.required ? " *" : ""}`}
        value={selected ? [selected] : []}
        onChange={(items) => onChange(items[0] ?? "")}
        placeholder={field.pickerPlaceholder}
        emptyLabel="未找到可选对象"
        disabled={readOnly || field.autoFilled}
      />
    );
  }

  async function handleUpload(files: File[]) {
    if (!files.length || !field.upload) return;
    setUploading(true);
    setUploadProgress("准备上传");
    try {
      const assets = await uploadFieldFiles(field.upload, files, token, (percent) => {
        setUploadProgress(percent ? `上传中 ${percent}%` : "上传中");
      });
      const urls = assets.map((asset) => asset.url).filter(Boolean);
      const nextValue = field.uploadTarget === "json-list" ? appendUploadURLs(value, urls) : urls[0];
      onChange(nextValue);
      onUpload?.(assets, files);
      setUploadProgress("上传完成");
      toast.success(`${field.label}上传完成`);
    } catch (error) {
      toast.error(errorMessage(error));
      setUploadProgress("");
    } finally {
      setUploading(false);
    }
  }

  const uploadControl = field.upload && !readOnly ? (
    <div className="mt-2 flex flex-col gap-2 rounded-lg border border-dashed border-black/[0.08] bg-white p-2 sm:flex-row sm:items-center sm:justify-between">
      <div className="min-w-0 text-xs text-[#7b808c]">
        <p className="font-semibold text-[#4f5562]">{uploadKindLabel(field.upload)}</p>
        <p className="truncate">
          {uploadProgress || (field.upload === "media"
            ? adminT("mediaLibrary.unrestrictedUploadHint")
            : uploadHint(field.upload))}
        </p>
      </div>
      <label className={cn("inline-flex h-9 cursor-pointer items-center justify-center gap-2 rounded-lg px-3 text-sm font-semibold transition", uploading ? "bg-[#edf0f5] text-[#8a8f9d]" : "bg-[#17171d] text-white hover:bg-[#2a2b32]")}>
        {uploading ? <Loader2 className="size-4 animate-spin" /> : <Upload className="size-4" />}
        <span>{uploading ? "上传中" : "选择文件"}</span>
        <input
          type="file"
          disabled={uploading}
          multiple={field.upload === "image" && field.uploadTarget === "json-list"}
          accept={uploadAccept(field.upload)}
          onChange={(event) => {
            const files = Array.from(event.target.files ?? []);
            event.target.value = "";
            void handleUpload(files);
          }}
          className="sr-only"
        />
      </label>
    </div>
  ) : null;

  const preview = field.upload && valueText.trim() ? (
    <div className="mt-2">
      <UploadPreview kind={field.upload} value={valueText} />
    </div>
  ) : null;
  const disabled = readOnly || field.autoFilled;

  return (
    <div className="block">
      <span className="mb-1.5 block text-xs font-semibold text-[#5f6674]">{field.label}{field.required ? " *" : ""}</span>
      {field.type === "richtext" ? (
        <AdminRichTextEditor
          disabled={disabled}
          token={token}
          value={valueText}
          onChange={(next) => onChange(next)}
          placeholder={field.placeholder}
        />
      ) : isImageUpload ? (
        <ImageUploadValue value={valueText} onClear={disabled ? undefined : () => onChange("")} />
      ) : field.type === "textarea" || field.type === "json" ? (
        <textarea
          value={field.type === "json" && typeof value !== "string" ? JSON.stringify(value ?? {}, null, 2) : valueText}
          onChange={(event) => onChange(event.target.value)}
          disabled={disabled}
          required={field.required}
          className={cn(common, "min-h-[112px] py-2")}
          placeholder={field.placeholder}
        />
      ) : field.type === "boolean" ? (
        <ToggleSwitch value={truthy(value)} onChange={onChange} disabled={disabled} onLabel="是" offLabel="否" />
      ) : field.type === "select" ? (
        <ChoiceSelect
          value={String(value ?? "")}
          onChange={(next) => onChange(castSelectValue(next, field.optionValueType))}
          options={field.options ?? []}
          placeholder="请选择"
          disabled={disabled}
          required={field.required}
          className={cn(common, "h-10")}
        />
      ) : (
        <input
          value={valueText}
          onChange={(event) => onChange(field.type === "number" ? event.target.value : event.target.value)}
          disabled={disabled}
          required={field.required}
          className={cn(common, "h-10")}
          type={field.type === "number" ? "number" : field.type === "password" ? "password" : field.type === "datetime" ? "datetime-local" : "text"}
          step={field.key === "size_mb" ? "0.01" : undefined}
          placeholder={field.placeholder}
        />
      )}
      {uploadControl}
      {isImageUpload ? null : preview}
    </div>
  );
}


export async function uploadFieldFiles(kind: UploadKind, files: File[], token: string, onProgress: (percent?: number) => void): Promise<UploadedAsset[]> {
  if (kind === "image" && files.length > 1) {
    const options = {
      auth: true,
      context: { token },
      onProgress: (progress: { percent?: number }) => onProgress(progress.percent),
    };
    return uploadImages(files, options);
  }
  const first = files[0];
  return first ? [await uploadFieldFile(kind, first, token, onProgress)] : [];
}


export async function uploadFieldFile(kind: UploadKind, file: File, token: string, onProgress: (percent?: number) => void): Promise<UploadedAsset> {
  const options = {
    auth: true,
    context: { token },
    onProgress: (progress: { percent?: number }) => onProgress(progress.percent),
  };
  if (kind === "apk") return uploadAdminApk(file, options);
  if (kind === "media") return uploadAdminMedia(file, options);
  if (kind === "image") return uploadImage(file, options);
  if (kind === "video") return uploadVideo(file, undefined, options);
  return uploadAttachment(file, options);
}


export function uploadKindLabel(kind: UploadKind) {
  if (kind === "apk") return "APK / APKS 上传";
  if (kind === "image") return "图片上传";
  if (kind === "video") return "视频上传";
  if (kind === "media") return "图片 / 视频 / 文件上传";
  return "附件上传";
}


export function uploadHint(kind: UploadKind) {
  if (kind === "apk") return "支持 .apk、.apks，大文件自动分片";
  if (kind === "image") return "支持图片，较大图片自动分片";
  if (kind === "video") return "支持视频，较大视频自动分片";
  if (kind === "media") return "按文件类型自动调用后端上传接口";
  return "支持音频、文档、压缩包等附件";
}


export function uploadAccept(kind: UploadKind) {
  if (kind === "apk") return ".apk,.apks,application/vnd.android.package-archive,.mobileconfig";
  if (kind === "image") return "image/*";
  if (kind === "video") return "video/*";
  if (kind === "attachment") return "audio/*,.zip,.rar,.7z,.gz,.tar,.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt";
  return undefined;
}


export function listValueText(value: unknown) {
  if (Array.isArray(value)) return value.map((item) => String(item ?? "").trim()).filter(Boolean).join("\n");
  return String(value ?? "");
}


export function appendUploadURLs(value: unknown, urlsToAdd: string[]) {
  const urls = listValueText(value).split(/\r?\n/).map((item) => item.trim()).filter(Boolean);
  urlsToAdd.forEach((url) => {
    if (url && !urls.includes(url)) urls.push(url);
  });
  return urls.join("\n");
}


export function uploadAutoFillDraft(resource: AdminListResource, field: FieldConfig, assets: UploadedAsset[], files: File[], current: Record<string, unknown>) {
  if (resource === "app-versions" && field.key === "download_url" && assets.length > 0) {
    const size = Number(assets[0]?.size ?? files[0]?.size ?? 0);
    return size > 0 ? { size_mb: bytesToMBNumber(size) } : {};
  }
  if (resource !== "media-library" || field.key !== "url" || assets.length === 0) {
    return {};
  }
  const file = files[0];
  const asset = assets[0];
  const filename = asset.originalname || file?.name || filenameFromURL(asset.url);
  const next: Record<string, unknown> = {
    type: detectMediaType(file, asset.url),
  };
  if (filename && !String(current.filename ?? "").trim()) {
    next.filename = filename;
  }
  if (filename && !String(current.title ?? "").trim()) {
    next.title = stripFileExtension(filename);
  }
  return next;
}

function bytesToMBNumber(bytes: number) {
  return Number((bytes / (1024 * 1024)).toFixed(2));
}


export function detectMediaType(file?: File, url = "") {
  const filename = file?.name ?? "";
  if (file?.type.startsWith("image/") || isImageURL(filename) || isImageURL(url)) return "image";
  if (file?.type.startsWith("video/") || /\.(mp4|mov|m4v|webm|mkv|avi)(\?.*)?$/i.test(filename) || /\.(mp4|mov|m4v|webm|mkv|avi)(\?.*)?$/i.test(url)) return "video";
  if (/\.(apk|apks)(\?.*)?$/i.test(filename) || /\.(apk|apks)(\?.*)?$/i.test(url)) return "apk";
  return "file";
}


export function filenameFromURL(value: string) {
  try {
    const url = new URL(value, typeof window !== "undefined" ? window.location.origin : "https://example.com");
    return decodeURIComponent(url.pathname.split("/").filter(Boolean).pop() ?? "");
  } catch {
    return value.split("/").filter(Boolean).pop() ?? "";
  }
}


export function stripFileExtension(value: string) {
  return value.replace(/\.[a-z0-9]+$/i, "");
}


export function ImageUploadValue({ value, onClear }: { value: string; onClear?: () => void }) {
  const urls = value.split(/\r?\n/).map((item) => item.trim()).filter((item) => item && isImageURL(item));
  if (!urls.length) {
    return (
      <div className="flex min-h-24 items-center justify-center rounded-lg border border-dashed border-black/[0.08] bg-[#fafbfe] text-sm text-[#8b919e]">
        暂无图片
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-2">
      <div className="grid grid-cols-3 gap-2 sm:grid-cols-4">
        {urls.slice(0, 12).map((url, index) => (
          <div key={`${url}-${index}`} className="aspect-square overflow-hidden rounded-lg bg-white">
            <Thumbnail url={url} className="size-full rounded-lg" />
          </div>
        ))}
      </div>
      <div className="mt-2 flex items-center justify-between gap-2 text-xs text-[#8b919e]">
        <span>{urls.length > 1 ? `${urls.length} 张图片` : "已选择图片"}</span>
        {onClear ? (
          <button type="button" onClick={onClear} className="font-semibold text-[#d71935] hover:underline">
            移除
          </button>
        ) : null}
      </div>
    </div>
  );
}


export function UploadPreview({ kind, value }: { kind: UploadKind; value: string }) {
  const urls = value.split(/\r?\n/).map((item) => item.trim()).filter(Boolean);
  const first = urls[0];
  if (!first) return null;
  if ((kind === "image" || kind === "media") && isImageURL(first)) {
    if (kind === "media") {
      return (
        <div className="flex min-w-0 items-center gap-2 rounded-lg bg-[#fafbfe] p-2">
          <Thumbnail url={first} />
          <a href={first} target="_blank" rel="noreferrer" className="min-w-0 truncate text-xs font-medium text-[#59606c] hover:text-[#1d4ed8]">
            {first}
          </a>
          {urls.length > 1 ? <span className="rounded-full bg-white px-2 py-1 text-xs text-[#8b919e]">+{urls.length - 1}</span> : null}
        </div>
      );
    }
    return (
      <div className="flex min-w-0 items-center gap-2 rounded-lg bg-[#fafbfe] p-2">
        <Thumbnail url={first} />
        <span className="min-w-0 truncate text-xs font-medium text-[#59606c]">已上传图片</span>
        {urls.length > 1 ? <span className="rounded-full bg-white px-2 py-1 text-xs text-[#8b919e]">+{urls.length - 1}</span> : null}
      </div>
    );
  }
  return (
    <a href={first} target="_blank" rel="noreferrer" className="block truncate rounded-lg bg-[#fafbfe] px-3 py-2 text-xs font-medium text-[#59606c] hover:text-[#1d4ed8]">
      {first}
    </a>
  );
}


export function PaginationBar({ page, total, hasNext, disabled, onPrev, onNext }: { page: number; total?: number; hasNext: boolean; disabled: boolean; onPrev: () => void; onNext: () => void }) {
  return (
    <div className="flex flex-col gap-2 border-t border-black/[0.06] px-4 py-3 text-xs text-[#7a808c] sm:flex-row sm:items-center sm:justify-between">
      <span>第 {page} 页{total !== undefined ? ` · 共 ${formatCompact(total)} 条` : ""}</span>
      <div className="flex gap-2">
        <Button type="button" variant="outline" size="sm" disabled={disabled || page <= 1} onClick={onPrev} className="rounded-lg border-black/[0.08] bg-white hover:bg-[#f6f7fb]">
          <ChevronLeft className="size-4" />
          上一页
        </Button>
        <Button type="button" variant="outline" size="sm" disabled={disabled || !hasNext} onClick={onNext} className="rounded-lg border-black/[0.08] bg-white hover:bg-[#f6f7fb]">
          下一页
          <ChevronRight className="size-4" />
        </Button>
      </div>
    </div>
  );
}


export function LoadingBlock({ label }: { label: string }) {
  return (
    <div className="flex min-h-[320px] flex-col items-center justify-center rounded-lg border border-dashed border-black/[0.08] bg-white/56 px-4 text-center text-sm text-[#7b808c]">
      <Loader2 className="mb-2 size-5 animate-spin text-[#1d4ed8]" />
      {label}
    </div>
  );
}


export function EmptyBlock({ icon: Icon, label }: { icon: LucideIcon; label: string }) {
  return (
    <div className="flex min-h-[220px] flex-col items-center justify-center rounded-lg border border-dashed border-black/[0.08] bg-[#fafbfe] px-4 text-center text-sm text-[#8a8f9d]">
      <Icon className="mb-2 size-5" />
      {label}
    </div>
  );
}


export function DetailGrid({ row, fields }: { row: AdminListRow | null; fields: FieldConfig[] }) {
  if (!row) return <EmptyBlock icon={Database} label="暂无详情" />;
  const fieldMap = new Map(fields.map((field) => [field.key, field]));
  const fieldKeys = new Set(fieldMap.keys());
  const entries = Object.entries(row).filter(([key, value]) => !isHiddenDetailKey(key, value));
  const ordered = [
    ...entries.filter(([key]) => fieldKeys.has(key)),
    ...entries.filter(([key]) => !fieldKeys.has(key)),
  ];
  return (
    <div className="grid gap-3">
      <div className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
        <p className="text-xs font-semibold text-[#8a8f9d]">记录摘要</p>
        <div className="mt-2 grid gap-2 sm:grid-cols-2">
          {ordered.slice(0, 8).map(([key, value]) => (
            <InfoTile key={key} label={columnLabel(key)} value={detailSummaryValue(key, value, fieldMap.get(key))} />
          ))}
        </div>
      </div>
      {ordered.length > 8 ? (
        <KeyValueGrid entries={ordered.slice(8)} fieldMap={fieldMap} />
      ) : null}
    </div>
  );
}


export function KeyValueGrid({ entries, fieldMap }: { entries: Array<[string, unknown]>; fieldMap?: Map<string, FieldConfig> }) {
  if (!entries.length) return null;
  return (
    <div className="grid gap-2">
      {entries.map(([key, value]) => (
        <div key={key} className="rounded-lg border border-black/[0.06] bg-white px-3 py-2">
          <div className="mb-1 flex items-center justify-between gap-2">
            <span className="text-xs font-semibold text-[#737987]">{columnLabel(key)}</span>
            <span className="text-[11px] text-[#a0a5b1]">{key}</span>
          </div>
          <div className="break-words text-sm text-[#252932]">{renderDetailValue(key, value, fieldMap?.get(key))}</div>
        </div>
      ))}
    </div>
  );
}


export function isHiddenDetailKey(key: string, value: unknown) {
  return key === "password" || key === "api_key" || value === undefined;
}


export function detailSummaryValue(key: string, value: unknown, field?: FieldConfig) {
  if (isImageDetailValue(key, value, field)) {
    const count = imageURLsFromValue(value).length;
    return count > 1 ? `${count} 张图片` : "图片";
  }
  return readableValue(value);
}


export function renderDetailValue(key: string, value: unknown, field?: FieldConfig): ReactNode {
  if (isImageDetailValue(key, value, field)) {
    const urls = imageURLsFromValue(value);
    if (!urls.length) return <span className="text-[#8b919e]">暂无图片</span>;
    return (
      <div className="grid grid-cols-4 gap-2 sm:grid-cols-6">
        {urls.slice(0, 12).map((url, index) => (
          <Thumbnail key={`${url}-${index}`} url={url} className="size-14 rounded-lg" />
        ))}
      </div>
    );
  }
  return renderReadableValue(value);
}


export function isImageDetailValue(key: string, value: unknown, field?: FieldConfig) {
  if (field?.upload === "image") return true;
  if (typeof value === "string" && isImageURL(value) && imageFieldKey(key)) return true;
  if (Array.isArray(value) && value.some((item) => typeof item === "string" && isImageURL(item))) return true;
  if (value && typeof value === "object" && imageURLsFromValue(value).length > 0) return true;
  return false;
}


export function imageFieldKey(key: string) {
  return key === "avatar" ||
    key === "audit_result" ||
    key === "background" ||
    key === "image" ||
    key === "images" ||
    key.includes("image") ||
    key.includes("cover") ||
    key.includes("thumbnail") ||
    key.includes("poster");
}


export function imageURLsFromValue(value: unknown): string[] {
  if (typeof value === "string") {
    const parsed = parseJsonLike(value);
    if (parsed !== null) return imageURLsFromValue(parsed);
    return value.split(/\r?\n|,|，/).map((item) => item.trim()).filter((item) => item && isImageURL(item));
  }
  if (Array.isArray(value)) {
    return value.flatMap((item) => imageURLsFromValue(item));
  }
  if (value && typeof value === "object") {
    const record = value as Record<string, unknown>;
    return ["image_url", "url", "src", "preview", "cover_url", "avatar", "background", "images", "imageUrls", "image_urls"]
      .flatMap((key) => imageURLsFromValue(record[key]));
  }
  return [];
}


export function VerificationImagesCell({ row }: { row: AdminListRow }) {
  const urls = imageURLsFromValue(row.audit_result);
  if (!urls.length) {
    return <span className="text-xs text-[#9aa0ad]">未上传</span>;
  }
  return (
    <div className="flex min-w-0 items-center gap-2">
      <Thumbnail url={urls[0]} className="size-10 rounded-lg" />
      <span className="text-xs font-semibold text-[#59606c]">
        {urls.length > 1 ? `${urls.length} 张` : "1 张"}
      </span>
    </div>
  );
}
