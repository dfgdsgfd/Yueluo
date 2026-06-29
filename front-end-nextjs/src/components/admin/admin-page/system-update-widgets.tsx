"use client";
import {
  Activity,
  Loader2,
  Package
} from "lucide-react";
import type {
  SystemUpdateCheck,
  SystemUpdateJobPayload,
  SystemUpdateReleaseOptionPayload,
  SystemUpdateToolStatus,
  SystemUpdateVersionPayload
} from "@/lib/types";
import {
  Tone
} from "./types";
import {
  EmptyBlock
} from "./resource-editor";
import {
  StatusPill
} from "./resource-cells";
import {
  InfoTile
} from "./operations-widgets";
import {
  formatBytes,
  formatDateTime,
  systemUpdateReleaseOptionLabel,
  systemUpdateSelectedRelease,
  systemUpdateStartCommand
} from "./helpers";

export function SystemUpdateInput({ label, value, onChange, placeholder }: { label: string; value: string; onChange: (value: string) => void; placeholder?: string }) {
  return (
    <label className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
      <span className="mb-2 block text-sm font-semibold text-[#343944]">{label}</span>
      <input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="h-10 w-full rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
        placeholder={placeholder}
      />
    </label>
  );
}


export function SystemUpdateSelect({ label, value, onChange, options }: { label: string; value: string; onChange: (value: string) => void; options: Array<{ value: string; label: string; hint?: string }> }) {
  return (
    <label className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
      <span className="mb-2 block text-sm font-semibold text-[#343944]">{label}</span>
      <select
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="h-10 w-full rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>{option.label}</option>
        ))}
      </select>
      <span className="mt-1 block text-xs text-[#8b919e]">{options.find((option) => option.value === value)?.hint ?? systemUpdateStartCommand(value)}</span>
    </label>
  );
}


export function SystemUpdateReleasePicker({
  label,
  value,
  releases,
  error,
  loading,
  onChange,
}: {
  label: string;
  value: string;
  releases: SystemUpdateReleaseOptionPayload[];
  error?: string;
  loading: boolean;
  onChange: (value: string) => void;
}) {
  const selected = systemUpdateSelectedRelease(value, releases);
  return (
    <label className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
      <span className="mb-2 flex items-center justify-between gap-2 text-sm font-semibold text-[#343944]">
        <span>{label}</span>
        {loading ? <Loader2 className="size-4 animate-spin text-[#1d4ed8]" /> : null}
      </span>
      <select
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="h-10 w-full rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
      >
        <option value="latest">latest（最新 Release）</option>
        {releases.map((release) => (
          <option key={release.tag_name} value={release.tag_name}>
            {systemUpdateReleaseOptionLabel(release)}
          </option>
        ))}
      </select>
      {error ? <p className="mt-2 break-words rounded-md border border-[#f59e0b]/20 bg-[#fffbeb] px-2 py-1.5 text-xs text-[#92400e]">{error}</p> : null}
      {selected ? <SystemUpdateReleaseSummary release={selected} /> : <p className="mt-2 text-xs text-[#8b919e]">暂无可匹配的 Release。保存仓库配置后可刷新版本。</p>}
    </label>
  );
}


export function SystemUpdateReleaseSummary({ release }: { release: SystemUpdateReleaseOptionPayload }) {
  const assets = release.matching_assets ?? [];
  return (
    <div className="mt-2 rounded-md border border-black/[0.06] bg-white p-2 text-xs text-[#4e5561]">
      <div className="mb-1.5 flex flex-wrap items-center gap-x-3 gap-y-1">
        <span className="font-semibold text-[#252932]">{release.tag_name}</span>
        <span>发布 {formatDateTime(release.published_at ?? release.created_at)}</span>
        {systemUpdateReleaseHash(release) ? <span>commit {shortHash(systemUpdateReleaseHash(release))}</span> : null}
      </div>
      <div className="grid gap-1.5">
        {assets.map((asset) => (
          <div key={asset.name} className="grid gap-1 rounded bg-[#f8fafc] px-2 py-1.5">
            <p className="break-all font-semibold text-[#252932]">{asset.name} · {formatBytes(asset.size_bytes ?? 0)}</p>
            <p className="break-all">SHA256 {asset.sha256 || "未提供"}</p>
            <p>更新时间 {formatDateTime(asset.updated_at)}</p>
          </div>
        ))}
      </div>
    </div>
  );
}


