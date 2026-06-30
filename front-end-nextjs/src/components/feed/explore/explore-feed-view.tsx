import {
Button
} from "@/components/ui/button";
import {
TooltipProvider
} from "@/components/ui/tooltip";
import {
cn
} from "@/lib/utils";
import {
Menu,
Plus,
RefreshCw,
Search
} from "lucide-react";
import { LoginUnlockDialog } from "../login-unlock-dialog";
import {
Masonry
} from "react-plock";
import {
PostCard
} from "../post-card";
import {
  Lightbox,
followingSortTabs
} from "./explore-config";
import { ExploreMoreSheet } from "./explore-more-sheet";
import {
ExploreDesktopSidebar,
ExploreSiteBrand,
ExploreMobileBottomNav,
} from "./explore-navigation";
import {
SearchForm
} from "./explore-widgets";
import type { useExploreFeedController } from "./use-explore-feed-controller";

export function ExploreFeedView({ controller }: { controller: ReturnType<typeof useExploreFeedController> }) {
  const { activeCategory, activeMode, activeSearch, bottomSentinelRef, canLoadMore, categories, categorySwipeStartRef, categoryVirtualizer, clearSearch, exploreTheme, exploreThemePreference, exploreThemeVars, feedQuery, feedSwipeStartRef, fetchPageWithAnchor, followingSort, handleCategoryPointerEnd, handleCategoryPointerStart, handleFeedClickCapture, handleFeedPointerEnd, handleFeedPointerStart, handleLike, handleLogout, handleSearchSubmit, hasClientAccessToken, isLightTheme, isLoggingOut, isPending, isReplacingFeed, isWalletBalanceLoading, lightboxIndex, lightboxSlides, loadCategory, loadMode, loginUnlockOpen, messageBadgeCount, mobileMoreOpen, mobileSearchOpen, noMoreUnlockSentinelRef, posts, priorityPostIds, refreshFeed, scrollCategoryIntoView, searchActive, searchInput, setExploreTheme, setExploreThemePreference, setLightboxIndex, setLoginUnlockOpen, setMobileMoreOpen, setMobileSearchOpen, setSearchInput, setThemeSettingsOpen, showNextLoading, showPreviousLoading, siteProfile, t, themeSettingsOpen, toolbarItems, topSentinelRef, visibleDesktopNavItems, walletBalance, walletBalanceError, warmNavigationTarget } = controller;
  return (
    <div
      className="flex h-dvh flex-col overflow-x-hidden bg-[var(--explore-bg)] text-[var(--explore-text)]"
      data-theme={exploreTheme}
      style={exploreThemeVars}
    >
      <ExploreDesktopSidebar
        messageBadgeCount={messageBadgeCount}
        mobileMoreOpen={mobileMoreOpen}
        onOpenMore={() => setMobileMoreOpen(true)}
        siteProfile={siteProfile}
        toolbarItems={toolbarItems}
        visibleItems={visibleDesktopNavItems}
        warmNavigationTarget={warmNavigationTarget}
      />

      <main
        className="flex min-h-0 flex-1 flex-col pb-[calc(3.5rem+env(safe-area-inset-bottom))] lg:block lg:pb-0 lg:pl-[164px]"
      >
        <header className="fixed inset-x-0 top-0 z-30 h-[72px] bg-[var(--explore-bg)] lg:hidden">
          <div className="flex h-full items-center justify-between gap-3 px-4">
            <ExploreSiteBrand siteProfile={siteProfile} compact />
            <div className="flex items-center gap-1">
              <Button
                variant="ghost"
                size="icon"
                aria-label={t("search.submit")}
                aria-expanded={mobileSearchOpen}
                className="size-10 text-[var(--explore-text)] hover:bg-[var(--explore-control)]"
                onClick={() => setMobileSearchOpen((open) => !open)}
              >
                <Search className="size-5" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                aria-label={t("nav.more")}
                aria-expanded={mobileMoreOpen}
                className="size-10 text-[var(--explore-text)] hover:bg-[var(--explore-control)]"
                onClick={() => setMobileMoreOpen(true)}
              >
                <Menu className="size-5" />
              </Button>
            </div>
          </div>
        </header>

        {mobileSearchOpen ? (
          <div className="fixed inset-x-0 top-[72px] z-30 bg-[var(--explore-bg)] px-4 pb-3 lg:hidden">
            <SearchForm
              value={searchInput}
              placeholder={t("search.placeholder")}
              submitLabel={t("search.submit")}
              clearLabel={t("search.clear")}
              active={searchActive}
              disabled={false}
              onChange={setSearchInput}
              onClear={clearSearch}
              onSubmit={handleSearchSubmit}
            />
          </div>
        ) : null}

        <ExploreMoreSheet
          exploreThemePreference={exploreThemePreference}
          hasClientAccessToken={hasClientAccessToken}
          isLightTheme={isLightTheme}
          isLoggingOut={isLoggingOut}
          isWalletBalanceLoading={isWalletBalanceLoading}
          mobileMoreOpen={mobileMoreOpen}
          onLogout={() => void handleLogout()}
          setExploreTheme={setExploreTheme}
          setExploreThemePreference={setExploreThemePreference}
          setMobileMoreOpen={setMobileMoreOpen}
          setThemeSettingsOpen={setThemeSettingsOpen}
          t={t}
          themeSettingsOpen={themeSettingsOpen}
          walletBalance={walletBalance}
          walletBalanceError={walletBalanceError}
        />

        <div className="hidden px-8 pt-[62px] lg:block">
          <form
            onSubmit={handleSearchSubmit}
            suppressHydrationWarning
            className="mx-auto flex h-[117px] max-w-[920px] flex-col justify-between rounded-[24px] border border-[var(--explore-border)] bg-[var(--explore-surface)] px-4 pb-[17px] pt-5 shadow-[0_16px_40px_rgba(0,0,0,0.08)]"
          >
            <input
              type="search"
              value={searchInput}
              onChange={(event) => setSearchInput(event.target.value)}
              aria-label={t("search.submit")}
              autoComplete="off"
              enterKeyHint="search"
              placeholder={t("search.placeholder")}
              suppressHydrationWarning
              className="min-h-0 flex-1 bg-transparent px-1 text-base text-[var(--explore-strong)] outline-none placeholder:text-[var(--explore-subtle)]"
            />
            <div className="flex h-8 items-center gap-3">
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label={t("search.compose")}
                className="size-8 text-[var(--explore-muted)] hover:bg-[var(--explore-control)] hover:text-[var(--explore-strong)]"
              >
                <Plus className="size-5" />
              </Button>
              {searchActive ? (
                <>
                  <div className="h-4 w-px bg-[var(--explore-border)]" />
                  <span className="flex min-w-0 items-center gap-2 text-base text-[var(--explore-strong)]">
                    <span className="size-2 shrink-0 rounded-full border border-[var(--explore-muted)]" />
                    <span className="min-w-0 truncate">
                      {t("search.resultFor", { keyword: activeSearch })}
                    </span>
                  </span>
                </>
              ) : null}
              {searchActive ? (
                <button
                  type="button"
                  onClick={clearSearch}
                  className="h-8 rounded-full px-3 text-xs font-semibold text-[var(--explore-muted)] hover:bg-[var(--explore-control)] hover:text-[var(--explore-strong)]"
                >
                  {t("search.clear")}
                </button>
              ) : null}
              <Button
                type="submit"
                variant="ghost"
                size="icon"
                aria-label={t("search.submit")}
                className="ml-auto size-8 bg-[var(--explore-control)] text-[var(--explore-strong)] hover:bg-[var(--explore-control-hover)]"
              >
                <Search className="size-4" />
              </Button>
            </div>
          </form>
        </div>

        <div className={cn("pt-[72px] lg:pt-[34px]", mobileSearchOpen && "pt-[132px]")}>
          <nav className="mx-[10px] h-11 overflow-hidden lg:mx-8 lg:h-16">
            {activeMode === "following" ? (
              <div className="flex h-full items-center gap-2 overflow-x-auto overflow-y-hidden [scrollbar-width:none] lg:mx-auto lg:max-w-[756px] [&::-webkit-scrollbar]:hidden">
                {followingSortTabs.map((tab) => {
                  const active = followingSort === tab.sort;

                  return (
                    <button
                      key={tab.sort}
                      type="button"
                      aria-pressed={active}
                      onClick={() => loadMode("following", "all", tab.sort)}
                        className={cn(
                          "flex h-8 min-w-16 items-center justify-center rounded-full px-4 text-[13px] font-medium transition-colors lg:h-10 lg:px-5 lg:text-sm",
                          active
                            ? "bg-primary text-white shadow-[0_8px_20px_rgba(255,36,66,0.18)]"
                            : "bg-[var(--explore-control)] text-[var(--explore-muted)] hover:text-[var(--explore-strong)] active:scale-[0.98]",
                        )}
                    >
                      {t(`tabs.${tab.labelKey}`)}
                    </button>
                  );
                })}
              </div>
            ) : (
              <div
                id="category-strip"
                className="h-full max-w-full overflow-x-auto overflow-y-hidden overscroll-x-contain [scrollbar-width:none] [&::-webkit-scrollbar]:hidden lg:mx-auto lg:max-w-[756px]"
                onPointerDown={handleCategoryPointerStart}
                onPointerCancel={() => {
                  categorySwipeStartRef.current = null;
                }}
                onPointerUp={handleCategoryPointerEnd}
              >
                <div
                  className="relative h-full"
                  style={{ width: categoryVirtualizer.getTotalSize() }}
                >
                  {categoryVirtualizer.getVirtualItems().map((virtualItem) => {
                    const category = categories[virtualItem.index];
                    const isAll = category.id === -1;
                    const active =
                      (isAll && activeCategory === "all") ||
                      activeCategory === category.id;

                    return (
                      <button
                        key={category.id}
                        type="button"
                        data-category-id={category.id}
                        aria-pressed={active}
                        onClick={() => {
                          loadCategory(isAll ? "all" : category.id);
                          window.requestAnimationFrame(() => scrollCategoryIntoView(category));
                        }}
                        className={cn(
                          "absolute top-2 flex h-8 items-center justify-center rounded-full px-4 text-[13px] font-medium transition-[background-color,color,box-shadow,transform] duration-200 ease-out active:scale-[0.98] lg:top-2.5 lg:h-10 lg:bg-transparent lg:px-3 lg:text-sm",
                          active
                            ? "bg-primary text-white shadow-[0_8px_20px_rgba(255,36,66,0.16)] lg:text-white"
                            : "bg-[var(--explore-control)] text-[var(--explore-muted)] hover:text-[var(--explore-strong)] lg:bg-transparent lg:text-[var(--explore-muted)]",
                        )}
                        style={{
                          left: virtualItem.start,
                          width: virtualItem.size - 8,
                        }}
                      >
                        <span className="whitespace-nowrap">{category.name}</span>
                      </button>
                    );
                  })}
                </div>
              </div>
            )}
          </nav>

          <div
            ref={topSentinelRef}
            aria-hidden
            className="h-px w-full"
          />
          {showPreviousLoading ? (
            <div
              aria-live="polite"
              className="flex min-h-12 animate-[feed-loading-in_120ms_cubic-bezier(0.16,1,0.3,1)] items-center justify-center text-sm text-[var(--explore-subtle)] motion-reduce:animate-none"
            >
              <span className="inline-flex items-center gap-2 rounded-full border border-[var(--explore-border)] bg-[var(--explore-surface-soft)] px-4 py-2 backdrop-blur">
                <RefreshCw className="size-4 motion-safe:animate-spin" />
                {t("feed.loadingMore")}
              </span>
            </div>
          ) : feedQuery.isFetchPreviousPageError ? (
            <div className="flex min-h-12 items-center justify-center">
              <Button
                type="button"
                variant="ghost"
                onClick={() => void fetchPageWithAnchor("previous")}
              >
                {t("feed.retry")}
              </Button>
            </div>
          ) : null}

          <section
            key={`${activeMode}-${activeCategory}-${activeSearch}`}
            aria-label={t("nav.home")}
            className="w-full animate-[feed-content-in_220ms_ease-out] px-2 sm:px-6 lg:px-8"
            onClickCapture={handleFeedClickCapture}
            onPointerCancel={() => {
              feedSwipeStartRef.current = null;
            }}
            onPointerDown={handleFeedPointerStart}
            onPointerUp={handleFeedPointerEnd}
          >
            {isReplacingFeed && posts.length > 0 ? (
              <div className="sticky top-[132px] z-20 mb-3 flex justify-center lg:top-6">
                <span
                  aria-live="polite"
                  className="inline-flex h-9 items-center gap-2 rounded-full border border-[var(--explore-border)] bg-[var(--explore-bottom)] px-4 text-xs font-semibold text-[var(--explore-strong)] shadow-[0_10px_24px_rgba(0,0,0,0.16)] backdrop-blur"
                >
                  <RefreshCw className="size-3.5 motion-safe:animate-spin" />
                  {t("feed.loading")}
                </span>
              </div>
            ) : null}
            {posts.length > 0 ? (
              <TooltipProvider>
                <Masonry
                  items={posts}
                  config={{
                    columns: [2, 3, 4, 5],
                    gap: [8, 18, 24, 32],
                    media: [640, 960, 1280, 1600],
                    useBalancedLayout: false,
                  }}
                  render={(post, index) => (
                    <PostCard
                      key={post.id}
                      post={post}
                      index={index}
                      imagePriority={priorityPostIds.has(String(post.id))}
                      transitionScope="explore"
                      onLike={handleLike}
                      theme={exploreTheme}
                    />
                  )}
                />
              </TooltipProvider>
            ) : isReplacingFeed ? (
              <div
                aria-live="polite"
                className="flex min-h-[42vh] flex-col items-center justify-center rounded-[10px] border border-dashed border-[var(--explore-border)] px-6 text-center"
              >
                <RefreshCw className="size-5 motion-safe:animate-spin text-[var(--explore-muted)]" />
                <p className="mt-3 text-sm font-semibold text-[var(--explore-strong)]">
                  {t("feed.loading")}
                </p>
              </div>
            ) : (
              <div className="flex min-h-[42vh] flex-col items-center justify-center rounded-[10px] border border-dashed border-[var(--explore-border)] px-6 text-center">
                <p className="text-base font-semibold text-[var(--explore-strong)]">{t("feed.emptyTitle")}</p>
                <p className="mt-2 text-sm text-[var(--explore-subtle)]">
                  {t("feed.emptyDescription")}
                </p>
              </div>
            )}
          </section>

          <div
            ref={bottomSentinelRef}
            aria-hidden
            className="h-px w-full"
          />
          <div
            ref={noMoreUnlockSentinelRef}
            aria-hidden
            className="h-px w-full"
          />
          <div
            className="mt-8 flex min-h-12 w-full items-center justify-center px-4 pb-3 text-sm text-[var(--explore-subtle)] sm:px-6 lg:px-8"
          >
            {showNextLoading ? (
              <span
                aria-live="polite"
                className="inline-flex animate-[feed-loading-in_120ms_cubic-bezier(0.16,1,0.3,1)] items-center gap-2 rounded-full border border-[var(--explore-border)] bg-[var(--explore-surface-soft)] px-4 py-2 backdrop-blur motion-reduce:animate-none"
              >
                <RefreshCw className="size-4 motion-safe:animate-spin" />
                {t("feed.loadingMore")}
              </span>
            ) : feedQuery.isFetchNextPageError ? (
              <Button
                type="button"
                variant="ghost"
                onClick={() => void fetchPageWithAnchor("next")}
              >
                {t("feed.retry")}
              </Button>
            ) : !canLoadMore && posts.length > 0 ? (
              <span>{t("feed.noMore")}</span>
            ) : null}
          </div>
        </div>
      </main>

      <div className="fixed bottom-20 right-4 z-30 hidden flex-col gap-3 lg:flex">
        <Button
          size="icon"
          variant="outline"
          aria-label={t("status.refresh")}
          className="size-12 border-[var(--explore-border)] bg-[var(--explore-control)] text-[var(--explore-strong)] shadow-lg hover:bg-[var(--explore-control-hover)]"
          onClick={() => void refreshFeed()}
          disabled={isPending}
        >
          <RefreshCw className={cn("size-5", isPending && "motion-safe:animate-spin")} />
        </Button>
      </div>

      <ExploreMobileBottomNav
        activeMode={activeMode}
        loadMode={loadMode}
        messageBadgeCount={messageBadgeCount}
        warmNavigationTarget={warmNavigationTarget}
      />
      <LoginUnlockDialog
        open={loginUnlockOpen}
        onOpenChange={setLoginUnlockOpen}
      />

      {lightboxIndex >= 0 ? (
        <Lightbox
          open
          close={() => setLightboxIndex(-1)}
          index={lightboxIndex}
          slides={lightboxSlides}
        />
      ) : null}
    </div>
  );
}
