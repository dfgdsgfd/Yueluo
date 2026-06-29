"use client";
import {
  Activity,
  ClipboardList,
  Package
} from "lucide-react";
import type {
  AdminListRow
} from "@/lib/types";
import {
  ActivityItem,
  ApkFileItem,
  AppVersionStatsPayload,
  BatchUploadFilesPayload,
  ContentReviewSettingsPayload,
  MissingCoverStats,
  OperationResult
} from "./types";
import {
  Panel
} from "./layout-widgets";
import {
  EmptyBlock,
  KeyValueGrid
} from "./resource-editor";
import {
  StatusPill
} from "./resource-cells";
import {
  activityTypeLabel,
  formatBytes,
  formatCompact,
  formatDateTime,
  formatDuration,
  readableValue
} from "./helpers";

export function ActivityList({ items }: { items: ActivityItem[] }) {
  if (!items.length) return <EmptyBlock icon={Activity} label="暂无活动" />;
  return (
    <div className="grid gap-2">
      {items.slice(0, 12).map((item, index) => (
        <article key={String(item.id ?? index)} className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
          <div className="mb-1 flex items-center justify-between gap-3">
            <h3 className="truncate text-sm font-semibold text-[#252932]">{item.title || activityTypeLabel(item.type)}</h3>
            <span className="text-[11px] text-[#8b919e]">{formatDateTime(item.created_at)}</span>
          </div>
          <p className="line-clamp-2 text-xs text-[#6f7582]">{item.content || item.nickname || item.user_id || "-"}</p>
        </article>
      ))}
    </div>
  );
}


export function FileList({ files, empty }: { files: ApkFileItem[]; empty: string }) {
  if (!files.length) return <EmptyBlock icon={Package} label={empty} />;
  return (
    <div className="grid gap-2">
      {files.slice(0, 12).map((file, index) => (
        <a key={`${file.name ?? index}`} href={file.url || "#"} className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3 transition hover:bg-white">
          <div className="flex items-center justify-between gap-3">
            <span className="min-w-0 truncate text-sm font-semibold text-[#252932]">{file.name || "文件"}</span>
            <span className="text-xs text-[#8b919e]">{formatBytes(file.size)}</span>
          </div>
          <p className="mt-1 truncate text-xs text-[#8b919e]">{formatDateTime(file.createdAt ?? file.created_at)}</p>
        </a>
      ))}
    </div>
  );
}


export function BatchFilesPanel({ payload }: { payload: BatchUploadFilesPayload | null }) {
  const images = payload?.images ?? [];
  const videos = payload?.videos ?? [];
  return (
    <div className="grid gap-3">
      <div className="grid grid-cols-2 gap-2">
        <InfoTile label="图片素材" value={formatCompact(images.length)} />
        <InfoTile label="视频素材" value={formatCompact(videos.length)} />
      </div>
      <FileList files={[...images, ...videos].slice(0, 8)} empty="暂无批量素材" />
    </div>
  );
}


export function CoverStatsPanel({ stats, limit, onLimitChange }: { stats: MissingCoverStats | null; limit: string; onLimitChange: (value: string) => void }) {
  return (
    <div className="grid gap-3">
      <div className="grid grid-cols-3 gap-2">
        <InfoTile label="缺封面" value={formatCompact(stats?.total ?? 0)} />
        <InfoTile label="本地可处理" value={formatCompact(stats?.accessible ?? 0)} />
        <InfoTile label="远程视频" value={formatCompact(stats?.remote ?? 0)} />
      </div>
      <label className="grid gap-1.5">
        <span className="text-xs font-semibold text-[#666c78]">单次处理上限</span>
        <input value={limit} onChange={(event) => onLimitChange(event.target.value)} type="number" min={1} max={200} className="h-10 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]" />
      </label>
    </div>
  );
}


export function ReviewSettingsPanel({ settings, loading, onChange }: { settings: ContentReviewSettingsPayload | null; loading: boolean; onChange: (settings: ContentReviewSettingsPayload) => void }) {
  const current = settings ?? {};
  const items: Array<[keyof ContentReviewSettingsPayload, string]> = [
    ["ai_auto_review", "自动审核"],
    ["ai_username_review", "用户名审核"],
    ["ai_content_review", "内容审核"],
  ];
  return (
    <div className="grid gap-2">
      {items.map(([key, label]) => {
        const active = Boolean(current[key]);
        return (
          <button
            key={key}
            type="button"
            disabled={loading}
            onClick={() => onChange({ ...current, [key]: !active })}
            className="flex items-center justify-between rounded-lg border border-black/[0.06] bg-[#fafbfe] px-3 py-2 text-left transition hover:bg-white"
          >
            <span className="text-sm font-semibold text-[#30333b]">{label}</span>
            <StatusPill value={active ? "开启" : "关闭"} tone={active ? "green" : "slate"} />
          </button>
        );
      })}
    </div>
  );
}


