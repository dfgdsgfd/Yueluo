import {
  Activity,
  CircleDollarSign,
  ClipboardList,
  Database,
  Eye,
  FileText,
  Folder,
  Image as ImageIcon,
  Lock,
  Package,
  Plus,
  Save,
  Search,
  Settings,
  ShieldCheck,
  SlidersHorizontal,
  Star,
  Trash2,
  UploadCloud,
  UserCog,
  Wallet,
  Zap
} from "lucide-react";
import {
  Button
} from "@/components/ui/button";
import {
  cn
} from "@/lib/utils";
import {
  qualityLevelOptions
} from "./types";
import {
  HeaderCard,
  Panel
} from "./layout-widgets";
import {
  LoadingBlock
} from "./resource-editor";
import {
  AdminObjectPicker
} from "./object-picker";
import {
  ActivityList,
  AppStatsPanel,
  BatchFilesPanel,
  CoverStatsPanel,
  FileList,
  OperationResultPanel,
  ReviewSettingsPanel,
  SupportDataPanel
} from "./operations-widgets";
import {
  columnLabel,
  parseConfigValue,
  recommendConfigEntries
} from "./helpers";
import {
  ChoiceSelect,
  ToggleSwitch
} from "./form-fields";
import type {
  useOperationsController
} from "./use-operations-controller";

