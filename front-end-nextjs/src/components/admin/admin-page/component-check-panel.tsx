"use client";
import {
  useCallback,
  useEffect,
  useState
} from "react";
import {
  Check,
  Loader2,
  Package,
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
  SystemUpdateJobPayload,
  SystemUpdateStatusPayload
} from "@/lib/types";
import {
  Panel
} from "./layout-widgets";
import {
  LoadingBlock
} from "./resource-editor";
import {
  SystemUpdateCheckGrid
} from "./system-update-widgets";
import {
  errorMessage
} from "./helpers";


export function ComponentCheckPanel({ token }: { token: string }) {
  const [status, setStatus] = useState<SystemUpdateStatusPayload | null>(null);
  const [loading, setLoading] = useState(true);
  const [checking, setChecking] = useState(false);
  const [componentRunning, setComponentRunning] = useState<string | null>(null);
  const isJobRunning = status?.last_job?.status === "running";

  const load = useCallback(async (showLoading = true) => {
    if (showLoading) setLoading(true);
    try {
      const data = await adminRequest<SystemUpdateStatusPayload>("/api/admin/system-update/status", { method: "GET", token });
      setStatus(data);
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      if (showLoading) setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    queueMicrotask(() => { void load(); });
  }, [load]);

  useEffect(() => {
    if (!isJobRunning) return;
    const timer = window.setInterval(() => { void load(false); }, 2500);
    return () => window.clearInterval(timer);
  }, [isJobRunning, load]);

  async function check() {
    setChecking(true);
    try {
      const data = await adminRequest<SystemUpdateStatusPayload>("/api/admin/system-update/check", { method: "POST", token });
      setStatus(data);
      toast.success("组件检查完成");
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setChecking(false);
    }
  }

  async function runComponentAction(action: "ffmpeg_test" | "geoip_update") {
    setComponentRunning(action);
    try {
      const job = await adminRequest<SystemUpdateJobPayload>("/api/admin/system-update/run", {
        method: "POST",
        token,
        body: JSON.stringify({ action }),
      });
      setStatus((current) => current ? { ...current, last_job: job } : current);
      toast.success(action === "ffmpeg_test" ? "FFmpeg 测试已启动" : "GeoIP 数据库更新已启动");
      void load(false);
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setComponentRunning(null);
    }
  }

  const componentChecks = status?.component_checks ?? [];

  return (
    <div className="grid gap-4">
      <Panel
        title="组件检查与测试"
        icon={Package}
        action={
          <div className="flex shrink-0 flex-wrap gap-2">
            <Button type="button" variant="outline" disabled={checking || loading} onClick={() => void check()} className="h-9 rounded-lg border-black/[0.08] bg-white">
              {checking ? <Loader2 className="size-4 animate-spin" /> : <Check className="size-4" />}
              运行检查
            </Button>
            <Button type="button" disabled={componentRunning === "ffmpeg_test" || isJobRunning} onClick={() => void runComponentAction("ffmpeg_test")} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]">
              {componentRunning === "ffmpeg_test" ? <Loader2 className="size-4 animate-spin" /> : <Check className="size-4" />}
              FFmpeg 测试
            </Button>
            <Button type="button" disabled={componentRunning === "geoip_update" || isJobRunning} onClick={() => void runComponentAction("geoip_update")} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]">
              {componentRunning === "geoip_update" ? <Loader2 className="size-4 animate-spin" /> : <UploadCloud className="size-4" />}
              更新 GeoIP
            </Button>
          </div>
        }
      >
        {loading ? (
          <LoadingBlock label="正在加载组件状态" />
        ) : componentChecks.length ? (
          <SystemUpdateCheckGrid checks={componentChecks} />
        ) : (
          <LoadingBlock label="点击运行检查以获取组件状态" />
        )}
      </Panel>
    </div>
  );
}