export function AppStatsPanel({ stats }: { stats: AppVersionStatsPayload | null }) {
  const duration = stats?.usage_duration ?? {};
  return (
    <div className="grid gap-3">
      <div className="grid gap-2 sm:grid-cols-4">
        <InfoTile label="累计设备" value={formatCompact(stats?.total_users ?? 0)} />
        <InfoTile label="今日活跃" value={formatCompact(stats?.today_active_users ?? 0)} />
        <InfoTile label="使用上报" value={formatCompact(duration.report_count ?? 0)} />
        <InfoTile label="平均时长" value={formatDuration(duration.avg_seconds)} />
      </div>
      <div className="grid gap-2 lg:grid-cols-2">
        <MiniList title="版本更新" items={stats?.version_updates ?? []} primaryKey="version_name" secondaryKey="update_count" />
        <MiniList title="平台分布" items={stats?.platform_stats ?? []} primaryKey="platform" secondaryKey="user_count" />
      </div>
    </div>
  );
}


export function OperationResultPanel({ result }: { result: OperationResult }) {
  const entries = Object.entries(result.values ?? {}).filter(([, value]) => value !== undefined && value !== null && value !== "");
  return (
    <Panel title={result.title} icon={ClipboardList}>
      <div className="grid gap-3">
        {result.message ? (
          <div className="rounded-lg border border-[#18a058]/20 bg-[#f0fff7] px-3 py-2 text-sm font-medium text-[#107a43]">
            {result.message}
          </div>
        ) : null}
        {entries.length ? <KeyValueGrid entries={entries} /> : <EmptyBlock icon={ClipboardList} label="暂无返回详情" />}
      </div>
    </Panel>
  );
}


export function SupportDataPanel({
  templateDefaults,
  testUsers,
  lastAppForm,
}: {
  templateDefaults: AdminListRow[];
  testUsers: AdminListRow[];
  lastAppForm: Record<string, unknown>;
}) {
  const lastFormEntries = Object.entries(lastAppForm).filter(([, value]) => value !== undefined && value !== null && value !== "");
  return (
    <div className="grid gap-3">
      <div className="grid gap-2 sm:grid-cols-3">
        <InfoTile label="默认模板" value={formatCompact(templateDefaults.length)} />
        <InfoTile label="测试用户" value={formatCompact(testUsers.length)} />
        <InfoTile label="版本草稿" value={lastFormEntries.length ? "可恢复" : "暂无"} />
      </div>
      <div className="grid gap-3 lg:grid-cols-2">
        <CompactRecordList title="默认模板" items={templateDefaults} primaryKeys={["name", "template_key"]} secondaryKeys={["type", "subject"]} />
        <CompactRecordList title="测试用户" items={testUsers} primaryKeys={["nickname", "user_id", "email"]} secondaryKeys={["email", "user_id", "id"]} />
      </div>
      {lastFormEntries.length ? (
        <div className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
          <h3 className="mb-2 text-sm font-semibold text-[#30333b]">上次 App 填写</h3>
          <KeyValueGrid entries={lastFormEntries.slice(0, 8)} />
        </div>
      ) : null}
    </div>
  );
}


export function CompactRecordList({
  title,
  items,
  primaryKeys,
  secondaryKeys,
}: {
  title: string;
  items: Array<Record<string, unknown>>;
  primaryKeys: string[];
  secondaryKeys: string[];
}) {
  return (
    <div className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
      <h3 className="mb-2 text-sm font-semibold text-[#30333b]">{title}</h3>
      {items.length ? (
        <div className="grid gap-1">
          {items.slice(0, 6).map((item, index) => (
            <div key={index} className="flex min-w-0 items-center justify-between gap-3 rounded-md bg-white px-2 py-1.5 text-sm">
              <span className="min-w-0 truncate text-[#555b66]">{firstReadable(item, primaryKeys)}</span>
              <span className="min-w-0 truncate text-right text-xs font-semibold text-[#17171d]">{firstReadable(item, secondaryKeys)}</span>
            </div>
          ))}
          {items.length > 6 ? <p className="px-2 pt-1 text-xs text-[#8b919e]">还有 {formatCompact(items.length - 6)} 项</p> : null}
        </div>
      ) : (
        <p className="text-xs text-[#8b919e]">暂无数据</p>
      )}
    </div>
  );
}


export function firstReadable(item: Record<string, unknown>, keys: string[]) {
  for (const key of keys) {
    const value = item[key];
    if (value !== undefined && value !== null && value !== "") return readableValue(value);
  }
  return "-";
}


export function MiniList({ title, items, primaryKey, secondaryKey }: { title: string; items: Array<Record<string, unknown>>; primaryKey: string; secondaryKey: string }) {
  return (
    <div className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
      <h3 className="mb-2 text-sm font-semibold text-[#30333b]">{title}</h3>
      {items.length ? (
        <div className="grid gap-1">
          {items.slice(0, 8).map((item, index) => (
            <div key={index} className="flex items-center justify-between gap-3 rounded-md bg-white px-2 py-1.5 text-sm">
              <span className="truncate text-[#555b66]">{readableValue(item[primaryKey])}</span>
              <span className="font-semibold text-[#17171d]">{formatCompact(item[secondaryKey])}</span>
            </div>
          ))}
        </div>
      ) : (
        <p className="text-xs text-[#8b919e]">暂无数据</p>
      )}
    </div>
  );
}


export function InfoTile({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg bg-white px-3 py-2">
      <p className="text-xs text-[#8b919e]">{label}</p>
      <p className="font-semibold text-[#252932]">{value}</p>
    </div>
  );
}
