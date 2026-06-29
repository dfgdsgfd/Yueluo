"use client";
import type {
  FormEvent
} from "react";
import {
  useCallback,
  useEffect,
  useState
} from "react";
import {
  Check,
  Loader2,
  Search,
  X
} from "lucide-react";
import {
  toast
} from "sonner";
import {
  Button
} from "@/components/ui/button";
import {
  getAdminList
} from "@/lib/api";
import type {
  AdminListRow
} from "@/lib/types";
import {
  cn
} from "@/lib/utils";
import {
  FieldConfig,
  PickerResource,
  PickerSelection
} from "./types";
import {
  Avatar
} from "./resource-cells";
import {
  errorMessage,
  fieldText,
  formatDateTime
} from "./helpers";

export function TextField({ label, value, onChange, type = "text", placeholder }: { label: string; value: unknown; onChange: (value: string) => void; type?: string; placeholder?: string }) {
  return (
    <label className="grid gap-1.5">
      <span className="text-xs font-semibold text-[#666c78]">{label}</span>
      <input value={String(value ?? "")} onChange={(event) => onChange(event.target.value)} type={type} placeholder={placeholder} className="h-10 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]" />
    </label>
  );
}


export function AdminObjectPicker({
  token,
  resource,
  label,
  value,
  onChange,
  multiple = false,
  placeholder,
  emptyLabel = "暂无可选对象",
  addEmptyLabel = "请先搜索并选择对象，操作前会按所选对象提交。",
  clearLabel = "清空",
  disabled = false,
  loadingLabel = "正在加载",
  removeTitle = "点击移除",
  searchLabel = "搜索",
}: {
  token: string;
  resource: PickerResource;
  label: string;
  value: PickerSelection[];
  onChange: (value: PickerSelection[]) => void;
  multiple?: boolean;
  placeholder?: string;
  emptyLabel?: string;
  addEmptyLabel?: string;
  clearLabel?: string;
  disabled?: boolean;
  loadingLabel?: string;
  removeTitle?: string;
  searchLabel?: string;
}) {
  const [keyword, setKeyword] = useState("");
  const [rows, setRows] = useState<PickerSelection[]>([]);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async (nextKeyword = keyword) => {
    setLoading(true);
    try {
      const payload = await getAdminList(resource, {
        page: 1,
        limit: 8,
        keyword: nextKeyword.trim(),
        sortField: resource === "categories" ? "id" : "created_at",
        sortOrder: "DESC",
      }, token);
      setRows(payload.items.flatMap((row) => {
        const item = pickerSelectionFromRow(resource, row);
        return item ? [item] : [];
      }));
    } catch (error) {
      toast.error(errorMessage(error));
      setRows([]);
    } finally {
      setLoading(false);
    }
  }, [keyword, resource, token]);

  useEffect(() => {
    queueMicrotask(() => {
      void load("");
    });
  }, [load]);

  async function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (disabled) return;
    await load(keyword);
  }

  function selected(item: PickerSelection) {
    return value.some((current) => String(current.id) === String(item.id));
  }

  function toggle(item: PickerSelection) {
    if (disabled) return;
    if (!multiple) {
      onChange(selected(item) ? [] : [item]);
      return;
    }
    onChange(selected(item)
      ? value.filter((current) => String(current.id) !== String(item.id))
      : [...value, item]);
  }

  return (
    <div className="grid gap-2 rounded-lg border border-black/[0.06] bg-white p-3">
      <div className="flex items-center justify-between gap-3">
        <span className="text-xs font-semibold text-[#4f5665]">{label}</span>
        {value.length && !disabled ? (
          <button type="button" onClick={() => onChange([])} className="text-xs font-semibold text-[#59606c] hover:text-[#1d4ed8]">
            {clearLabel}
          </button>
        ) : null}
      </div>
      {value.length ? (
        <div className="flex flex-wrap gap-2">
          {value.map((item) => (
            <button
              key={String(item.id)}
              type="button"
              onClick={() => toggle(item)}
              disabled={disabled}
              className="inline-flex max-w-full items-center gap-2 rounded-lg border border-[#1d4ed8]/20 bg-[#eff6ff] px-2.5 py-1.5 text-left text-xs text-[#1e3a8a] disabled:cursor-default disabled:opacity-80"
              title={disabled ? undefined : removeTitle}
            >
              <span className="min-w-0 truncate font-semibold">{item.label}</span>
              {item.displayId ? <span className="shrink-0 text-[#64748b]">{item.displayId}</span> : null}
              <X className="size-3.5 shrink-0" />
            </button>
          ))}
        </div>
      ) : (
        <p className="rounded-lg bg-[#f8fafc] px-3 py-2 text-xs text-[#7a8495]">{addEmptyLabel}</p>
      )}
      <form onSubmit={submitSearch} className="flex min-w-0 gap-2">
        <label className="relative min-w-0 flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[#9aa3b2]" />
          <input
            value={keyword}
            onChange={(event) => setKeyword(event.target.value)}
            disabled={disabled}
            className="h-10 w-full rounded-lg border border-black/[0.08] bg-[#f8fafc] pl-9 pr-3 text-sm outline-none transition focus:border-[#1d4ed8] focus:bg-white focus:ring-4 focus:ring-[#1d4ed8]/10"
            placeholder={placeholder ?? "搜索名称、账号或标题"}
          />
        </label>
        <Button type="submit" variant="outline" disabled={disabled || loading} className="h-10 rounded-lg border-black/[0.08] bg-white px-3">
          {loading ? <Loader2 className="size-4 animate-spin" /> : <Search className="size-4" />}
          {searchLabel}
        </Button>
      </form>
      <div className="grid max-h-[280px] gap-1.5 overflow-y-auto">
        {loading ? (
          <div className="rounded-lg bg-[#f8fafc] px-3 py-4 text-center text-xs text-[#7a8495]">{loadingLabel}</div>
        ) : rows.length ? rows.map((item) => (
          <button
            key={String(item.id)}
            type="button"
            onClick={() => toggle(item)}
            disabled={disabled}
            className={cn(
              "flex min-w-0 items-center gap-2 rounded-lg border px-3 py-2 text-left transition disabled:cursor-default",
              selected(item) ? "border-[#1d4ed8]/30 bg-[#eff6ff]" : "border-black/[0.05] bg-[#f8fafc] hover:bg-white",
            )}
          >
            {resource === "users" ? <Avatar src={item.avatar} label={item.label} /> : null}
            <span className="min-w-0 flex-1">
              <span className="block truncate text-sm font-semibold text-[#252b36]">{item.label}</span>
              <span className="block truncate text-xs text-[#7a8495]">{item.description ?? item.displayId ?? "-"}</span>
            </span>
            {selected(item) ? <Check className="size-4 shrink-0 text-[#1d4ed8]" /> : null}
          </button>
        )) : (
          <div className="rounded-lg bg-[#f8fafc] px-3 py-4 text-center text-xs text-[#7a8495]">{emptyLabel}</div>
        )}
      </div>
    </div>
  );
}