export function OperationsPanelView({ controller }: { controller: ReturnType<typeof useOperationsController> }) {
  const {
    activities,
    acting,
    apkFiles,
    appStats,
    bannedWordsDraft,
    batchContent,
    batchFiles,
    batchStatusId,
    batchTitle,
    batchUsers,
    checkBatchStatus,
    coverLimit,
    createBatchPosts,
    createBatchPostsAsync,
    deleteBatchFiles,
    earningsAmount,
    earningsUsers,
    exportBannedWords,
    generateCovers,
    importBannedWords,
    lastAppForm,
    loading,
    missingCovers,
    operationResult,
    operationTab,
    postBatchCategory,
    postBatchPosts,
    postTransferUser,
    previewTemplate,
    pushPosts,
    pushRecommendation,
    pushScore,
    qualityBatchLevel,
    qualityBatchPosts,
    recommendBatchBoost,
    recommendBatchPinned,
    recommendBatchPosts,
    recommendBatchReason,
    recommendBatchSuppressed,
    recommendConfig,
    reviewSettings,
    runEarnings,
    runPostBatch,
    runQualityBatch,
    runRecommendationBatch,
    saveRecommendConfig,
    setBannedWordsDraft,
    setBatchContent,
    setBatchStatusId,
    setBatchTitle,
    setBatchUsers,
    setCoverLimit,
    setEarningsAmount,
    setEarningsUsers,
    setOperationTab,
    setPostBatchCategory,
    setPostBatchPosts,
    setPostTransferUser,
    setPushPosts,
    setPushScore,
    setQualityBatchLevel,
    setQualityBatchPosts,
    setRecommendBatchBoost,
    setRecommendBatchPinned,
    setRecommendBatchPosts,
    setRecommendBatchReason,
    setRecommendBatchSuppressed,
    setRecommendConfig,
    setTemplateContent,
    setTemplateSubject,
    templateContent,
    templateDefaults,
    templateSubject,
    testUsers,
    token,
    updateReviewSettings,
  } = controller;

  return (
    <div className="grid gap-4">
      <HeaderCard icon={Activity} title="运营控制台" description="系统维护、内容运营、推荐增长、应用素材与创作收益" tone="blue" />
      <div className="flex flex-wrap gap-2 rounded-lg border border-black/[0.06] bg-white p-2 shadow-[0_12px_34px_rgba(20,20,35,0.05)]">
        {([
          ["system", "系统维护", Settings],
          ["content", "内容运营", FileText],
          ["growth", "推荐增长", Zap],
          ["assets", "应用素材", Package],
          ["finance", "创作收益", Wallet],
        ] as const).map(([key, label, Icon]) => (
          <button
            key={key}
            type="button"
            onClick={() => setOperationTab(key)}
            className={cn(
              "inline-flex h-10 items-center gap-2 rounded-lg px-3 text-sm font-semibold transition",
              operationTab === key ? "bg-[#1d4ed8] text-white shadow-sm shadow-[#1d4ed8]/20" : "text-[#59606c] hover:bg-[#f6f7fb]",
            )}
          >
            <Icon className="size-4" />
            {label}
          </button>
        ))}
      </div>
      {loading ? (
        <LoadingBlock label="正在加载运营控制台" />
      ) : (
        <section className="grid gap-4">
          {operationTab === "system" ? (
            <>
              <div className="grid gap-4 xl:grid-cols-[minmax(0,1.1fr)_minmax(340px,0.9fr)]">
                <Panel title="审核自动化" icon={ShieldCheck}>
                  <ReviewSettingsPanel settings={reviewSettings} loading={acting === "review"} onChange={(next) => void updateReviewSettings(next)} />
                </Panel>
                <Panel title="后端辅助数据" icon={Database}>
                  <SupportDataPanel templateDefaults={templateDefaults} testUsers={testUsers} lastAppForm={lastAppForm} />
                </Panel>
              </div>
              <Panel title="敏感词导入导出" icon={ShieldCheck}>
                <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_220px]">
                  <textarea
                    value={bannedWordsDraft}
                    onChange={(event) => setBannedWordsDraft(event.target.value)}
                    className="min-h-[140px] rounded-lg border border-black/[0.08] bg-[#fafbfe] p-3 text-sm outline-none focus:border-[#1d4ed8]"
                    placeholder="每行一个敏感词，也支持逗号分隔"
                  />
                  <div className="grid content-start gap-2">
                    <Button type="button" disabled={acting === "banned-import"} onClick={() => void importBannedWords()} className="h-10 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]"><UploadCloud className="size-4" />导入</Button>
                    <Button type="button" variant="outline" disabled={acting === "banned-export"} onClick={() => void exportBannedWords()} className="h-10 rounded-lg border-black/[0.08] bg-white px-3 hover:bg-[#f6f7fb]"><ClipboardList className="size-4" />读取导出</Button>
                  </div>
                </div>
              </Panel>
              <Panel title="通知模板预览" icon={ClipboardList}>
                <form onSubmit={previewTemplate} className="grid gap-3">
                  <input value={templateSubject} onChange={(event) => setTemplateSubject(event.target.value)} className="h-10 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]" placeholder="主题" />
                  <textarea value={templateContent} onChange={(event) => setTemplateContent(event.target.value)} className="min-h-[110px] rounded-lg border border-black/[0.08] bg-[#fafbfe] p-3 text-sm outline-none focus:border-[#1d4ed8]" placeholder="模板内容" />
                  <div className="flex justify-end">
                    <Button type="submit" disabled={acting === "template-preview"} className="h-10 rounded-lg bg-[#17171d] px-3 hover:bg-[#2a2b32]"><Eye className="size-4" />生成预览</Button>
                  </div>
                </form>
              </Panel>
            </>
          ) : null}

          {operationTab === "content" ? (
            <>
              <div className="grid gap-4 xl:grid-cols-[minmax(0,0.95fr)_minmax(340px,0.8fr)]">
                <Panel title="实时活动" icon={Activity}>
                  <ActivityList items={activities} />
                </Panel>
                <Panel title="视频封面维护" icon={ImageIcon} action={<Button type="button" disabled={acting === "covers"} onClick={() => void generateCovers()} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]"><UploadCloud className="size-4" />生成</Button>}>
                  <CoverStatsPanel stats={missingCovers} limit={coverLimit} onLimitChange={setCoverLimit} />
                </Panel>
              </div>
              <Panel title="内容批量操作" icon={FileText}>
                <div className="grid gap-3">
                  <AdminObjectPicker
                    token={token}
                    resource="posts"
                    label="待操作内容"
                    multiple
                    value={postBatchPosts}
                    onChange={setPostBatchPosts}
                    placeholder="搜索标题、正文或作者"
                    emptyLabel="未找到内容"
                  />
                  <div className="grid gap-3 lg:grid-cols-2">
                    <AdminObjectPicker
                      token={token}
                      resource="categories"
                      label="目标分类"
                      value={postBatchCategory}
                      onChange={setPostBatchCategory}
                      placeholder="搜索分类"
                      emptyLabel="未找到分类"
                    />
                    <AdminObjectPicker
                      token={token}
                      resource="users"
                      label="目标作者"
                      value={postTransferUser}
                      onChange={setPostTransferUser}
                      placeholder="搜索昵称、账号或邮箱"
                      emptyLabel="未找到用户"
                    />
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <Button type="button" variant="outline" disabled={acting === "post-private"} onClick={() => void runPostBatch("private")} className="rounded-lg border-black/[0.08] bg-white"><Lock className="size-4" />设为私密</Button>
                    <Button type="button" variant="outline" disabled={acting === "post-public"} onClick={() => void runPostBatch("public")} className="rounded-lg border-black/[0.08] bg-white"><Eye className="size-4" />设为公开</Button>
                    <Button type="button" variant="outline" disabled={acting === "post-category"} onClick={() => void runPostBatch("category")} className="rounded-lg border-black/[0.08] bg-white"><Folder className="size-4" />设置分类</Button>
                    <Button type="button" disabled={acting === "post-transfer"} onClick={() => void runPostBatch("transfer")} className="rounded-lg bg-[#17171d] hover:bg-[#2a2b32]"><UserCog className="size-4" />转移作者</Button>
                  </div>
                </div>
              </Panel>
              <Panel title="内容质量批量标记" icon={Star}>
                <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_260px]">
                  <AdminObjectPicker
                    token={token}
                    resource="posts"
                    label="待标记内容"
                    multiple
                    value={qualityBatchPosts}
                    onChange={setQualityBatchPosts}
                    placeholder="搜索标题、正文或作者"
                    emptyLabel="未找到内容"
                  />
                  <div className="grid content-start gap-2">
                    <ChoiceSelect value={qualityBatchLevel} onChange={setQualityBatchLevel} options={qualityLevelOptions} />
                    <Button type="button" disabled={acting === "quality-batch"} onClick={() => void runQualityBatch()} className="h-10 rounded-lg bg-[#17171d] hover:bg-[#2a2b32]"><Star className="size-4" />批量标记</Button>
                  </div>
                </div>
              </Panel>
              <Panel title="批量素材发布" icon={Folder}>
                <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_260px]">
                  <div className="grid gap-3">
                    <AdminObjectPicker
                      token={token}
                      resource="users"
                      label="发布用户"
                      value={batchUsers}
                      onChange={setBatchUsers}
                      placeholder="搜索昵称、账号或邮箱"
                      emptyLabel="未找到用户"
                    />
                    <div className="grid gap-2 sm:grid-cols-2">
                    <input value={batchTitle} onChange={(event) => setBatchTitle(event.target.value)} className="h-10 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]" placeholder="标题" />
                    <input value={batchContent} onChange={(event) => setBatchContent(event.target.value)} className="h-10 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]" placeholder="正文" />
                    </div>
                  </div>
                  <div className="grid gap-2">
                    <Button type="button" disabled={acting === "batch-create"} onClick={() => void createBatchPosts()} className="h-10 rounded-lg bg-[#1d4ed8] hover:bg-[#1e40af]"><Plus className="size-4" />创建草稿</Button>
                    <Button type="button" variant="outline" disabled={acting === "batch-async-create"} onClick={() => void createBatchPostsAsync()} className="h-10 rounded-lg border-black/[0.08] bg-white hover:bg-[#f6f7fb]"><UploadCloud className="size-4" />异步创建</Button>
                    <div className="flex gap-2">
                      <input value={batchStatusId} onChange={(event) => setBatchStatusId(event.target.value)} className="h-10 min-w-0 flex-1 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]" placeholder="批次编号" />
                      <Button type="button" variant="outline" disabled={acting === "batch-status"} onClick={() => void checkBatchStatus()} className="h-10 rounded-lg border-black/[0.08] bg-white"><Search className="size-4" />状态</Button>
                    </div>
                  </div>
                </div>
              </Panel>
            </>
          ) : null}

          {operationTab === "growth" ? (
            <div className="grid gap-4">
              <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(340px,0.8fr)]">
                <Panel title="推荐配置" icon={SlidersHorizontal}>
                  <form onSubmit={saveRecommendConfig} className="grid gap-3">
                    <div className="grid gap-3 md:grid-cols-2">
                      {recommendConfigEntries(recommendConfig).map(([key, value]) => (
                        <label key={key} className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
                          <span className="mb-2 block text-xs font-semibold text-[#666c78]">{columnLabel(key)}</span>
                          {typeof value === "boolean" ? (
                            <ToggleSwitch value={value} onChange={(next) => setRecommendConfig((current) => ({ ...current, [key]: next }))} />
                          ) : (
                            <input
                              value={String(value ?? "")}
                              onChange={(event) => setRecommendConfig((current) => ({ ...current, [key]: parseConfigValue(event.target.value, value) }))}
                              type={typeof value === "number" ? "number" : "text"}
                              className="h-10 w-full rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
                            />
                          )}
                        </label>
                      ))}
                    </div>
                    <div className="flex justify-end">
                      <Button type="submit" disabled={acting === "recommend-config"} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]"><Save className="size-4" />保存推荐配置</Button>
                    </div>
                  </form>
                </Panel>
                <Panel title="主动推荐" icon={Zap}>
                  <form onSubmit={pushRecommendation} className="grid gap-3">
                    <AdminObjectPicker
                      token={token}
                      resource="posts"
                      label="推荐内容"
                      value={pushPosts}
                      onChange={setPushPosts}
                      placeholder="搜索标题、正文或作者"
                      emptyLabel="未找到内容"
                    />
                    <label className="grid gap-1.5">
                      <span className="text-xs font-semibold text-[#666c78]">加权分</span>
                      <input value={pushScore} onChange={(event) => setPushScore(event.target.value)} type="number" className="h-10 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]" />
                    </label>
                    <Button type="submit" disabled={acting === "push"} className="h-10 rounded-lg bg-[#17171d] px-3 hover:bg-[#2a2b32]"><Zap className="size-4" />推送到推荐</Button>
                  </form>
                </Panel>
              </div>
              <Panel title="推荐规则批量写入" icon={Zap}>
                <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_320px]">
                  <AdminObjectPicker
                    token={token}
                    resource="posts"
                    label="推荐内容"
                    multiple
                    value={recommendBatchPosts}
                    onChange={setRecommendBatchPosts}
                    placeholder="搜索标题、正文或作者"
                    emptyLabel="未找到内容"
                  />
                  <div className="grid content-start gap-2">
                    <input value={recommendBatchBoost} onChange={(event) => setRecommendBatchBoost(event.target.value)} type="number" className="h-10 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]" placeholder="加权分" />
                    <input value={recommendBatchReason} onChange={(event) => setRecommendBatchReason(event.target.value)} className="h-10 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]" placeholder="推荐原因" />
                    <div className="grid grid-cols-2 gap-2">
                      <ToggleSwitch value={recommendBatchPinned} onChange={setRecommendBatchPinned} onLabel="置顶" offLabel="不置顶" />
                      <ToggleSwitch value={recommendBatchSuppressed} onChange={setRecommendBatchSuppressed} onLabel="抑制" offLabel="不抑制" />
                    </div>
                    <Button type="button" disabled={acting === "recommend-batch"} onClick={() => void runRecommendationBatch()} className="h-10 rounded-lg bg-[#1d4ed8] hover:bg-[#1e40af]"><Save className="size-4" />批量保存</Button>
                  </div>
                </div>
              </Panel>
            </div>
          ) : null}

          {operationTab === "assets" ? (
            <div className="grid gap-4">
              <Panel title="App 使用概览" icon={Package}>
                <AppStatsPanel stats={appStats} />
              </Panel>
              <div className="grid gap-4 xl:grid-cols-2">
                <Panel title="APK 文件" icon={Package}>
                  <FileList files={apkFiles} empty="暂无 APK 文件" />
                </Panel>
                <Panel
                  title="批量素材文件"
                  icon={Folder}
                  action={
                    <div className="flex flex-wrap gap-1">
                      <Button type="button" size="sm" variant="outline" disabled={acting === "batch-delete-images"} onClick={() => void deleteBatchFiles("images")} className="h-8 rounded-lg border-black/[0.08] bg-white px-2 text-xs">清图片</Button>
                      <Button type="button" size="sm" variant="outline" disabled={acting === "batch-delete-videos"} onClick={() => void deleteBatchFiles("videos")} className="h-8 rounded-lg border-black/[0.08] bg-white px-2 text-xs">清视频</Button>
                      <Button type="button" size="sm" variant="outline" disabled={acting === "batch-delete-all"} onClick={() => void deleteBatchFiles("all")} className="h-8 rounded-lg border-[#dc2626]/20 bg-white px-2 text-xs text-[#b91c1c] hover:bg-[#fef2f2]">清全部</Button>
                    </div>
                  }
                >
                  <BatchFilesPanel payload={batchFiles} />
                </Panel>
              </div>
            </div>
          ) : null}

          {operationTab === "finance" ? (
            <Panel title="收益调账" icon={CircleDollarSign}>
              <div className="grid gap-3">
                <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_220px]">
                  <AdminObjectPicker
                    token={token}
                    resource="users"
                    label="调账用户"
                    value={earningsUsers}
                    onChange={setEarningsUsers}
                    placeholder="搜索昵称、账号或邮箱"
                    emptyLabel="未找到用户"
                  />
                  <input value={earningsAmount} onChange={(event) => setEarningsAmount(event.target.value)} type="number" className="h-10 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]" placeholder="金额" />
                </div>
                <div className="flex flex-wrap gap-2">
                  <Button type="button" variant="outline" disabled={acting === "earnings-info"} onClick={() => void runEarnings("info")} className="rounded-lg border-black/[0.08] bg-white"><Search className="size-4" />查询</Button>
                  <Button type="button" disabled={acting === "earnings-add"} onClick={() => void runEarnings("add")} className="rounded-lg bg-[#18a058] hover:bg-[#138a4a]"><Plus className="size-4" />增加收益</Button>
                  <Button type="button" variant="outline" disabled={acting === "earnings-deduct"} onClick={() => void runEarnings("deduct")} className="rounded-lg border-[#dc2626]/20 bg-white text-[#b91c1c] hover:bg-[#fef2f2]"><Trash2 className="size-4" />扣减收益</Button>
                  <Button type="button" variant="outline" disabled={acting === "earnings-transfer"} onClick={() => void runEarnings("transfer")} className="rounded-lg border-black/[0.08] bg-white"><Wallet className="size-4" />余额转收益</Button>
                </div>
              </div>
            </Panel>
          ) : null}

          {operationResult ? <OperationResultPanel result={operationResult} /> : null}
        </section>
      )}
    </div>
  );
}

