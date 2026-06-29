"use client";

import Link from "next/link";
import { useEffect, useMemo, useState, type FormEvent } from "react";
import {
  ArrowLeft,
  Check,
  Code2,
  Copy,
  ExternalLink,
  FileJson,
  KeyRound,
  Loader2,
  Plus,
  RefreshCw,
  Search,
  Shield,
  Trash2,
} from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  createUserApiKey,
  deleteUserApiKey,
  getBackendApiSpec,
  getStoredAccessToken,
  getSwaggerDocsUrl,
  getSwaggerJsonUrl,
  getUserApiKeys,
} from "@/lib/api";
import type { BackendApiOperation, BackendApiSpec, UserApiKey } from "@/lib/types";
import { cn } from "@/lib/utils";

const httpMethods = ["get", "post", "put", "patch", "delete", "head", "options"] as const;

type HttpMethod = (typeof httpMethods)[number];
type MethodFilter = "all" | HttpMethod;

type OperationEntry = {
  method: HttpMethod;
  operation: BackendApiOperation;
  path: string;
};

const methodLabels: Record<HttpMethod, string> = {
  delete: "DELETE",
  get: "GET",
  head: "HEAD",
  options: "OPTIONS",
  patch: "PATCH",
  post: "POST",
  put: "PUT",
};