export function SystemUpdateToolGrid({ tools }: { tools: SystemUpdateToolStatus[] }) {
  if (!tools.length) return <EmptyBlock icon={Activity} label="暂无工具链状态" />;
  return (
    <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-4">
      {tools.map((tool) => (
        <article key={tool.name} className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
          <div className="mb-2 flex items-center justify-between gap-2">
            <h3 className="truncate text-sm font-semibold text-[#252932]">{tool.name}</h3>
            <StatusPill value={tool.ok ? "可用" : "缺失"} tone={tool.ok ? "green" : "red"} />
          </div>
          <p className="truncate text-xs text-[#6f7582]">{tool.version || tool.message || "-"}</p>
          {tool.path ? <p className="mt-1 truncate text-[11px] text-[#9aa0aa]">{tool.path}</p> : null}
        </article>
      ))}
    </div>
  );
}


export function SystemUpdateCheckGrid({ checks }: { checks: SystemUpdateCheck[] }) {
  return (
    <div className="grid gap-2 md:grid-cols-2">
      {checks.map((check) => (
        <article key={check.key} className="rounded-lg border border-black/[0.06] bg-white p-3">
          <div className="mb-2 flex items-center justify-between gap-2">
            <h3 className="truncate text-sm font-semibold text-[#252932]">{check.label}</h3>
            <StatusPill value={systemUpdateStatusLabel(check.status)} tone={systemUpdateStatusTone(check.status)} />
          </div>
          <p className="break-words text-xs text-[#6f7582]">{check.message || "-"}</p>
          {check.path ? <p className="mt-1 break-all text-[11px] text-[#9aa0aa]">{check.path}</p> : null}
        </article>
      ))}
    </div>
  );
}


export function SystemUpdateJobCard({ job }: { job: SystemUpdateJobPayload | null }) {
  if (!job) return <EmptyBlock icon={Package} label="暂无更新任务" />;
  const artifacts = job.artifact_paths ?? [];
  const artifactMeta = job.artifacts ?? [];
  const progress = Math.max(0, Math.min(100, Number(job.progress ?? (job.status === "succeeded" || job.status === "failed" ? 100 : 0))));
  return (
    <article className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
      <div className="mb-3 flex min-w-0 flex-wrap items-center justify-between gap-2">
        <div className="min-w-0">
          <h3 className="truncate text-sm font-semibold text-[#252932]">最近任务 #{String(job.id)}</h3>
          <p className="truncate text-xs text-[#8b919e]">{formatDateTime(job.started_at ?? job.created_at)} - {formatDateTime(job.finished_at)}</p>
        </div>
        <StatusPill value={systemUpdateStatusLabel(job.status)} tone={systemUpdateStatusTone(job.status)} />
      </div>
      <div className="mb-3 grid gap-2 sm:grid-cols-4">
        <InfoTile label="前端" value={systemUpdateStatusLabel(job.frontend_state)} />
        <InfoTile label="后端" value={systemUpdateStatusLabel(job.backend_state)} />
        <InfoTile label="进度" value={`${progress}%`} />
        <InfoTile label="产物" value={`${artifactMeta.length || artifacts.length} 个`} />
      </div>
      <div className="mb-3 rounded-lg border border-black/[0.06] bg-white p-2">
        <div className="mb-1 flex items-center justify-between gap-2 text-xs text-[#6f7582]">
          <span className="truncate">{job.current_step || systemUpdateStatusLabel(job.status)}</span>
          <span className="shrink-0 font-semibold text-[#252932]">{progress}%</span>
        </div>
        <div className="h-2 overflow-hidden rounded-full bg-[#e5e7eb]">
          <div className="h-full rounded-full bg-[#1d4ed8] transition-all" style={{ width: `${progress}%` }} />
        </div>
      </div>
      {job.error_message ? <p className="mb-3 rounded-lg border border-[#dc2626]/15 bg-[#fff1f2] px-3 py-2 text-sm text-[#b91c1c]">{job.error_message}</p> : null}
      {artifactMeta.length ? (
        <div className="mb-3 grid gap-2">
          {artifactMeta.map((item, index) => (
            <article key={`${item.kind}-${item.sha256 || index}`} className="rounded-lg border border-black/[0.06] bg-white p-3">
              <div className="mb-2 flex min-w-0 flex-wrap items-center justify-between gap-2">
                <div className="min-w-0">
                  <h4 className="truncate text-sm font-semibold text-[#252932]">{item.kind === "frontend" ? "前端" : "后端"} · {item.name}</h4>
                  <p className="truncate text-xs text-[#8b919e]">{formatBytes(item.size_bytes ?? 0)} · {formatDateTime(item.downloaded_at)}</p>
                </div>
                {item.release_tag ? <StatusPill value={item.release_tag} tone="blue" /> : null}
              </div>
              <div className="grid gap-1.5 text-xs text-[#4e5561]">
                <p className="break-all"><span className="font-semibold text-[#252932]">SHA256：</span>{item.sha256 || "-"}</p>
                <p className="break-all"><span className="font-semibold text-[#252932]">安装位置：</span>{item.install_path || "-"}</p>
                <p className="break-all"><span className="font-semibold text-[#252932]">缓存文件：</span>{item.cache_path || "-"}</p>
                {item.github_updated_at ? <p><span className="font-semibold text-[#252932]">Release 更新时间：</span>{formatDateTime(item.github_updated_at)}</p> : null}
              </div>
            </article>
          ))}
        </div>
      ) : null}
      {artifacts.length ? (
        <div className="mb-3 grid gap-1.5">
          {artifacts.map((path) => (
            <code key={path} className="break-all rounded-md bg-white px-2 py-1 text-xs text-[#3f4652]">{path}</code>
          ))}
        </div>
      ) : null}
      {job.logs ? (
        <pre className="max-h-[360px] overflow-auto rounded-lg bg-[#111827] p-3 text-xs leading-5 text-[#e5e7eb]">{job.logs}</pre>
      ) : null}
    </article>
  );
}