export function pickerSelectionFromRow(resource: PickerResource, row: AdminListRow): PickerSelection | null {
  if (row.id === undefined || row.id === null) return null;
  if (resource === "users") {
    const nickname = fieldText(row, "nickname");
    const displayId = fieldText(row, "user_id");
    return {
      id: row.id,
      label: nickname === "-" ? displayId : nickname,
      description: [displayId === "-" ? "" : `账号 ${displayId}`, fieldText(row, "email")].filter(Boolean).join(" · "),
      displayId: displayId === "-" ? undefined : displayId,
      avatar: typeof row.avatar === "string" ? row.avatar : null,
    };
  }
  if (resource === "posts") {
    const title = fieldText(row, "title");
    return {
      id: row.id,
      label: title === "-" ? `内容 ${row.id}` : title,
      description: [fieldText(row, "nickname"), formatDateTime(row.created_at)].filter((item) => item !== "-").join(" · "),
    };
  }
  const name = fieldText(row, "name");
  const title = fieldText(row, "category_title");
  return {
    id: row.id,
    label: name === "-" ? title : name,
    description: title === "-" ? undefined : title,
  };
}


export function pickerIDs(value: PickerSelection[]) {
  return value
    .map((item) => Number(item.id))
    .filter((item) => Number.isInteger(item) && item > 0);
}


