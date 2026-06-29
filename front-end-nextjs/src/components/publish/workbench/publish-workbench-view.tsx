import {
LanguageSwitcher
} from "@/components/language-switcher";
import {
Button
} from "@/components/ui/button";
import {
cn
} from "@/lib/utils";
import {
ChevronDown,
Clock3,
Loader2,
LogOut,
PanelLeftClose,
Plus,
  Save,
  Send,
  Settings2,
  Sparkles
} from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import {
AIFormatPanel
} from "../shared/ai-format-panel";
import {
PublishGenerationCard
} from "../shared/publish-generation-card";
import {
PublishGenerationPanel
} from "../shared/publish-generation-panel";
import type { PublishGenerationRunOptions } from "../shared/publish-generation-action";
import {
ArticleComposer
} from "./article-composer";
import {
CreatorHome
} from "./creator-home";
import {
DraftsPanel
} from "./drafts-panel";
import {
UploadPanel
} from "./upload-panel";
import type { usePublishWorkbenchController } from "./use-publish-workbench-controller";
import {
CreatorWithdrawManagement
} from "./withdraw-management";
import {
PaymentMethod,
publishModes,
RichTextEditor,
sideNavItems,
visibilityOptions
} from "./workbench-config";

export function PublishWorkbenchView({ controller }: { controller: ReturnType<typeof usePublishWorkbenchController> }) {
  const { accountMenuOpen, accountMenuRef, activeSection, aiFormatOpen, articleComposerOpen, body, bodyContentLength, bodyLimit, cancelPublishGeneration, categories, completion, draftActionId, drafts, draftsLoading, draftsOpen, handleApplyPublishGeneration, handleBatchUpdateImages, handleDeleteDraft, handleGeneratePublishContent, handleLogout, handleMoveAsset, handleOpenDrafts, handlePaymentMethodChange, handleRemoveAsset, handleRemoveImageAssets, handleRemoveUploadFailure, handleReorderImageAssets, handleReplaceThumbnail, handleRetryUpload, handleSubmitPost, handleToggleImageFree, handleToggleImageProtection, handleUploadFiles, imagePaymentSettings, imageProtectionEnabled, imageProtectionNoticeEnabled, imageSelectAllEnabled, isLoggingOut, isSubmitting, mode, openComposer, paidContentEnabled, paidContentPaymentMethods, paidImageCount, paymentMaxPrices, pendingFiles, publishGeneration, publishGenerationCanRun, publishGenerationImageCount, refreshDrafts, replaceThumbnailInputRef, restoreDraft, selectedCategoryId, setAIFormatOpen, setAccountMenuOpen, setActiveSection, setArticleComposerOpen, setBody, setDraftsOpen, setImagePaymentSettings, setMode, setSelectedCategoryId, setTags, setTitle, setVisibility, t, tags, title, titleLimit, uploadAbortControllerRef, uploadFailures, uploadProgress, uploadProgressDetails, uploadedAssets, uploadingMode, visibility } = controller;
  const [publishGenerationOpen, setPublishGenerationOpen] = useState(false);
  const runPublishGenerationInPanel = (options?: PublishGenerationRunOptions) => {
    setPublishGenerationOpen(true);
    void handleGeneratePublishContent(options);
  };
  return (
    <div className="min-h-screen bg-[#f6f6f7] text-[#1f1f24] [--publish-sidebar-width:clamp(176px,16vw,224px)]">
      <header className="fixed inset-x-0 top-0 z-40 flex h-16 items-center border-b border-[#eeeeef] bg-white px-6">
        <Link href="/" className="flex h-9 items-center gap-3 rounded-full outline-none focus-visible:ring-2 focus-visible:ring-primary">
          <span className="flex h-9 w-[78px] items-center justify-center rounded-full bg-primary text-[17px] font-black tracking-normal text-white">
            {t("app.name")}
          </span>
          <span className="text-lg font-semibold text-[#25252b]">
            {t("publish.header.title")}
          </span>
        </Link>

        <div className="ml-auto flex items-center gap-3">
          <LanguageSwitcher tone="light" />
          <div ref={accountMenuRef} className="relative">
            <button
              type="button"
              aria-haspopup="menu"
              aria-expanded={accountMenuOpen}
              onClick={() => setAccountMenuOpen((open) => !open)}
              className="flex h-9 items-center gap-2 rounded-full px-2 text-sm text-[#4c4c54] hover:bg-[#f5f5f6]"
            >
              <span className="flex size-7 items-center justify-center rounded-full bg-[#1f1f24] text-xs font-semibold text-white">
                Y
              </span>
              <span className="max-w-[160px] truncate">{t("publish.home.profile.name")}</span>
              <ChevronDown
                className={cn(
                  "size-4 text-[#8a8a91] transition-transform",
                  accountMenuOpen && "rotate-180",
                )}
              />
            </button>

            {accountMenuOpen ? (
              <div
                role="menu"
                className="absolute right-0 top-full z-50 mt-2 w-44 overflow-hidden rounded-2xl border border-[#eeeeef] bg-white p-1 shadow-xl shadow-black/10"
              >
                <Link
                  href="/profile"
                  onClick={() => setAccountMenuOpen(false)}
                  role="menuitem"
                  className="flex h-10 items-center rounded-xl px-3 text-sm font-medium text-[#4c4c54] hover:bg-[#f6f6f7] hover:text-[#1f1f24]"
                >
                  {t("publish.header.profile")}
                </Link>
                <button
                  type="button"
                  role="menuitem"
                  onClick={() => void handleLogout()}
                  disabled={isLoggingOut}
                  className="flex h-10 w-full items-center gap-2 rounded-xl px-3 text-left text-sm font-medium text-[#f43f5e] hover:bg-[#fff1f3] disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {isLoggingOut ? (
                    <Loader2 className="size-4 animate-spin" />
                  ) : (
                    <LogOut className="size-4" />
                  )}
                  <span>{t("publish.header.logout")}</span>
                </button>
              </div>
            ) : null}
          </div>
        </div>
      </header>

      <aside className="fixed bottom-0 left-0 top-16 z-30 hidden w-[var(--publish-sidebar-width)] flex-col border-r border-[#eeeeef] bg-white lg:flex">
        <div className="px-[clamp(16px,2vw,24px)] pb-4 pt-6">
          <Button
            type="button"
            onClick={() => openComposer()}
            className="h-11 w-full justify-start gap-3 px-5 text-[15px] font-semibold shadow-none"
          >
            <Plus className="size-5" />
            {t("publish.sidebar.publishNote")}
            <ChevronDown className="ml-auto size-4" />
          </Button>
        </div>
        <nav className="min-w-0 flex-1 px-[clamp(16px,2vw,24px)]">
          {sideNavItems.map(({ key, icon: Icon }) => (
            <button
              key={key}
              type="button"
              onClick={() => {
                const nextSection =
                  key === "home" ? "home" : key === "withdrawManagement" ? "withdraw" : "publish";
                setActiveSection(nextSection);
              }}
              className={cn(
                "flex h-11 w-full items-center gap-3 rounded-xl px-3 text-sm font-medium text-[#67676f] transition-colors hover:bg-[#f6f6f7] hover:text-[#1f1f24]",
                key === "home" && activeSection === "home" && "bg-[#eeeeef] text-[#1f1f24]",
                key === "noteManagement" && activeSection === "publish" && "bg-[#eeeeef] text-[#1f1f24]",
                key === "withdrawManagement" && activeSection === "withdraw" && "bg-[#eeeeef] text-[#1f1f24]",
              )}
            >
              <Icon className="size-5" />
              <span className="truncate">{t(`publish.sidebar.${key}`)}</span>
            </button>
          ))}
        </nav>

        <button
          type="button"
          className="mx-[clamp(16px,2vw,24px)] mb-4 mt-auto flex h-11 items-center gap-3 rounded-xl px-3 text-sm font-medium text-[#67676f] hover:bg-[#f6f6f7]"
        >
          <PanelLeftClose className="size-5" />
          {t("publish.sidebar.collapse")}
        </button>
      </aside>

      <main className="min-h-screen pt-16 lg:pl-[var(--publish-sidebar-width)]">
        {activeSection === "home" ? (
          <CreatorHome onCreate={openComposer} />
        ) : activeSection === "withdraw" ? (
          <CreatorWithdrawManagement />
        ) : (
          <>
        <div
          className={cn(
            "mx-auto grid min-h-[calc(100vh-64px)] w-full max-w-[1440px] grid-cols-1 gap-5 px-4 py-6 sm:px-5 lg:px-6 2xl:px-8",
            mode === "article" && articleComposerOpen
              ? "xl:grid-cols-1"
              : "xl:grid-cols-[minmax(0,1fr)_minmax(240px,0.28fr)]",
          )}
        >
          <section className="min-w-0 flex-1">
            <div className="mb-4 flex min-h-10 flex-wrap items-center gap-3">
              <div className="flex min-w-0 flex-wrap items-center gap-2">
                {publishModes.map(({ key, icon: Icon }) => {
                  const active = mode === key;

                  return (
                    <button
                      key={key}
                      type="button"
                      onClick={() => {
                        setMode(key);
                        if (key !== "article") {
                          setArticleComposerOpen(false);
                        }
                      }}
                      className={cn(
                        "relative flex h-10 items-center gap-2 rounded-full px-4 text-sm font-semibold text-[#777780] transition-colors hover:bg-white hover:text-[#1f1f24]",
                        active && "bg-white text-[#1f1f24] shadow-sm",
                      )}
                    >
                      <Icon className="size-4" />
                      {t(`publish.modes.${key}`)}
                      {active ? (
                        <span
                          className="absolute inset-x-4 -bottom-1 h-0.5 rounded-full bg-primary"
                        />
                      ) : null}
                    </button>
                  );
                })}
              </div>

              <div className="ml-auto flex min-w-0 flex-wrap items-center justify-end gap-2">
                <button
                  type="button"
                  onClick={handleOpenDrafts}
                  className="flex h-10 items-center gap-2 rounded-full bg-white px-4 text-sm font-semibold text-[#55555d] shadow-sm hover:text-primary"
                >
                  <Clock3 className="size-4" />
                  {t("publish.drafts", { count: drafts.length })}
                </button>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => handleSubmitPost(true)}
                  disabled={isSubmitting || Boolean(uploadingMode)}
                  className="h-10 border-[#e3e3e6] bg-white px-5 text-[#55555d] hover:bg-[#f7f7f8]"
                >
                  <Save className="size-4" />
                  {t("publish.footer.saveDraft")}
                </Button>
                <Button
                  type="button"
                  onClick={() => handleSubmitPost(false)}
                  disabled={isSubmitting || Boolean(uploadingMode)}
                  className="h-10 px-6"
                >
                  <Send className="size-4" />
                  {t("publish.footer.publish")}
                </Button>
              </div>
            </div>
            {draftsOpen ? (
              <DraftsPanel
                actionId={draftActionId}
                drafts={drafts}
                loading={draftsLoading}
                onClose={() => setDraftsOpen(false)}
                onDelete={(draftId) => void handleDeleteDraft(draftId)}
                onRefresh={() => void refreshDrafts()}
                onRestore={restoreDraft}
              />
            ) : null}

            <div className={cn(
              "rounded-2xl bg-white p-4 shadow-[0_8px_28px_rgba(0,0,0,0.04)]",
              mode === "article" && articleComposerOpen && "min-h-[calc(100vh-220px)]",
            )}>
              {mode === "article" ? (
                <ArticleComposer
                  body={body}
                  bodyContentLength={bodyContentLength}
                  bodyLimit={bodyLimit}
                  categories={categories}
                  categoryId={selectedCategoryId}
                  completion={completion}
                  composerOpen={articleComposerOpen}
                  onBodyChange={setBody}
                  onCategoryChange={setSelectedCategoryId}
                  onOpenComposer={() => setArticleComposerOpen(true)}
                  onTagsChange={setTags}
                  onTitleChange={setTitle}
                  onVisibilityChange={setVisibility}
                  tags={tags}
                  title={title}
                  titleLimit={titleLimit}
                  visibility={visibility}
                />
              ) : (
                <>
                  <input
                    ref={replaceThumbnailInputRef}
                    type="file"
                    accept="image/*"
                    onChange={(event) => {
                      const file = event.target.files?.[0];
                      event.target.value = "";
                      if (!file) {
                        return;
                      }
                      const reader = new FileReader();
                      reader.onload = () => {
                        if (typeof reader.result === "string") {
                          handleReplaceThumbnail(reader.result);
                        }
                      };
                      reader.readAsDataURL(file);
                    }}
                    className="hidden"
                  />
                  <UploadPanel
                    assets={uploadedAssets[mode]}
                    failures={uploadFailures[mode]}
                    imageProtectionEnabled={imageProtectionEnabled}
                    imageProtectionNoticeEnabled={imageProtectionNoticeEnabled}
                    imageSelectAllEnabled={imageSelectAllEnabled}
                    paidContentEnabled={paidContentEnabled}
                    mode={mode}
                    progress={uploadProgress[mode]}
                    progressDetail={uploadProgressDetails[mode]}
                    thumbnailDataUrl={mode === "video" ? (pendingFiles.video[0]?.thumbnailDataUrl ?? null) : null}
                    uploading={uploadingMode === mode}
                    onCancelUpload={() => uploadAbortControllerRef.current?.abort()}
                    onBatchUpdateImages={handleBatchUpdateImages}
                    onRemoveAsset={(assetUrl) => handleRemoveAsset(mode, assetUrl)}
                    onRemoveAssets={handleRemoveImageAssets}
                    onRemoveFailure={(failureId) => handleRemoveUploadFailure(mode, failureId)}
                    onMoveAsset={(assetUrl, direction) => handleMoveAsset(mode, assetUrl, direction)}
                    onReorderAssets={handleReorderImageAssets}
                    onReplaceThumbnail={() => replaceThumbnailInputRef.current?.click()}
                    onRetryFailure={(failure) => handleRetryUpload(mode, failure)}
                    onToggleImageFree={handleToggleImageFree}
                    onToggleImageProtection={handleToggleImageProtection}
                    onUploadFiles={(files) => handleUploadFiles(files, mode)}
                  />
                </>
              )}
            </div>

            {mode === "image" ? (
              <div className="mt-5">
                <PublishGenerationCard
                  canRun={publishGenerationCanRun}
                  imageCount={publishGenerationImageCount}
                  onCancel={cancelPublishGeneration}
                  onOpen={() => setPublishGenerationOpen(true)}
                  onRun={runPublishGenerationInPanel}
                  state={publishGeneration}
                  t={t}
                />
              </div>
            ) : null}

            {mode !== "article" ? (
            <div className="mt-5 grid grid-cols-1 gap-5 xl:grid-cols-[minmax(0,1fr)_minmax(260px,0.42fr)]">
              <section className="rounded-2xl bg-white p-5 shadow-[0_8px_28px_rgba(0,0,0,0.04)]">
                <div className="mb-4 flex items-center justify-between">
                  <h2 className="text-base font-semibold text-[#25252b]">
                    {t("publish.editor.title")}
                  </h2>
                  <span className="text-xs text-[#9a9aa1]">
                    {t("publish.editor.completion", { percent: completion })}
                  </span>
                </div>

                <div className="space-y-4">
                  <label className="block">
                    <span className="mb-2 block text-sm font-medium text-[#4f4f57]">
                      {t("publish.editor.titleLabel")}
                    </span>
                    <input
                      value={title}
                      onChange={(event) => setTitle(event.target.value.slice(0, titleLimit))}
                      placeholder={t("publish.editor.titlePlaceholder")}
                      className="h-12 w-full rounded-xl border border-[#e8e8eb] bg-white px-4 text-[15px] outline-none transition focus:border-primary"
                    />
                    <span className="mt-1 block text-right text-xs text-[#a5a5ab]">
                      {title.length}/{titleLimit}
                    </span>
                  </label>

                  <div className="block">
                    <div className="mb-2 flex items-center justify-between gap-3">
                      <span className="block text-sm font-medium text-[#4f4f57]">
                        {t("publish.editor.bodyLabel")}
                      </span>
                      <button
                        type="button"
                        onClick={() => setAIFormatOpen(true)}
                        className="flex h-8 shrink-0 items-center gap-1.5 rounded-full border border-[#dbe7ff] bg-[#f4f8ff] px-3 text-xs font-semibold text-[#1d4ed8] transition hover:bg-[#eaf2ff]"
                      >
                        <Sparkles className="size-3.5" />
                        <span>{t("publish.aiFormat.trigger")}</span>
                      </button>
                    </div>
                    <RichTextEditor
                      limit={bodyLimit}
                      onChange={setBody}
                      placeholder={t(`publish.editor.${mode}Placeholder`)}
                      value={body}
                    />
                  </div>

                  <div className="grid gap-4">
                    <label className="block">
                      <span className="mb-2 block text-sm font-medium text-[#4f4f57]">
                        {t("publish.editor.tagsLabel")}
                      </span>
                      <input
                        value={tags}
                        onChange={(event) => setTags(event.target.value)}
                        placeholder={t("publish.editor.tagsPlaceholder")}
                        className="h-11 w-full rounded-xl border border-[#e8e8eb] bg-white px-4 text-sm outline-none transition focus:border-primary"
                      />
                    </label>
                  </div>
                </div>
              </section>

              <aside className="rounded-2xl bg-white p-5 shadow-[0_8px_28px_rgba(0,0,0,0.04)]">
                <h2 className="flex items-center gap-2 text-base font-semibold text-[#25252b]">
                  <Settings2 className="size-4" />
                  {t("publish.settings.title")}
                </h2>

                <div className="mt-5 space-y-5">
                  <div>
                    <p className="mb-2 text-sm font-medium text-[#4f4f57]">
                      {t("publish.settings.visibility")}
                    </p>
                    <div className="grid grid-cols-3 gap-2">
                      {visibilityOptions.map((option) => (
                        <button
                          key={option}
                          type="button"
                          onClick={() => setVisibility(option)}
                          className={cn(
                            "h-9 rounded-xl border border-[#e8e8eb] text-xs font-semibold text-[#66666f] transition-colors hover:border-primary/40",
                            visibility === option && "border-primary bg-[#fff1f3] text-primary",
                          )}
                        >
                          {t(`publish.settings.${option}`)}
                        </button>
                      ))}
                    </div>
                  </div>

                  {mode === "image" && paidImageCount > 0 ? (
                    <div className="rounded-2xl border border-[#eeeeef] bg-[#fafafa] p-3">
                      <p className="text-sm font-semibold text-[#34343a]">
                        {t("publish.protection.paymentTitle", { count: paidImageCount })}
                      </p>
                      <div className="mt-3 grid grid-cols-2 gap-2">
                        {(["balance", "points"] as PaymentMethod[]).map((method) => (
                          <button
                            key={method}
                            type="button"
                            onClick={() => handlePaymentMethodChange(method)}
                            disabled={!paidContentPaymentMethods[method]}
                            className={cn(
                              "h-9 rounded-xl border border-[#e8e8eb] text-xs font-semibold text-[#66666f] transition-colors hover:border-primary/40 disabled:cursor-not-allowed disabled:opacity-40",
                              imagePaymentSettings.paymentMethod === method && "border-primary bg-[#fff1f3] text-primary",
                            )}
                          >
                            {t(`publish.protection.${method}`)}
                          </button>
                        ))}
                      </div>
                      <label className="mt-3 block">
                        <span className="mb-1 block text-xs font-medium text-[#777780]">
                          {t("publish.protection.priceLabel")}
                        </span>
                        <input
                          value={imagePaymentSettings.price}
                          inputMode="decimal"
                          max={paymentMaxPrices[imagePaymentSettings.paymentMethod]}
                          onChange={(event) =>
                            setImagePaymentSettings((current) => ({
                              ...current,
                              price: event.target.value.replace(/[^\d.]/g, "").slice(0, 8),
                            }))
                          }
                          placeholder={t("publish.protection.pricePlaceholder")}
                          className="h-10 w-full rounded-xl border border-[#e8e8eb] bg-white px-3 text-sm outline-none transition focus:border-primary"
                        />
                      </label>
                    </div>
                  ) : null}

                </div>
              </aside>
            </div>
            ) : null}
          </section>

          <aside className={cn(
            "hidden min-w-0 space-y-5 xl:block",
            mode === "article" && articleComposerOpen && "xl:hidden",
          )}>
            <section className="rounded-2xl bg-white p-5 shadow-[0_8px_28px_rgba(0,0,0,0.04)]">
              <h2 className="text-base font-semibold text-[#25252b]">
                {t("publish.tips.title")}
              </h2>
              <div className="mt-4 space-y-3">
                {["cover", "hook", "tags"].map((tip) => (
                  <div key={tip} className="rounded-xl bg-[#f7f7f8] p-3">
                    <p className="text-sm font-semibold text-[#3c3c43]">
                      {t(`publish.tips.${tip}Title`)}
                    </p>
                    <p className="mt-1 text-xs leading-5 text-[#777780]">
                      {t(`publish.tips.${tip}Body`)}
                    </p>
                  </div>
                ))}
              </div>
            </section>
          </aside>
        </div>

          </>
        )}
      </main>
      <AIFormatPanel
        open={aiFormatOpen}
        value={body}
        onApply={setBody}
        onClose={() => setAIFormatOpen(false)}
      />
      <PublishGenerationPanel
        canRun={publishGenerationCanRun}
        imageCount={publishGenerationImageCount}
        onApply={handleApplyPublishGeneration}
        onCancel={cancelPublishGeneration}
        onClose={() => setPublishGenerationOpen(false)}
        onRun={runPublishGenerationInPanel}
        open={publishGenerationOpen}
        state={publishGeneration}
        t={t}
      />
    </div>
  );
}