export function SystemUpdateVersionCompare({
  currentBackend,
  currentFrontend,
  latestBackend,
  latestFrontend,
}: {
  currentBackend: SystemUpdateVersionPayload | undefined | null;
  currentFrontend: SystemUpdateVersionPayload | undefined | null;
  latestBackend: SystemUpdateReleaseOptionPayload | undefined;
  latestFrontend: SystemUpdateReleaseOptionPayload | undefined;
}) {
  const backendHash = currentBackend?.commit_hash?.trim() || "";
  const frontendHash = currentFrontend?.commit_hash?.trim() || "";
  const hashesMatch = backendHash && frontendHash && backendHash === frontendHash;
  const hasBackend = !!backendHash && backendHash !== "unknown";
  const hasFrontend = !!frontendHash && frontendHash !== "unknown";
  const canCompare = hasBackend && hasFrontend;

  const latestBackendHash = systemUpdateReleaseHash(latestBackend) || "";
  const latestFrontendHash = systemUpdateReleaseHash(latestFrontend) || "";
  const backendBehind = hasBackend && latestBackendHash && backendHash !== latestBackendHash;
  const frontendBehind = hasFrontend && latestFrontendHash && frontendHash !== latestFrontendHash;
  const hasLatestBackend = !!latestBackendHash;
  const hasLatestFrontend = !!latestFrontendHash;

  return (
    <div className="grid gap-3">
      <div className="grid gap-3 md:grid-cols-2">
        <VersionCard
          side="后端"
          tag={currentBackend?.version_tag}
          hash={backendHash}
          source={currentBackend?.source}
          highlight={canCompare && !hashesMatch ? "red" : undefined}
          note={canCompare && !hashesMatch ? "与前端版本不一致，需要重启后端才能识别新版本" : undefined}
        />
        <VersionCard
          side="前端"
          tag={currentFrontend?.version_tag}
          hash={frontendHash}
          source={currentFrontend?.source}
          highlight={canCompare && !hashesMatch ? "red" : undefined}
          note={canCompare && !hashesMatch ? "与后端版本不一致，需要重新编译部署前端才能生效" : undefined}
        />
      </div>
      {(hasLatestBackend || hasLatestFrontend) && (
        <div className="grid gap-3 md:grid-cols-2">
          {hasLatestBackend && (
            <VersionCard
              side="GitHub latest 后端"
              tag={latestBackend?.tag_name}
              hash={latestBackendHash}
              highlight={backendBehind ? "amber" : "green"}
              note={backendBehind ? "当前后端版本落后，建议更新" : "当前后端已是最新"}
            />
          )}
          {hasLatestFrontend && (
            <VersionCard
              side="GitHub latest 前端"
              tag={latestFrontend?.tag_name}
              hash={latestFrontendHash}
              highlight={frontendBehind ? "amber" : "green"}
              note={frontendBehind ? "当前前端版本落后，建议更新" : "当前前端已是最新"}
            />
          )}
        </div>
      )}
    </div>
  );
}


