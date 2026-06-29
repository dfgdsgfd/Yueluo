"use client";
import {
  useCallback,
  useEffect,
  useState
} from "react";
import {
  Activity,
  Check,
  Loader2,
  Package,
  RefreshCw,
  Save,
  Settings,
  UploadCloud
} from "lucide-react";
import {
  toast
} from "sonner";
import {
  Button
} from "@/components/ui/button";
import {
  adminRequest
} from "@/lib/api";
import type {
  SystemUpdateConfigPayload,
  SystemUpdateJobPayload,
  SystemUpdateReleaseOptionsPayload,
  SystemUpdateStatusPayload
} from "@/lib/types";
import {
  cn
} from "@/lib/utils";
import {
  systemUpdateStartModeOptions
} from "./types";
import {
  HeaderCard,
  Panel
} from "./layout-widgets";
import {
  LoadingBlock
} from "./resource-editor";
import {
  SystemUpdateCheckGrid,
  SystemUpdateInput,
  SystemUpdateJobCard,
  SystemUpdateReleasePicker,
  SystemUpdateSelect,
  SystemUpdateToolGrid,
  SystemUpdateVersionCompare,
  systemUpdateReleaseHashLabel,
  systemUpdateVersionLabel
} from "./system-update-widgets";
import {
  InfoTile
} from "./operations-widgets";
import {
  errorMessage,
  systemUpdateDraftFromConfig,
  systemUpdateStartModeLabel
} from "./helpers";
import {
  ToggleSwitch
} from "./form-fields";

export type SystemUpdateDraft = {
  frontend_repo_url: string;
  backend_repo_url: string;
  frontend_release_tag: string;
  backend_release_tag: string;
  frontend_artifact_url: string;
  backend_artifact_url: string;
  frontend_asset_pattern: string;
  backend_asset_pattern: string;
  frontend_install_dir: string;
  backend_install_path: string;
  frontend_start_mode: string;
  artifact_dir: string;
};


