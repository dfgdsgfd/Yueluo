"use client";

import { Folder, Image as ImageIcon, Tags, Trash2 } from "lucide-react";
import type { ResourceConfig, Tone } from "./types";
import { mediaTypeOptions } from "./types";
import { FileRecycleRemainingCell } from "./file-recycle-countdown-cell";
import { LongTextCell, MediaLibraryCell, StatusPill, mediaTypeLabel, mediaTypeTone } from "./resource-cells";
import { fieldText, fieldBytes, formatDateTime, simpleColumns } from "./helpers";

function fileRecycleStatusTone(value: unknown): Tone {
  switch (String(value ?? "")) {
    case "recycled":
      return "amber";
    case "purged":
      return "slate";
    case "missing":
      return "purple";
    case "skipped":
      return "blue";
    case "failed":
    case "purge_failed":
      return "red";
    default:
      return "slate";
  }
}

export const taxonomyMediaResourceConfigs: ResourceConfig[] = [
  {
    resource: "categories",
    label: "分类管理",
    singular: "分类",
    description: "内容分类与使用量",
    icon: Folder,
    tone: "blue",
    columns: simpleColumns(["name", "category_title", "translations", "use_count", "post_count", "created_at"]),
    filters: [{ key: "name", label: "名称" }],
    fields: [
      { key: "name", label: "名称", required: true },
      { key: "category_title", label: "兼容标题" },
      { key: "translations.en", label: "English", required: true },
      { key: "translations.zh-CN", label: "简体中文", required: true },
      { key: "translations.zh-TW", label: "繁體中文", required: true },
      { key: "translations.vi", label: "Tiếng Việt", required: true },
      { key: "translations.ja", label: "日本語", required: true },
      { key: "translations.ko", label: "한국어", required: true },
      { key: "use_count", label: "使用量", type: "number" },
    ],
  },
  {
    resource: "tags",
    label: "标签管理",
    singular: "标签",
    description: "内容标签与热度",
    icon: Tags,
    tone: "amber",
    columns: simpleColumns(["name", "use_count", "created_at"]),
    filters: [{ key: "name", label: "名称" }],
    fields: [{ key: "name", label: "名称", required: true }, { key: "use_count", label: "使用量", type: "number" }],
  },
  {
    resource: "media-library",
    label: "媒体库",
    singular: "媒体",
    description: "图片、视频与文件素材",
    icon: ImageIcon,
    tone: "purple",
    columns: [
      { key: "url", label: "媒体", render: (row) => <MediaLibraryCell row={row} />, className: "w-[360px]" },
      { key: "type", label: "类型", render: (row) => <StatusPill value={mediaTypeLabel(row.type)} tone={mediaTypeTone(row.type)} /> },
      { key: "filename", label: "文件名", render: (row) => fieldText(row, "filename") },
      { key: "created_at", label: "创建时间", render: (row) => formatDateTime(row.created_at) },
    ],
    filters: [{ key: "type", label: "类型", type: "select", options: mediaTypeOptions }, { key: "title", label: "标题" }],
    fields: [
      { key: "title", label: "标题" },
      { key: "filename", label: "文件名" },
      { key: "url", label: "URL", required: true, upload: "media" },
      { key: "type", label: "类型", type: "select", autoFilled: true, options: mediaTypeOptions },
    ],
    canEdit: false,
  },
  {
    resource: "file-recycle-bin",
    label: "文件回收站",
    singular: "回收文件",
    description: "帖子删除后的本地文件保留与提前清理",
    icon: Trash2,
    tone: "slate",
    columns: [
      { key: "original_url", label: "文件", render: (row) => <LongTextCell value={row.original_url || row.original_path} lines={2} />, className: "w-[360px]" },
      { key: "post_id", label: "帖子 ID", render: (row) => fieldText(row, "post_id"), className: "w-[110px]" },
      { key: "kind", label: "类型", render: (row) => <StatusPill value={fieldText(row, "kind")} tone="blue" />, className: "w-[130px]" },
      { key: "size_bytes", label: "大小", render: (row) => fieldBytes(row, "size_bytes"), className: "w-[110px]" },
      { key: "deleted_at", label: "删除时间", render: (row) => formatDateTime(row.deleted_at), className: "w-[170px]" },
      { key: "purge_after", label: "即将删除时间", render: (row) => formatDateTime(row.purge_after), className: "w-[170px]" },
      { key: "remaining", labelKey: "fileRecycleBin.columns.remaining", render: (row) => <FileRecycleRemainingCell purgeAfter={row.purge_after} status={row.status} />, className: "w-[150px]" },
      { key: "status", label: "状态", render: (row) => <StatusPill value={fieldText(row, "status")} tone={fileRecycleStatusTone(row.status)} />, className: "w-[120px]" },
    ],
    filters: [
      { key: "status", label: "状态", type: "select", options: [
        { value: "recycled", label: "已回收" },
        { value: "missing", label: "文件缺失" },
        { value: "skipped", label: "已跳过" },
        { value: "failed", label: "回收失败" },
        { value: "purged", label: "已清理" },
        { value: "purge_failed", label: "清理失败" },
      ] },
      { key: "kind", label: "类型" },
      { key: "post_id", label: "帖子 ID" },
    ],
    fields: [
      { key: "original_url", label: "原始 URL" },
      { key: "original_path", label: "原始路径" },
      { key: "recycled_path", label: "回收路径" },
      { key: "kind", label: "类型" },
      { key: "status", label: "状态" },
      { key: "deleted_at", label: "删除时间", type: "datetime" },
      { key: "purge_after", label: "即将删除时间", type: "datetime" },
      { key: "error", label: "错误", type: "textarea" },
    ],
    defaultSort: "purge_after",
    canCreate: false,
    canEdit: false,
    canDelete: true,
    canBulkDelete: true,
    tableMinWidth: 1330,
  },
];