export function VersionCard({
  side,
  tag,
  hash,
  source,
  highlight,
  note,
}: {
  side: string;
  tag?: string;
  hash: string;
  source?: string;
  highlight?: "red" | "amber" | "green";
  note?: string;
}) {
  const borderColor = highlight === "red" ? "border-red-300" : highlight === "amber" ? "border-amber-300" : highlight === "green" ? "border-green-300" : "border-black/[0.06]";
  const bgColor = highlight === "red" ? "bg-red-50" : highlight === "amber" ? "bg-amber-50" : highlight === "green" ? "bg-green-50" : "bg-[#fafbfe]";
  const dotColor = highlight === "red" ? "bg-red-500" : highlight === "amber" ? "bg-amber-500" : highlight === "green" ? "bg-green-500" : "bg-[#8b919e]";

  return (
    <div className={`rounded-lg border ${borderColor} ${bgColor} p-3`}>
      <div className="mb-2 flex items-center gap-2">
        <span className={`size-2 rounded-full ${dotColor}`} />
        <span className="text-xs font-semibold text-[#343944]">{side}</span>
      </div>
      <div className="space-y-1 text-xs text-[#4e5561]">
        {tag && tag !== "unknown" && <p>版本: {tag}</p>}
        <p className="font-mono">哈希: {hash ? shortHash(hash) : "未知"}</p>
        {source && <p>来源: {source}</p>}
      </div>
      {note && (
        <p className="mt-2 text-xs font-medium text-red-600">{note}</p>
      )}
    </div>
  );
}


export function systemUpdateVersionLabel(version?: { version_tag?: string; commit_hash?: string; source?: string } | null) {
  if (!version) return "未找到 version.json";
  const hash = shortHash(version.commit_hash);
  const tag = version.version_tag && version.version_tag !== "unknown" ? version.version_tag : "";
  return [hash, tag, version.source].filter(Boolean).join(" · ");
}


export function systemUpdateReleaseHash(release?: SystemUpdateReleaseOptionPayload | null) {
  return String(release?.commit_hash || release?.target_commit || "");
}


export function systemUpdateReleaseHashLabel(release?: SystemUpdateReleaseOptionPayload | null) {
  const hash = systemUpdateReleaseHash(release);
  if (!hash) return "未加载";
  return [shortHash(hash), release?.tag_name].filter(Boolean).join(" · ");
}


export function shortHash(value: unknown) {
  const raw = String(value ?? "");
  return raw.length > 12 ? raw.slice(0, 12) : raw || "-";
}


export function systemUpdateStatusLabel(value: unknown) {
  const raw = String(value ?? "");
  if (raw === "succeeded") return "成功";
  if (raw === "failed") return "失败";
  if (raw === "running") return "执行中";
  if (raw === "pending") return "等待";
  if (raw === "skipped") return "跳过";
  return raw || "-";
}


export function systemUpdateStatusTone(value: unknown): Tone {
  const raw = String(value ?? "");
  if (raw === "succeeded") return "green";
  if (raw === "failed") return "red";
  if (raw === "running") return "blue";
  if (raw === "pending") return "amber";
  return "slate";
}