export function BackendApiPage() {
  const [authChecked, setAuthChecked] = useState(false);
  const [authToken, setAuthToken] = useState<string | null>(null);
  const [spec, setSpec] = useState<BackendApiSpec | null>(null);
  const [apiKeys, setApiKeys] = useState<UserApiKey[]>([]);
  const [createdApiKey, setCreatedApiKey] = useState<UserApiKey | null>(null);
  const [keyName, setKeyName] = useState("");
  const [methodFilter, setMethodFilter] = useState<MethodFilter>("all");
  const [query, setQuery] = useState("");
  const [copiedValue, setCopiedValue] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | number | null>(null);
  const [isCreatingKey, setIsCreatingKey] = useState(false);
  const [isLoadingKeys, setIsLoadingKeys] = useState(false);
  const [isLoadingSpec, setIsLoadingSpec] = useState(true);

  const docsUrl = getSwaggerDocsUrl();
  const jsonUrl = getSwaggerJsonUrl();
  const operations = useMemo(() => flattenOperations(spec), [spec]);
  const filteredOperations = useMemo(
    () => filterOperations(operations, methodFilter, query),
    [methodFilter, operations, query],
  );
  const methodCounts = useMemo(() => {
    const counts = new Map<HttpMethod, number>();
    for (const operation of operations) {
      counts.set(operation.method, (counts.get(operation.method) ?? 0) + 1);
    }
    return counts;
  }, [operations]);

  useEffect(() => {
    void loadSpec();
    let cancelled = false;
    queueMicrotask(() => {
      if (cancelled) {
        return;
      }

      const token = getStoredAccessToken();
      setAuthToken(token);
      setAuthChecked(true);
      if (token) {
        void loadApiKeys();
      }
    });

    return () => {
      cancelled = true;
    };
  }, []);

  async function loadSpec() {
    setIsLoadingSpec(true);
    try {
      setSpec(await getBackendApiSpec());
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "后端 API 加载失败");
    } finally {
      setIsLoadingSpec(false);
    }
  }

  async function loadApiKeys() {
    setIsLoadingKeys(true);
    try {
      setApiKeys(await getUserApiKeys());
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "API Key 加载失败");
    } finally {
      setIsLoadingKeys(false);
    }
  }

  async function handleCreateKey(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!keyName.trim() || isCreatingKey) {
      return;
    }

    setIsCreatingKey(true);
    try {
      const created = await createUserApiKey(keyName);
      setCreatedApiKey(created);
      setKeyName("");
      toast.success("API Key 已创建");
      await loadApiKeys();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "API Key 创建失败");
    } finally {
      setIsCreatingKey(false);
    }
  }

  async function handleDeleteKey(key: UserApiKey) {
    if (deletingId) {
      return;
    }

    const confirmed = window.confirm(`删除 API Key「${key.name}」？`);
    if (!confirmed) {
      return;
    }

    setDeletingId(key.id);
    try {
      await deleteUserApiKey(key.id);
      setApiKeys((items) => items.filter((item) => item.id !== key.id));
      toast.success("API Key 已删除");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "API Key 删除失败");
    } finally {
      setDeletingId(null);
    }
  }

  async function copyValue(value: string, label: string) {
    try {
      await navigator.clipboard.writeText(value);
      setCopiedValue(value);
      toast.success(`${label}已复制`);
      window.setTimeout(() => {
        setCopiedValue((current) => (current === value ? null : current));
      }, 1600);
    } catch {
      toast.error("复制失败");
    }
  }

  return (
    <main className="theme-adaptive min-h-dvh bg-[#121212] text-white">
      <div className="mx-auto flex min-h-dvh w-full max-w-[1180px] flex-col">
        <header className="sticky top-0 z-20 flex h-14 items-center gap-2 border-b border-white/[0.07] bg-[#121212]/95 px-3 backdrop-blur">
          <Button
            asChild
            variant="ghost"
            size="icon"
            className="size-10 text-white/78 hover:bg-white/[0.06] hover:text-white"
          >
            <Link href="/admin">
              <ArrowLeft className="size-5" />
            </Link>
          </Button>
          <div className="min-w-0 flex-1">
            <h1 className="truncate text-lg font-bold">后端 API</h1>
          </div>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            disabled={isLoadingSpec}
            aria-label="刷新后端 API"
            onClick={() => void loadSpec()}
            className="size-10 text-white/70 hover:bg-white/[0.06] hover:text-white"
          >
            <RefreshCw className={cn("size-5", isLoadingSpec && "animate-spin")} />
          </Button>
        </header>

        <section className="min-h-0 flex-1 overflow-y-auto px-4 pb-[calc(28px+env(safe-area-inset-bottom))] pt-4">
          <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
            <section className="min-w-0 space-y-4">
              <div className="rounded-[8px] border border-white/[0.08] bg-white/[0.06] p-4">
                <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
                  <div className="min-w-0">
                    <div className="flex items-center gap-3">
                      <span className="flex size-11 shrink-0 items-center justify-center rounded-full bg-sky-300/12 text-sky-200">
                        <Code2 className="size-5" />
                      </span>
                      <div className="min-w-0">
                        <h2 className="truncate text-xl font-black">
                          {spec?.info?.title ?? "Yuem Go-Gin API"}
                        </h2>
                        <p className="mt-1 text-sm text-white/46">
                          {operations.length} 个接口 · {spec?.openapi ?? "OpenAPI"}
                        </p>
                      </div>
                    </div>
                  </div>

                  <div className="flex shrink-0 flex-wrap gap-2">
                    <Button
                      asChild
                      variant="outline"
                      className="h-9 rounded-[8px] border-white/[0.12] bg-transparent px-3 text-white hover:bg-white/[0.07]"
                    >
                      <a href={docsUrl} target="_blank" rel="noreferrer">
                        <ExternalLink className="size-4" />
                        Swagger
                      </a>
                    </Button>
                    <Button
                      asChild
                      variant="outline"
                      className="h-9 rounded-[8px] border-white/[0.12] bg-transparent px-3 text-white hover:bg-white/[0.07]"
                    >
                      <a href={jsonUrl} target="_blank" rel="noreferrer">
                        <FileJson className="size-4" />
                        JSON
                      </a>
                    </Button>
                  </div>
                </div>

                <div className="mt-4 grid gap-3 md:grid-cols-[minmax(0,1fr)_220px]">
                  <label className="relative block min-w-0">
                    <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-white/36" />
                    <input
                      value={query}
                      onChange={(event) => setQuery(event.target.value)}
                      placeholder="搜索路径、摘要或标签"
                      className="h-11 w-full rounded-[8px] border border-white/[0.08] bg-black/24 pl-10 pr-3 text-sm text-white outline-none placeholder:text-white/30 focus:border-sky-300/55 focus:ring-2 focus:ring-sky-300/15"
                    />
                  </label>
                  <select
                    value={methodFilter}
                    onChange={(event) => setMethodFilter(event.target.value as MethodFilter)}
                    className="h-11 rounded-[8px] border border-white/[0.08] bg-black/24 px-3 text-sm font-semibold text-white outline-none focus:border-sky-300/55 focus:ring-2 focus:ring-sky-300/15"
                  >
                    <option value="all">全部方法 ({operations.length})</option>
                    {httpMethods.map((method) => (
                      <option key={method} value={method}>
                        {methodLabels[method]} ({methodCounts.get(method) ?? 0})
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div className="grid gap-2">
                {isLoadingSpec ? (
                  <div className="flex min-h-[260px] items-center justify-center rounded-[8px] border border-white/[0.08] bg-white/[0.06] text-white/56">
                    <Loader2 className="mr-2 size-5 animate-spin" />
                    正在加载后端 API
                  </div>
                ) : filteredOperations.length > 0 ? (
                  filteredOperations.map((entry) => (
                    <OperationCard
                      key={`${entry.method}-${entry.path}-${entry.operation.operationId ?? entry.operation.summary ?? ""}`}
                      entry={entry}
                      copiedValue={copiedValue}
                      onCopy={copyValue}
                    />
                  ))
                ) : (
                  <div className="rounded-[8px] border border-dashed border-white/[0.12] bg-white/[0.04] px-4 py-12 text-center text-sm text-white/45">
                    没有匹配的接口
                  </div>
                )}
              </div>
            </section>

            <aside className="min-w-0 space-y-4">
              <section className="rounded-[8px] border border-white/[0.08] bg-white/[0.06] p-4">
                <div className="flex items-center gap-3">
                  <span className="flex size-10 shrink-0 items-center justify-center rounded-full bg-emerald-300/12 text-emerald-200">
                    <KeyRound className="size-5" />
                  </span>
                  <div className="min-w-0 flex-1">
                    <h2 className="truncate text-base font-bold">个人 API Key</h2>
                    <p className="mt-1 text-xs text-white/42">用于换取当前账号访问令牌。</p>
                  </div>
                </div>

                {!authChecked ? (
                  <div className="mt-4 flex h-28 items-center justify-center rounded-[8px] bg-black/22 text-sm text-white/50">
                    <Loader2 className="mr-2 size-4 animate-spin" />
                    正在检查登录状态
                  </div>
                ) : !authToken ? (
                  <div className="mt-4 rounded-[8px] bg-black/22 px-4 py-6 text-center">
                    <Shield className="mx-auto size-7 text-white/50" />
                    <p className="mt-3 text-sm font-bold">登录后管理密钥</p>
                    <Button asChild className="mt-4 h-9 rounded-full bg-primary px-4 text-white">
                      <Link href="/login">去登录</Link>
                    </Button>
                  </div>
                ) : (
                  <>
                    <form onSubmit={handleCreateKey} className="mt-4 flex gap-2">
                      <input
                        value={keyName}
                        onChange={(event) => setKeyName(event.target.value)}
                        maxLength={50}
                        placeholder="密钥名称"
                        className="min-w-0 flex-1 rounded-[8px] border border-white/[0.08] bg-black/24 px-3 text-sm text-white outline-none placeholder:text-white/30 focus:border-emerald-300/55 focus:ring-2 focus:ring-emerald-300/15"
                      />
                      <Button
                        type="submit"
                        disabled={!keyName.trim() || isCreatingKey}
                        size="icon"
                        aria-label="创建 API Key"
                        className="size-10 rounded-[8px] bg-emerald-500 text-white hover:bg-emerald-500/90"
                      >
                        {isCreatingKey ? <Loader2 className="size-4 animate-spin" /> : <Plus className="size-4" />}
                      </Button>
                    </form>

                    {createdApiKey?.api_key ? (
                      <div className="mt-4 rounded-[8px] border border-emerald-300/20 bg-emerald-300/10 p-3">
                        <div className="flex items-center justify-between gap-3">
                          <p className="text-xs font-bold text-emerald-100">新密钥仅显示一次</p>
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            aria-label="复制新 API Key"
                            onClick={() => void copyValue(createdApiKey.api_key ?? "", "API Key")}
                            className="size-8 text-emerald-100 hover:bg-emerald-300/10"
                          >
                            {copiedValue === createdApiKey.api_key ? (
                              <Check className="size-4" />
                            ) : (
                              <Copy className="size-4" />
                            )}
                          </Button>
                        </div>
                        <code className="mt-2 block break-all rounded-[6px] bg-black/32 px-3 py-2 text-xs text-emerald-50">
                          {createdApiKey.api_key}
                        </code>
                      </div>
                    ) : null}

                    <div className="mt-4 space-y-2">
                      {isLoadingKeys ? (
                        <div className="flex h-24 items-center justify-center text-sm text-white/50">
                          <Loader2 className="mr-2 size-4 animate-spin" />
                          正在加载
                        </div>
                      ) : apiKeys.length > 0 ? (
                        apiKeys.map((key) => (
                          <ApiKeyRow
                            key={key.id}
                            apiKey={key}
                            deleting={deletingId === key.id}
                            onDelete={handleDeleteKey}
                          />
                        ))
                      ) : (
                        <div className="rounded-[8px] border border-dashed border-white/[0.12] px-3 py-6 text-center text-sm text-white/40">
                          暂无 API Key
                        </div>
                      )}
                    </div>
                  </>
                )}
              </section>

              <section className="rounded-[8px] border border-white/[0.08] bg-white/[0.06] p-4">
                <div className="flex items-center gap-2 text-sm font-bold text-white/80">
                  <Shield className="size-4 text-amber-200" />
                  调用入口
                </div>
                <div className="mt-3 space-y-2 text-xs leading-5 text-white/50">
                  <p>个人 API Key 先请求 <code className="rounded bg-black/24 px-1 text-white/78">POST /api/auth/token</code> 换取令牌。</p>
                  <p><code className="rounded bg-black/24 px-1 text-white/78">/api/open/*</code> 使用后台 OpenAPI 密钥。</p>
                </div>
              </section>
            </aside>
          </div>
        </section>
      </div>
    </main>
  );
}

function OperationCard({
  copiedValue,
  entry,
  onCopy,
}: {
  copiedValue: string | null;
  entry: OperationEntry;
  onCopy: (value: string, label: string) => Promise<void>;
}) {
  const tags = entry.operation.tags ?? [];
  const value = `${methodLabels[entry.method]} ${entry.path}`;

  return (
    <article className="rounded-[8px] border border-white/[0.08] bg-white/[0.06] px-3 py-3 sm:px-4">
      <div className="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <span className={cn("rounded-[6px] px-2 py-1 text-xs font-black", methodClassName(entry.method))}>
              {methodLabels[entry.method]}
            </span>
            <code className="min-w-0 break-all text-sm font-bold text-white/88">{entry.path}</code>
          </div>
          <p className="mt-2 line-clamp-2 text-sm leading-5 text-white/56">
            {entry.operation.summary || entry.operation.description || entry.operation.operationId || "未命名接口"}
          </p>
          {tags.length > 0 ? (
            <div className="mt-3 flex flex-wrap gap-1.5">
              {tags.slice(0, 4).map((tag) => (
                <span key={tag} className="rounded-full bg-white/[0.07] px-2 py-0.5 text-xs font-semibold text-white/45">
                  {tag}
                </span>
              ))}
            </div>
          ) : null}
        </div>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label="复制接口路径"
          onClick={() => void onCopy(value, "接口")}
          className="size-9 self-end text-white/56 hover:bg-white/[0.07] hover:text-white sm:self-start"
        >
          {copiedValue === value ? <Check className="size-4" /> : <Copy className="size-4" />}
        </Button>
      </div>
    </article>
  );
}

function ApiKeyRow({
  apiKey,
  deleting,
  onDelete,
}: {
  apiKey: UserApiKey;
  deleting: boolean;
  onDelete: (apiKey: UserApiKey) => void;
}) {
  return (
    <div className="flex min-w-0 items-center gap-3 rounded-[8px] bg-black/20 px-3 py-3">
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-bold text-white">{apiKey.name}</p>
        <p className="mt-1 truncate text-xs text-white/38">
          {apiKey.api_key_prefix ?? "无前缀"} · {formatDateTime(apiKey.created_at)}
        </p>
      </div>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        disabled={deleting}
        aria-label="删除 API Key"
        onClick={() => onDelete(apiKey)}
        className="size-9 text-rose-200/80 hover:bg-rose-300/10 hover:text-rose-100"
      >
        {deleting ? <Loader2 className="size-4 animate-spin" /> : <Trash2 className="size-4" />}
      </Button>
    </div>
  );
}

function flattenOperations(spec: BackendApiSpec | null): OperationEntry[] {
  const paths = spec?.paths ?? {};
  const entries: OperationEntry[] = [];

  for (const [path, operations] of Object.entries(paths)) {
    for (const [method, operation] of Object.entries(operations)) {
      if (!isHttpMethod(method) || !isRecord(operation)) {
        continue;
      }

      entries.push({
        method,
        operation: operation as BackendApiOperation,
        path,
      });
    }
  }

  return entries.sort((a, b) => `${a.path} ${a.method}`.localeCompare(`${b.path} ${b.method}`));
}

function filterOperations(entries: OperationEntry[], method: MethodFilter, query: string) {
  const term = query.trim().toLowerCase();
  return entries.filter((entry) => {
    if (method !== "all" && entry.method !== method) {
      return false;
    }

    if (!term) {
      return true;
    }

    const haystack = [
      entry.path,
      entry.method,
      entry.operation.summary,
      entry.operation.description,
      entry.operation.operationId,
      ...(entry.operation.tags ?? []),
    ]
      .filter(Boolean)
      .join(" ")
      .toLowerCase();

    return haystack.includes(term);
  });
}

function isHttpMethod(value: string): value is HttpMethod {
  return httpMethods.includes(value.toLowerCase() as HttpMethod);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function methodClassName(method: HttpMethod) {
  switch (method) {
    case "get":
      return "bg-sky-300/14 text-sky-100";
    case "post":
      return "bg-emerald-300/14 text-emerald-100";
    case "put":
      return "bg-amber-300/14 text-amber-100";
    case "patch":
      return "bg-violet-300/14 text-violet-100";
    case "delete":
      return "bg-rose-300/14 text-rose-100";
    default:
      return "bg-white/[0.08] text-white/64";
  }
}

function formatDateTime(value?: string | null) {
  if (!value) {
    return "未使用";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat("zh-CN", {
    dateStyle: "short",
    timeStyle: "short",
  }).format(date);
}