export function firstPickerID(value: PickerSelection[]) {
  const ids = pickerIDs(value);
  return ids[0] ?? 0;
}


export function pickerSelectionFromField(field: FieldConfig, value: unknown, row: AdminListRow | null): PickerSelection | null {
  const normalized = normalizePickerSelection(value);
  if (normalized) return normalized;

  const rawID = value === "" || value === undefined || value === null ? row?.[field.key] : value;
  if (rawID === "" || rawID === undefined || rawID === null || !field.picker) return null;

  if (field.picker === "users") {
    return userPickerSelectionFromField(field.key, rawID, row);
  }
  if (field.picker === "posts") {
    return postPickerSelectionFromField(rawID, row);
  }
  return categoryPickerSelectionFromField(rawID, row);
}


export function normalizePickerSelection(value: unknown): PickerSelection | null {
  if (!value || typeof value !== "object" || !("id" in value)) return null;
  const record = value as Record<string, unknown>;
  const id = record.id;
  if (id === "" || id === undefined || id === null) return null;
  return {
    id: id as string | number,
    label: String(record.label ?? record.nickname ?? record.name ?? record.title ?? id),
    description: typeof record.description === "string" ? record.description : undefined,
    displayId: typeof record.displayId === "string" ? record.displayId : undefined,
    avatar: typeof record.avatar === "string" ? record.avatar : null,
  };
}


export function pickerIDFromDraftValue(value: unknown) {
  const selected = normalizePickerSelection(value);
  const raw = selected?.id ?? value;
  if (raw === "" || raw === undefined || raw === null) return "";
  const numeric = Number(raw);
  return Number.isFinite(numeric) ? numeric : raw;
}


export function userPickerSelectionFromField(fieldKey: string, id: unknown, row: AdminListRow | null): PickerSelection {
  const labelKeys = fieldKey === "follower_id"
    ? ["follower_nickname", "follower_display_id"]
    : fieldKey === "following_id"
      ? ["following_nickname", "following_display_id"]
      : ["nickname", "user_nickname", "author_nickname", "user_display_id", "user_uid", "user_id"];
  const displayKeys = fieldKey === "follower_id"
    ? ["follower_display_id", "follower_id"]
    : fieldKey === "following_id"
      ? ["following_display_id", "following_id"]
      : ["user_display_id", "user_uid", "user_id", "author_account"];
  const label = firstRowText(row, labelKeys);
  const displayId = firstRowText(row, displayKeys);
  const fallback = displayId === "-" ? String(id) : displayId;
  return {
    id: id as string | number,
    label: label === "-" ? `用户 ${fallback}` : label,
    description: displayId === "-" ? `ID ${String(id)}` : `账号 ${displayId}`,
    displayId: displayId === "-" ? undefined : displayId,
    avatar: typeof row?.avatar === "string" ? row.avatar : null,
  };
}


export function postPickerSelectionFromField(id: unknown, row: AdminListRow | null): PickerSelection {
  const title = firstRowText(row, ["post_title", "title"]);
  const description = [firstRowText(row, ["nickname", "author_nickname"]), formatDateTime(row?.created_at)]
    .filter((item) => item !== "-")
    .join(" · ");
  return {
    id: id as string | number,
    label: title === "-" ? `内容 ${String(id)}` : title,
    description: description || `ID ${String(id)}`,
  };
}


export function categoryPickerSelectionFromField(id: unknown, row: AdminListRow | null): PickerSelection {
  const name = firstRowText(row, ["category_name", "name", "category_title"]);
  return {
    id: id as string | number,
    label: name === "-" ? `分类 ${String(id)}` : name,
    description: name === "-" ? undefined : `ID ${String(id)}`,
  };
}


export function firstRowText(row: AdminListRow | null, keys: string[]) {
  if (!row) return "-";
  for (const key of keys) {
    const value = fieldText(row, key);
    if (value !== "-") return value;
  }
  return "-";
}
