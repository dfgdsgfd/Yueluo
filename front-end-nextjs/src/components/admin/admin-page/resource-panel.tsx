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
  Database,
  Edit3,
  Eye,
  Coins,
  Loader2,
  Plus,
  Search,
  Trash2
} from "lucide-react";
import {
  toast
} from "sonner";
import {
  Button
} from "@/components/ui/button";
import {
  adminRequest,
  bulkDeleteAdminResource,
  createAdminResource,
  deleteAdminResource,
  getAdminList,
  updateAdminResource
} from "@/lib/api";
import type {
  AdminListPayload,
  AdminListResource,
  AdminListRow
} from "@/lib/types";
import {
  cn
} from "@/lib/utils";
import {
  ResourceConfig,
  defaultLimit
} from "./types";
import { resourceConfigMap, resourceConfigs } from "./resource-configs";
import {
  EditorDrawer,
  EmptyBlock,
  FilterControl,
  LoadingBlock,
  PaginationBar,
  uploadAutoFillDraft
} from "./resource-editor";
import {
  IconButton,
  ResourceRowActions
} from "./resource-cells";
import { FileRecycleRowActions } from "./file-recycle-preview";
import { UserPointsDialog } from "./user-points-dialog";
import { SystemNotificationTemplatePicker } from "./system-notification-templates";
import { UserBatchGenerate } from "./user-batch-generate";
import {
  appVersionLastFormDraft,
  defaultDraft,
  drawerTitle,
  errorMessage,
  fieldText,
  formatCompact,
  sanitizeDraft,
  toneSoftClass
} from "./helpers";
import { useTranslations } from "next-intl";