export function SystemUpdatePanel({ token }: { token: string }) {
  const [status, setStatus] = useState<SystemUpdateStatusPayload | null>(null);
  const [releaseOptions, setReleaseOptions] = useState<SystemUpdateReleaseOptionsPayload | null>(null);
  const [draft, setDraft] = useState<SystemUpdateDraft>(() => systemUpdateDraftFromConfig());
  const [tokenDraft, setTokenDraft] = useState("");
  const [clearToken, setClearToken] = useState(false);
  const [loading, setLoading] = useState(true);
  const [loadingReleases, setLoadingReleases] = useState(false);
  const [saving, setSaving] = useState(false);
  const [checking, setChecking] = useState(false);
  const [running, setRunning] = useState(false);
  const [runFrontend, setRunFrontend] = useState(true);
  const [runBackend, setRunBackend] = useState(true);
  const isJobRunning = status?.last_job?.status === "running";

  const load = useCallback(async (showLoading = true) => {
    if (showLoading) setLoading(true);
    try {
      const data = await adminRequest<SystemUpdateStatusPayload>("/api/admin/system-update/status", { method: "GET", token });
      setStatus(data);
      setDraft(systemUpdateDraftFromConfig(data.config));
      setTokenDraft("");
      setClearToken(false);
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      if (showLoading) setLoading(false);
    }
  }, [token]);

  const loadReleases = useCallback(async (showToast = false) => {
    setLoadingReleases(true);
    try {
      const data = await adminRequest<SystemUpdateReleaseOptionsPayload>("/api/admin/system-update/releases", { method: "GET", token });
      setReleaseOptions(data);
      if (showToast) toast.success("版本列表已刷新");
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setLoadingReleases(false);
    }
  }, [token]);

  useEffect(() => {
    queueMicrotask(() => {
      void load();
      void loadReleases();
    });
  }, [load, loadReleases]);

  useEffect(() => {
    if (!isJobRunning) return;
    const timer = window.setInterval(() => {
      void load(false);
    }, 2500);
    return () => window.clearInterval(timer);
  }, [isJobRunning, load]);

  async function save() {
    setSaving(true);
    try {
      const config = await adminRequest<SystemUpdateConfigPayload>("/api/admin/system-update/config", {
        method: "PUT",
        token,
        body: JSON.stringify({ ...draft, github_token: tokenDraft.trim() || undefined, clear_github_token: clearToken }),
      });
      setStatus((current) => current ? { ...current, config } : current);
      setDraft(systemUpdateDraftFromConfig(config));
      setTokenDraft("");
      setClearToken(false);
      toast.success("更新配置已保存");
      void loadReleases();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function check() {
    setChecking(true);
    try {
      const data = await adminRequest<SystemUpdateStatusPayload>("/api/admin/system-update/check", { method: "POST", token });
      setStatus(data);
      setDraft(systemUpdateDraftFromConfig(data.config));
      toast.success("状态检查完成");
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setChecking(false);
    }
  }

  async function run() {
    if (!runFrontend && !runBackend) {
      toast.error("至少选择前端或后端");
      return;
    }
    setRunning(true);
    try {
      const job = await adminRequest<SystemUpdateJobPayload>("/api/admin/system-update/run", {
        method: "POST",
        token,
        body: JSON.stringify({
          frontend: runFrontend,
          backend: runBackend,
          frontend_release_tag: draft.frontend_release_tag || undefined,
          backend_release_tag: draft.backend_release_tag || undefined,
        }),
      });
      setStatus((current) => current ? { ...current, last_job: job } : current);
      toast.success("更新任务已启动，可在下方查看进度");
      void load(false);
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setRunning(false);
    }
  }

  const tools = status?.environment?.tools ?? [];
  const checks = status?.checks ?? [];
  const lastJob = status?.last_job ?? null;
  const frontendReleases = releaseOptions?.frontend ?? [];
  const backendReleases = releaseOptions?.backend ?? [];

  return (
    <div className="grid gap-4">
      <HeaderCard icon={Package} title="系统更新" description="下载已编译产物，校验哈希并安装到容器挂载目录；重启由你手动控制" tone="blue" />

      <Panel
        title="状态检查"
        icon={Activity}
        action={
          <div className="flex shrink-0 flex-wrap gap-2">
            <Button type="button" variant="outline" disabled={loading || checking} onClick={() => void load()} className="h-9 rounded-lg border-black/[0.08] bg-white">
              <RefreshCw className={cn("size-4", loading ? "animate-spin" : "")} />
              刷新
            </Button>
            <Button type="button" disabled={checking} onClick={() => void check()} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]">
              {checking ? <Loader2 className="size-4 animate-spin" /> : <Check className="size-4" />}
              检查
            </Button>
          </div>
        }
      >
        {loading ? (
          <LoadingBlock label="正在读取更新状态" />
        ) : (
          <div className="grid gap-3">
            <div className="grid gap-2 md:grid-cols-3">
              <InfoTile label="系统" value={`${status?.environment?.os ?? "-"} / ${status?.environment?.arch ?? "-"}`} />
              <InfoTile label="前端目录" value={status?.config.frontend_install_dir ?? "-"} />
              <InfoTile label="后端文件" value={status?.config.backend_install_path ?? "-"} />
              <InfoTile label="下载缓存" value={status?.config.artifact_dir ?? "-"} />
              <InfoTile label="前端启动" value={systemUpdateStartModeLabel(status?.config.frontend_start_mode)} />
              <InfoTile label="密钥" value={status?.config.github_token_set ? `已配置 ${status.config.github_token_masked ?? ""}` : "未配置"} />
              <InfoTile label="当前后端哈希" value={systemUpdateVersionLabel(status?.current?.backend)} />
              <InfoTile label="当前前端哈希" value={systemUpdateVersionLabel(status?.current?.frontend)} />
              <InfoTile label="GitHub latest 后端" value={systemUpdateReleaseHashLabel(backendReleases[0])} />
              <InfoTile label="GitHub latest 前端" value={systemUpdateReleaseHashLabel(frontendReleases[0])} />
            </div>
            <SystemUpdateVersionCompare
              currentBackend={status?.current?.backend}
              currentFrontend={status?.current?.frontend}
              latestBackend={backendReleases[0]}
              latestFrontend={frontendReleases[0]}
            />
            <SystemUpdateToolGrid tools={tools} />
            {checks.length ? <SystemUpdateCheckGrid checks={checks} /> : null}
          </div>
        )}
      </Panel>

      <Panel
        title="更新配置"
        icon={Settings}
        action={
          <Button type="button" disabled={saving || loading} onClick={() => void save()} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]">
            {saving ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
            保存
          </Button>
        }
      >
        <div className="grid gap-4">
          <div className="grid gap-3 lg:grid-cols-2">
            <SystemUpdateInput label="前端产物 URL（优先）" value={draft.frontend_artifact_url} onChange={(value) => setDraft((current) => ({ ...current, frontend_artifact_url: value }))} placeholder="https://github.com/owner/repo/releases/download/v1/frontend-linux-amd64.zip" />
            <SystemUpdateInput label="后端产物 URL（优先）" value={draft.backend_artifact_url} onChange={(value) => setDraft((current) => ({ ...current, backend_artifact_url: value }))} placeholder="https://github.com/owner/repo/releases/download/v1/yuem-go-linux-amd64" />
            <SystemUpdateInput label="前端 GitHub 仓库" value={draft.frontend_repo_url} onChange={(value) => setDraft((current) => ({ ...current, frontend_repo_url: value }))} placeholder="https://github.com/owner/repo.git" />
            <SystemUpdateInput label="后端 GitHub 仓库" value={draft.backend_repo_url} onChange={(value) => setDraft((current) => ({ ...current, backend_repo_url: value }))} placeholder="https://github.com/owner/repo.git" />
            <SystemUpdateInput label="前端产物匹配" value={draft.frontend_asset_pattern} onChange={(value) => setDraft((current) => ({ ...current, frontend_asset_pattern: value }))} placeholder="frontend-linux-amd64.zip" />
            <SystemUpdateInput label="后端产物匹配" value={draft.backend_asset_pattern} onChange={(value) => setDraft((current) => ({ ...current, backend_asset_pattern: value }))} placeholder="yuem-go-linux-amd64" />
          </div>
          <div className="grid gap-3 lg:grid-cols-2">
            <SystemUpdateInput label="前端安装目录" value={draft.frontend_install_dir} onChange={(value) => setDraft((current) => ({ ...current, frontend_install_dir: value }))} />
            <SystemUpdateInput label="后端安装文件" value={draft.backend_install_path} onChange={(value) => setDraft((current) => ({ ...current, backend_install_path: value }))} />
            <SystemUpdateInput label="下载缓存目录" value={draft.artifact_dir} onChange={(value) => setDraft((current) => ({ ...current, artifact_dir: value }))} />
            <SystemUpdateSelect
              label="前端启动方式"
              value={draft.frontend_start_mode}
              onChange={(value) => setDraft((current) => ({ ...current, frontend_start_mode: value }))}
              options={systemUpdateStartModeOptions}
            />
          </div>
          <label className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
            <span className="mb-2 block text-sm font-semibold text-[#343944]">GitHub 访问密钥</span>
            <input
              value={tokenDraft}
              onChange={(event) => setTokenDraft(event.target.value)}
              className="h-10 w-full rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
              placeholder={status?.config.github_token_set ? `已保存 ${status.config.github_token_masked ?? ""}` : "github_pat_... 或 ghp_..."}
              type="password"
              autoComplete="new-password"
            />
            <div className="mt-2">
              <ToggleSwitch value={clearToken} onChange={setClearToken} onLabel="清空已保存密钥" offLabel="保留已保存密钥" />
            </div>
            <details className="mt-2">
              <summary className="cursor-pointer text-xs text-[#1d4ed8] hover:underline">如何创建密钥？</summary>
              <div className="mt-2 space-y-1.5 rounded-md bg-white p-2 text-xs text-[#4e5561]">
                <p><strong>Fine-grained PAT（推荐）：</strong></p>
                <ol className="ml-4 list-decimal space-y-1">
                  <li>前往 <a href="https://github.com/settings/tokens?type=beta" target="_blank" rel="noopener noreferrer" className="text-[#1d4ed8] hover:underline">GitHub Token 设置</a> 点击 &quot;Generate new token&quot;</li>
                  <li>Repository access 选择 &quot;Only select repositories&quot;，勾选你的仓库</li>
                  <li>Permissions 选择 &quot;Contents: Read-only&quot;（或 &quot;Releases: Read-only&quot;）</li>
                  <li>生成的密钥填入上方输入框并保存</li>
                </ol>
                <p className="mt-2"><strong>Classic PAT：</strong></p>
                <ol className="ml-4 list-decimal space-y-1">
                  <li>前往 <a href="https://github.com/settings/tokens" target="_blank" rel="noopener noreferrer" className="text-[#1d4ed8] hover:underline">GitHub Token 设置</a> 生成 classic token</li>
                  <li>勾选 <code className="rounded bg-[#f0f2f5] px-1">repo</code> 或 <code className="rounded bg-[#f0f2f5] px-1">public_repo</code> 作用域</li>
                </ol>
              </div>
            </details>
          </label>
        </div>
      </Panel>

      <Panel
        title="一键更新"
        icon={UploadCloud}
        action={
          <div className="flex shrink-0 flex-wrap gap-2">
            <Button type="button" variant="outline" disabled={loadingReleases} onClick={() => void loadReleases(true)} className="h-9 rounded-lg border-black/[0.08] bg-white">
              <RefreshCw className={cn("size-4", loadingReleases ? "animate-spin" : "")} />
              刷新版本
            </Button>
            <Button type="button" disabled={running || loading || isJobRunning} onClick={() => void run()} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]">
              {running || isJobRunning ? <Loader2 className="size-4 animate-spin" /> : <UploadCloud className="size-4" />}
              {isJobRunning ? "执行中" : "下载并安装"}
            </Button>
          </div>
        }
      >
        <div className="grid gap-3">
          <div className="grid gap-3 lg:grid-cols-2">
            <SystemUpdateReleasePicker
              label="前端版本"
              value={draft.frontend_release_tag}
              releases={frontendReleases}
              error={releaseOptions?.frontend_error}
              loading={loadingReleases}
              onChange={(value) => setDraft((current) => ({ ...current, frontend_release_tag: value }))}
            />
            <SystemUpdateReleasePicker
              label="后端版本"
              value={draft.backend_release_tag}
              releases={backendReleases}
              error={releaseOptions?.backend_error}
              loading={loadingReleases}
              onChange={(value) => setDraft((current) => ({ ...current, backend_release_tag: value }))}
            />
          </div>
          <div className="grid gap-2 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3 sm:grid-cols-2">
            <ToggleSwitch value={runFrontend} onChange={setRunFrontend} onLabel="更新前端" offLabel="跳过前端" />
            <ToggleSwitch value={runBackend} onChange={setRunBackend} onLabel="更新后端" offLabel="跳过后端" />
          </div>
          <SystemUpdateJobCard job={lastJob} />
        </div>
      </Panel>
    </div>
  );
}