export function ResourcePanel({
  resource,
  token,
}: {
  resource: AdminListResource;
  token: string;
}) {
  const userT = useTranslations("adminPortal.userManagement");
  const adminT = useTranslations("adminPortal");
  const config: ResourceConfig = resourceConfigMap.get(resource) ?? resourceConfigs[0];
  const [payload, setPayload] = useState<AdminListPayload | null>(null);
  const [page, setPage] = useState(1);
  const [keywordDraft, setKeywordDraft] = useState("");
  const [keyword, setKeyword] = useState("");
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [selectedIds, setSelectedIds] = useState<Array<string | number>>([]);
  const [loading, setLoading] = useState(true);
  const [panelMode, setPanelMode] = useState<"create" | "edit" | "detail" | null>(null);
  const [activeRow, setActiveRow] = useState<AdminListRow | null>(null);
  const [draft, setDraft] = useState<Record<string, unknown>>({});
  const [saving, setSaving] = useState(false);
  const [restoringLastApp, setRestoringLastApp] = useState(false);
  const [pointsRow, setPointsRow] = useState<AdminListRow | null>(null);

  const limit = config.limit ?? defaultLimit;
  const load = useCallback(
    async (next: { page?: number; keyword?: string; filters?: Record<string, string> } = {}) => {
      setLoading(true);
      try {
        const nextPage = next.page ?? page;
        const nextKeyword = next.keyword ?? keyword;
        const nextFilters = next.filters ?? filters;
        const data = await getAdminList(config.resource, {
          page: nextPage,
          limit,
          keyword: nextKeyword,
          sortField: config.defaultSort,
          sortOrder: "DESC",
          filters: nextFilters,
          basePath: config.basePath,
        }, token);
        setPayload(data);
        setSelectedIds([]);
      } catch (error) {
        toast.error(errorMessage(error));
        setPayload(null);
      } finally {
        setLoading(false);
      }
    },
    [config.basePath, config.defaultSort, config.resource, filters, keyword, limit, page, token],
  );

  useEffect(() => {
    queueMicrotask(() => {
      void load();
    });
  }, [load]);

  function openCreate() {
    setActiveRow(null);
    setDraft(defaultDraft(config.fields, null, "create"));
    setPanelMode("create");
  }

  async function restoreLastAppVersionDraft() {
    if (config.resource !== "app-versions") return;
    setRestoringLastApp(true);
    try {
      const lastForm = await adminRequest<Record<string, unknown>>("/api/admin/app-versions/last-form-data", { method: "GET", token });
      const restored = appVersionLastFormDraft(lastForm);
      if (!Object.keys(restored).length) {
        toast.info("暂无可恢复的版本信息");
        return;
      }
      setDraft((current) => ({ ...current, ...restored }));
      toast.success("已恢复上次填写的版本信息");
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setRestoringLastApp(false);
    }
  }

  function openEdit(row: AdminListRow) {
    setActiveRow(row);
    setDraft(defaultDraft(config.fields, row, "edit"));
    setPanelMode("edit");
  }

  function openDetail(row: AdminListRow) {
    setActiveRow(row);
    setDraft(defaultDraft(config.fields, row, "detail"));
    setPanelMode("detail");
  }

  async function handleSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const nextKeyword = keywordDraft.trim();
    setKeyword(nextKeyword);
    setPage(1);
    await load({ page: 1, keyword: nextKeyword });
  }

  async function updateFilter(key: string, value: string) {
    const nextFilters = { ...filters, [key]: value };
    if (!value) {
      delete nextFilters[key];
    }
    setFilters(nextFilters);
    setPage(1);
    await load({ page: 1, filters: nextFilters });
  }

  async function goPage(nextPage: number) {
    if (nextPage < 1) return;
    setPage(nextPage);
    await load({ page: nextPage });
  }

  async function submitForm(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    try {
      const body = sanitizeDraft(config.fields, draft, panelMode);
      if (panelMode === "create") {
        await createAdminResource(config.resource, body, token, config.basePath);
        if (config.resource === "app-versions") {
          await adminRequest("/api/admin/app-versions/last-form-data", {
            method: "POST",
            token,
            body: JSON.stringify(body),
          });
        }
        toast.success(`${config.singular}已创建`);
      } else if (panelMode === "edit" && activeRow?.id !== undefined) {
        if (config.resource === "posts-quality") {
          await adminRequest(`/api/admin/posts/${encodeURIComponent(String(activeRow.id))}/quality`, { method: "PUT", token, body: JSON.stringify(body) });
        } else {
          await updateAdminResource(config.resource, activeRow.id, body, token, config.basePath);
        }
        toast.success(`${config.singular}已更新`);
      }
      setPanelMode(null);
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(row: AdminListRow) {
    const recycleBin = config.resource === "file-recycle-bin";
    if (row.id === undefined || !window.confirm(recycleBin ? `确认提前清理该${config.singular}？` : `确认删除该${config.singular}？`)) return;
    try {
      await deleteAdminResource(config.resource, row.id, token, config.basePath);
      toast.success(recycleBin ? "已提前清理" : "已删除");
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    }
  }

  async function handleBulkDelete() {
    const recycleBin = config.resource === "file-recycle-bin";
    if (!selectedIds.length || !window.confirm(recycleBin ? `确认提前清理已选 ${selectedIds.length} 条记录？` : `确认删除已选 ${selectedIds.length} 条记录？`)) return;
    try {
      await bulkDeleteAdminResource(config.resource, selectedIds, token, config.basePath);
      toast.success(recycleBin ? "已批量提前清理" : "已批量删除");
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    }
  }

  async function runResourceAction(row: AdminListRow, action: "approve" | "reject" | "retry" | "resend" | "toggle-active" | "recommend" | "original-incentive" | "test-discord" | "test-email") {
    if (row.id === undefined) return;
    try {
      if (action === "approve" || action === "reject" || action === "retry") {
        const suffix = action === "approve" ? "approve" : action === "reject" ? "reject" : "retry";
        await adminRequest(`/api/admin/${config.resource}/${encodeURIComponent(String(row.id))}/${suffix}`, { method: "PUT", token, body: JSON.stringify({ reason: action === "reject" ? "管理员驳回" : "" }) });
        toast.success(action === "approve" ? "已通过" : action === "reject" ? "已驳回" : "已提交重试");
      } else if (action === "resend") {
        await adminRequest(`/api/admin/system-notifications/${encodeURIComponent(String(row.id))}/resend`, { method: "POST", token });
        toast.success("系统通知已重新发送");
      } else if (action === "toggle-active") {
        await adminRequest(`/api/admin/user-toolbar/${encodeURIComponent(String(row.id))}/toggle-active`, { method: "PUT", token });
        toast.success("状态已切换");
      } else if (action === "recommend") {
        const postID = row.post_id ?? row.id;
        await adminRequest("/api/admin/recommendation/push", { method: "POST", token, body: JSON.stringify({ post_id: postID, boost_score: 10, reason: "后台快捷推荐" }) });
        toast.success("已加入推荐");
      } else if (action === "original-incentive") {
        const postID = row.post_id ?? row.id;
        const currentAmount = Number(row.quality_reward ?? row.reward_amount ?? 0);
        const amountText = window.prompt("输入原创激励金额（余额）", Number.isFinite(currentAmount) && currentAmount > 0 ? String(currentAmount) : "");
        if (!amountText) return;
        const rewardAmount = Number(amountText);
        if (!Number.isFinite(rewardAmount) || rewardAmount <= 0) {
          toast.error("请输入有效原创激励金额");
          return;
        }
        await adminRequest(`/api/admin/posts/${encodeURIComponent(String(postID))}/quality`, { method: "PUT", token, body: JSON.stringify({ quality_reward: rewardAmount }) });
        toast.success("原创激励已更新");
      } else if (action === "test-discord") {
        await adminRequest(`/api/admin/notification-templates/${encodeURIComponent(String(row.id))}/test-discord`, { method: "POST", token });
        toast.success("Discord 测试已发送");
      } else if (action === "test-email") {
        const email = window.prompt("输入测试邮箱");
        if (!email) return;
        await adminRequest(`/api/admin/notification-templates/${encodeURIComponent(String(row.id))}/test-email`, { method: "POST", token, body: JSON.stringify({ email }) });
        toast.success("测试邮件已发送");
      }
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    }
  }

  const rows = payload?.items ?? [];
  const total = payload?.pagination.total ?? 0;
  const canCreate = config.canCreate ?? !config.readOnly;
  const canEdit = config.canEdit ?? !config.readOnly;
  const canDelete = config.canDelete ?? !config.readOnly;
  const canBulkDelete = config.canBulkDelete ?? canDelete;
  const deleteActionLabel = config.resource === "file-recycle-bin" ? "提前清理" : "删除";
  const drawerReadOnly = panelMode === "detail" || (panelMode === "create" && !canCreate) || (panelMode === "edit" && !canEdit);
  const actionColumnClass = config.resource === "file-recycle-bin" ? "w-[224px]" : "w-[168px]";
  const columnHeader = (column: ResourceConfig["columns"][number]) => {
    if (!column.labelKey) return column.label;
    if (column.labelKey.includes(".") && adminT.has(column.labelKey)) return adminT(column.labelKey);
    return userT(`fields.${column.labelKey}`);
  };

  return (
    <div className="grid min-w-0 gap-4">
      <section className="min-w-0 rounded-lg border border-black/[0.06] bg-white p-4 shadow-[0_12px_34px_rgba(20,20,35,0.05)] sm:p-5">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
          <div className="min-w-0">
            <div className="mb-2 flex items-center gap-2">
              <span className={cn("flex size-9 items-center justify-center rounded-lg", toneSoftClass(config.tone))}>
                <config.icon className="size-4" />
              </span>
              <div className="min-w-0">
                <h1 className="truncate text-xl font-semibold text-[#17171d]">{config.label}</h1>
                <p className="truncate text-sm text-[#777d89]">{config.description}</p>
              </div>
            </div>
            <p className="text-xs text-[#9096a3]">共 {formatCompact(total)} 条 · 当前第 {payload?.pagination.page ?? page} 页</p>
          </div>
          <div className="flex flex-wrap gap-2">
            {config.resource === "users" ? (
              <UserBatchGenerate token={token} t={userT} onGenerated={() => load({ page: 1 })} />
            ) : null}
            {canBulkDelete && selectedIds.length ? (
              <Button
                type="button"
                variant="outline"
                onClick={() => void handleBulkDelete()}
                className="h-10 rounded-lg border-[#dc2626]/20 bg-[#fef2f2] text-[#b91c1c] hover:bg-[#fee2e2]"
              >
                <Trash2 className="size-4" />
                {config.resource === "file-recycle-bin" ? "提前清理已选" : "删除已选"}
              </Button>
            ) : null}
            {canCreate ? (
              <Button type="button" onClick={openCreate} className="h-10 rounded-lg bg-[#1d4ed8] px-4 hover:bg-[#1e40af]">
                <Plus className="size-4" />
                新建{config.singular}
              </Button>
            ) : null}
          </div>
        </div>
      </section>

      <section className="min-w-0 rounded-lg border border-black/[0.06] bg-white shadow-[0_12px_34px_rgba(20,20,35,0.05)]">
        <div className="grid gap-3 border-b border-black/[0.06] p-3 sm:p-4 xl:grid-cols-[minmax(280px,1fr)_auto]">
          <form onSubmit={handleSearch} className="flex min-w-0 gap-2">
            <label className="relative min-w-0 flex-1">
              <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[#9aa0ad]" />
              <input
                value={keywordDraft}
                onChange={(event) => setKeywordDraft(event.target.value)}
                className="h-10 w-full rounded-lg border border-black/[0.08] bg-[#fafbfe] pl-9 pr-3 text-sm outline-none transition focus:border-[#1d4ed8] focus:bg-white focus:ring-4 focus:ring-[#1d4ed8]/10"
                placeholder={`搜索${config.singular}`}
              />
            </label>
            <Button type="submit" disabled={loading} className="h-10 rounded-lg bg-[#17171d] px-4 hover:bg-[#2a2b32]">
              <span className="inline-flex size-4 items-center justify-center" aria-hidden="true">
                {loading ? <Loader2 className="size-4 animate-spin" /> : <Search className="size-4" />}
              </span>
              <span>查询</span>
            </Button>
          </form>
          <div className="flex min-w-0 flex-wrap gap-2">
            {(config.filters ?? []).map((filter) => (
              <FilterControl
                key={filter.key}
                filter={filter}
                value={filters[filter.key] ?? ""}
                onChange={(value) => void updateFilter(filter.key, value)}
              />
            ))}
          </div>
        </div>

        <div className="min-h-[440px] min-w-0">
          {loading ? (
            <LoadingBlock label={`正在加载${config.label}`} />
          ) : rows.length ? (
            <div className="w-full min-w-0 max-w-full overflow-x-auto">
              <table className="w-full table-fixed text-left text-sm" style={{ minWidth: config.tableMinWidth ?? 900 }}>
                <thead className="border-b border-black/[0.06] bg-[#fafbfe] text-xs font-semibold text-[#7a808c]">
                  <tr>
                    <th className="w-12 px-4 py-3">
                      <input
                        type="checkbox"
                        checked={rows.length > 0 && selectedIds.length === rows.filter((row) => row.id !== undefined).length}
                        onChange={(event) => {
                          setSelectedIds(event.target.checked ? rows.flatMap((row) => (row.id === undefined ? [] : [row.id])) : []);
                        }}
                        aria-label="全选"
                      />
                    </th>
                    {config.columns.map((column) => (
                      <th key={column.key} className={cn("px-4 py-3", column.className)}>
                        {columnHeader(column)}
                      </th>
                    ))}
                    <th className={cn(actionColumnClass, "px-4 py-3 text-right")}>操作</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-black/[0.05]">
                  {rows.map((row, rowIndex) => {
                    const rowID = row.id ?? rowIndex;
                    const selectableID = row.id;
                    return (
                      <tr key={String(rowID)} className="text-[#3f444e] transition-colors hover:bg-[#fafbfe]">
                        <td className="px-4 py-3 align-top">
                          <input
                            type="checkbox"
                            checked={selectableID !== undefined && selectedIds.includes(selectableID)}
                            onChange={(event) => {
                              if (selectableID === undefined) return;
                              setSelectedIds((current) =>
                                event.target.checked ? [...current, selectableID] : current.filter((id) => id !== selectableID),
                              );
                            }}
                            aria-label="选择行"
                          />
                        </td>
                        {config.columns.map((column) => (
                          <td key={column.key} className={cn("px-4 py-3 align-top", column.className)}>
                            <div className="min-w-0">{column.render ? column.render(row) : fieldText(row, column.key)}</div>
                          </td>
                        ))}
                        <td className="px-4 py-3 align-top">
                          <div className="flex justify-end gap-1">
                            <ResourceRowActions
                              resource={config.resource}
                              row={row}
                              onAction={(action) => void runResourceAction(row, action)}
                            />
                            {config.resource === "file-recycle-bin" ? (
                              <FileRecycleRowActions row={row} token={token} />
                            ) : null}
                            {config.resource === "users" ? (
                              <IconButton label={userT("actions.managePoints")} icon={Coins} onClick={() => setPointsRow(row)} />
                            ) : null}
                            <IconButton label="查看" icon={Eye} onClick={() => openDetail(row)} />
                            {canEdit ? <IconButton label="编辑" icon={Edit3} onClick={() => openEdit(row)} /> : null}
                            {canDelete ? <IconButton label={deleteActionLabel} icon={Trash2} danger onClick={() => void handleDelete(row)} /> : null}
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          ) : (
            <EmptyBlock icon={Database} label={`暂无${config.singular}数据`} />
          )}
        </div>
        <PaginationBar
          page={payload?.pagination.page ?? page}
          total={payload?.pagination.total}
          hasNext={Boolean(payload?.pagination.hasNextPage)}
          disabled={loading}
          onPrev={() => void goPage(page - 1)}
          onNext={() => void goPage(page + 1)}
        />
      </section>

      <EditorDrawer
        mode={panelMode}
        title={drawerTitle(config, panelMode)}
        fields={config.fields}
        row={activeRow}
        draft={draft}
        token={token}
        saving={saving}
        readOnly={drawerReadOnly}
        restoreAction={config.resource === "app-versions" && panelMode === "create" ? {
          label: "恢复上次填写",
          loading: restoringLastApp,
          onClick: () => void restoreLastAppVersionDraft(),
        } : undefined}
        formAddon={config.resource === "system-notifications" && !drawerReadOnly ? (
          <SystemNotificationTemplatePicker
            onApply={(template) => {
              setDraft((current) => ({
                ...current,
                content: template.content,
                is_active: true,
                show_popup: template.showPopup,
                title: template.title,
                type: template.type,
              }));
              toast.success(`已填充「${template.label}」模板`);
            }}
          />
        ) : undefined}
        onDraftChange={(key, value) => setDraft((current) => ({ ...current, [key]: value }))}
        onFieldUpload={(field, assets, files) => {
          setDraft((current) => ({ ...current, ...uploadAutoFillDraft(config.resource, field, assets, files, current) }));
        }}
        onClose={() => setPanelMode(null)}
        onSubmit={(event) => void submitForm(event)}
      />
      <UserPointsDialog
        key={String(pointsRow?.id ?? "closed")}
        row={pointsRow}
        token={token}
        onClose={() => setPointsRow(null)}
        onSaved={async () => {
          setPointsRow(null);
          await load();
        }}
      />
    </div>
  );
}
