import {
  getFeedPage,
  getNoteCategories,
  getUserToolbarItems,
  getWithdrawWallet,
  hasNextFeedPage,
  logout,
  searchFeed,
} from "@/lib/api";
import type {
  Category,
  FeedMode,
  FeedPayload,
  InitialFeedData,
  WithdrawWalletPayload,
} from "@/lib/types";
import { normalizeSiteProfile } from "@/lib/seo";
import {
  keepPreviousData,
  useInfiniteQuery,
  useQueryClient,
  type InfiniteData,
} from "@tanstack/react-query";
import { useVirtualizer } from "@tanstack/react-virtual";
import { useLocale, useTranslations } from "next-intl";
import { useRouter } from "next/navigation";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type FormEvent,
  type MouseEvent,
  type PointerEvent,
} from "react";
import { toast } from "sonner";
import { getPostCover } from "../feed-utils";
import { FEED_QUERY_KEY } from "../post-feed-state";
import {
  ExploreTheme,
  ExploreThemePreference,
  FEED_BOTTOM_SENTINEL_ROOT_MARGIN,
  FEED_MAX_PAGES,
  FEED_TOP_SENTINEL_ROOT_MARGIN,
  FollowingSort,
  desktopNavItems,
  estimateCategoryItemSize,
  getExploreThemeVars,
} from "./explore-config";
import {
  getVisibleFeedAnchor,
  restoreFeedAnchor,
  useFeedSentinel,
  useGracefulLoading,
} from "./explore-hooks";
import { getSwipeCategoryOffset } from "./explore-swipe-utils";
import {
  normalizeCategories,
  type KnownCategoryNameKey,
} from "./explore-widgets";
import { useExploreClientEnvironment } from "./use-explore-client-environment";
import { useExploreFeedActions } from "./use-explore-feed-actions";
import { useLoginUnlockGate } from "./use-login-unlock-gate";
import { useMessageBadgeSync } from "./use-message-badge-sync";

export function useExploreFeedController({
  initialData,
}: {
  initialData: InitialFeedData;
}) {
  const locale = useLocale();
  const t = useTranslations();
  const router = useRouter();
  const queryClient = useQueryClient();
  const [activeMode, setActiveMode] = useState<FeedMode>("recommended");
  const [activeCategory, setActiveCategory] = useState<number | "all">("all");
  const [categoryItems, setCategoryItems] = useState<Category[]>(
    initialData.categories,
  );
  const [toolbarItems, setToolbarItems] = useState(
    initialData.toolbarItems ?? [],
  );
  const siteProfile = useMemo(
    () => normalizeSiteProfile(initialData.siteProfile),
    [initialData.siteProfile],
  );
  const [followingSort, setFollowingSort] = useState<FollowingSort>("time");
  const [lightboxIndex, setLightboxIndex] = useState(-1);
  const [searchInput, setSearchInput] = useState("");
  const [activeSearch, setActiveSearch] = useState("");
  const [mobileSearchOpen, setMobileSearchOpen] = useState(false);
  const [mobileMoreOpen, setMobileMoreOpen] = useState(false);
  const [hasClientAccessToken, setHasClientAccessToken] = useState(
    Boolean(initialData.viewer),
  );
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const [messageBadgeCount, setMessageBadgeCount] = useState(0);
  const [walletBalance, setWalletBalance] =
    useState<WithdrawWalletPayload | null>(null);
  const [isWalletBalanceLoading, setIsWalletBalanceLoading] = useState(false);
  const [walletBalanceError, setWalletBalanceError] = useState<string | null>(
    null,
  );
  const [exploreThemePreference, setExploreThemePreference] =
    useState<ExploreThemePreference>("system");
  const [exploreTheme, setExploreTheme] = useState<ExploreTheme>("dark");
  const [themeSettingsOpen, setThemeSettingsOpen] = useState(false);
  const [hasLoadedExploreTheme, setHasLoadedExploreTheme] = useState(false);
  const categorySwipeStartRef = useRef<{ x: number; y: number } | null>(null);
  const feedSwipeStartRef = useRef<{ x: number; y: number } | null>(null);
  const feedSwipeSuppressClickRef = useRef(false);
  const prefetchedNavigationRoutesRef = useRef(new Set<string>());
  const prefetchedNavigationDataRef = useRef(new Set<string>());
  const searchActive = activeSearch.trim().length > 0;
  const feedQueryKey = useMemo(
    () =>
      [
        FEED_QUERY_KEY,
        activeMode,
        activeCategory,
        activeMode === "following" ? followingSort : null,
        searchActive ? activeSearch : "",
      ] as const,
    [activeCategory, activeMode, activeSearch, followingSort, searchActive],
  );
  const isInitialFeedQuery =
    activeMode === "recommended" && activeCategory === "all" && !searchActive;
  const initialInfiniteData = useMemo<
    InfiniteData<FeedPayload, number> | undefined
  >(
    () =>
      isInitialFeedQuery && initialData.posts.length > 0
        ? {
            pages: [
              {
                posts: initialData.posts,
                pagination: initialData.pagination,
              },
            ],
            pageParams: [initialData.pagination.page],
          }
        : undefined,
    [initialData.pagination, initialData.posts, isInitialFeedQuery],
  );
  const feedQuery = useInfiniteQuery<
    FeedPayload,
    Error,
    InfiniteData<FeedPayload, number>,
    typeof feedQueryKey,
    number
  >({
    enabled:
      (activeMode !== "following" ||
        initialData.posts.length > 0 ||
        hasClientAccessToken ||
        searchActive),
    getNextPageParam: (lastPage) =>
      hasNextFeedPage(lastPage.pagination)
        ? lastPage.pagination.page + 1
        : undefined,
    getPreviousPageParam: (firstPage) =>
      firstPage.pagination.page > 1 ? firstPage.pagination.page - 1 : undefined,
    initialData: initialInfiniteData,
    initialPageParam: 1,
    maxPages: FEED_MAX_PAGES,
    placeholderData: keepPreviousData,
    queryFn: ({ pageParam, signal }) => {
      const limit = activeMode === "following" ? 20 : 24;
      if (searchActive) {
        return searchFeed(
          { keyword: activeSearch, page: pageParam, limit },
          { signal },
        );
      }
      return getFeedPage(
        activeMode,
        { signal },
        {
          category_id:
            activeMode === "following" || activeCategory === "all"
              ? undefined
              : activeCategory,
          page: pageParam,
          limit,
          sort: activeMode === "following" ? followingSort : undefined,
        },
      );
    },
    queryKey: feedQueryKey,
    refetchOnMount: false,
  });
  const posts = useMemo(() => {
    const seen = new Set<string>();
    return (feedQuery.data?.pages ?? []).flatMap((page) =>
      page.posts.filter((post) => {
        const id = String(post.id);
        if (seen.has(id)) {
          return false;
        }
        seen.add(id);
        return true;
      }),
    );
  }, [feedQuery.data?.pages]);
  const isReplacingFeed =
    feedQuery.isFetching &&
    !feedQuery.isFetchingNextPage &&
    !feedQuery.isFetchingPreviousPage;
  const isPending = isReplacingFeed;
  const showNextLoading = useGracefulLoading(feedQuery.isFetchingNextPage);
  const showPreviousLoading = useGracefulLoading(
    feedQuery.isFetchingPreviousPage,
  );
  const knownCategoryNames = useMemo<Record<KnownCategoryNameKey, string>>(
    () => ({
      beauty: t("category.names.beauty"),
      fashion: t("category.names.fashion"),
      food: t("category.names.food"),
      gaming: t("category.names.gaming"),
      interest: t("category.names.interest"),
      knowledge: t("category.names.knowledge"),
      lifestyle: t("category.names.lifestyle"),
      music: t("category.names.music"),
      novel: t("category.names.novel"),
      photography: t("category.names.photography"),
      pictureSet: t("category.names.pictureSet"),
      ropeBondage: t("category.names.ropeBondage"),
      shackles: t("category.names.shackles"),
      sports: t("category.names.sports"),
      technology: t("category.names.technology"),
      travel: t("category.names.travel"),
      video: t("category.names.video"),
    }),
    [t],
  );
  const categories = useMemo(
    () =>
      normalizeCategories(
        categoryItems,
        t("category.all"),
        locale,
        knownCategoryNames,
      ),
    [categoryItems, knownCategoryNames, locale, t],
  );
  const lightboxSlides = useMemo(
    () =>
      posts
        .map((post) => getPostCover(post))
        .filter((src): src is string => Boolean(src))
        .map((src) => ({ src })),
    [posts],
  );
  const priorityPostIds = useMemo(
    () => new Set(posts.slice(0, 4).map((post) => String(post.id))),
    [posts],
  );
  // eslint-disable-next-line react-hooks/incompatible-library -- TanStack Virtual is required for the category strip.
  const categoryVirtualizer = useVirtualizer({
    count: categories.length,
    estimateSize: (index) => estimateCategoryItemSize(categories[index].name),
    horizontal: true,
    getScrollElement: () => document.getElementById("category-strip"),
    overscan: 4,
  });
  const canLoadMore = feedQuery.hasNextPage;
  const exploreThemeVars = useMemo(
    () => getExploreThemeVars(exploreTheme),
    [exploreTheme],
  );
  const isLightTheme = exploreTheme === "light";
  const visibleDesktopNavItems = desktopNavItems;
  const warmNavigationTarget = useCallback(
    (key: string, href: string | null) => {
      if (
        href?.startsWith("/") &&
        !prefetchedNavigationRoutesRef.current.has(href)
      ) {
        prefetchedNavigationRoutesRef.current.add(href);
        router.prefetch(href);
      }
      const dataKey = key === "create" ? "publish-mobile" : key;
      if (prefetchedNavigationDataRef.current.has(dataKey)) {
        return;
      }
      if (
        !hasClientAccessToken &&
        (dataKey === "messages" ||
          dataKey === "publish" ||
          dataKey === "publish-mobile")
      ) {
        return;
      }
      prefetchedNavigationDataRef.current.add(dataKey);
      if (dataKey === "messages") {
        void import("@/lib/im-cache")
          .then((mod) => mod.prefetchMessagesPageData())
          .catch(() => undefined);
        return;
      }
      if (dataKey === "publish") {
        void import("@/components/publish/publish-workbench")
          .then((mod) => mod.prefetchPublishWorkbenchData())
          .catch(() => undefined);
        return;
      }
      if (dataKey === "publish-mobile") {
        void import("@/components/publish/mobile-publish-page")
          .then((mod) => mod.prefetchMobilePublishData())
          .catch(() => undefined);
        return;
      }
    },
    [hasClientAccessToken, router],
  );
  useExploreClientEnvironment({
    exploreTheme,
    exploreThemePreference,
    hasLoadedExploreTheme,
    setExploreTheme,
    setExploreThemePreference,
    setHasClientAccessToken,
    setHasLoadedExploreTheme,
  });
  useEffect(() => {
    if (!mobileMoreOpen) {
      return;
    }
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setMobileMoreOpen(false);
        setThemeSettingsOpen(false);
      }
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [mobileMoreOpen]);
  useEffect(() => {
    setCategoryItems(initialData.categories);
  }, [initialData.categories]);
  useEffect(() => {
    setToolbarItems(initialData.toolbarItems ?? []);
  }, [initialData.toolbarItems]);
  useEffect(() => {
    let cancelled = false;
    getUserToolbarItems()
      .then((items) => {
        if (!cancelled) {
          setToolbarItems(items);
        }
      })
      .catch(() => {
        // Keep the server-rendered toolbar items when the refresh request is unavailable.
      });
    return () => {
      cancelled = true;
    };
  }, []);
  useEffect(() => {
    if (categoryItems.length > 0 || !hasClientAccessToken) {
      return;
    }
    let cancelled = false;
    getNoteCategories()
      .then((nextCategories) => {
        if (!cancelled && nextCategories.length > 0) {
          setCategoryItems(nextCategories);
        }
      })
      .catch(() => {
        // Keep the "all" tab available when the authenticated category request is not ready yet.
      });
    return () => {
      cancelled = true;
    };
  }, [categoryItems.length, hasClientAccessToken]);
  useMessageBadgeSync({
    enabled: hasClientAccessToken,
    setMessageBadgeCount,
  });
  useEffect(() => {
    if (!hasClientAccessToken || !mobileMoreOpen) {
      setWalletBalance(null);
      setWalletBalanceError(null);
      setIsWalletBalanceLoading(false);
      return;
    }
    let cancelled = false;
    setIsWalletBalanceLoading(true);
    setWalletBalanceError(null);
    getWithdrawWallet()
      .then((payload) => {
        if (!cancelled) {
          setWalletBalance(payload);
        }
      })
      .catch((error) => {
        if (!cancelled) {
          setWalletBalance(null);
          const message = error instanceof Error ? error.message : "";
          setWalletBalanceError(
            message || t("moreMenu.wallet.error"),
          );
        }
      })
      .finally(() => {
        if (!cancelled) {
          setIsWalletBalanceLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [hasClientAccessToken, mobileMoreOpen, t]);
  useEffect(() => {
    if (!hasClientAccessToken) {
      return;
    }
    const idleCallback: typeof window.requestIdleCallback =
      window.requestIdleCallback ??
      ((callback) =>
        window.setTimeout(
          () => callback({ didTimeout: false, timeRemaining: () => 0 }),
          1,
        ));
    const cancelIdleCallback: typeof window.cancelIdleCallback =
      window.cancelIdleCallback ?? ((id) => window.clearTimeout(id));
    const idleId = idleCallback(() => {
      warmNavigationTarget("messages", "/messages");
      warmNavigationTarget("publish", "/publish");
      warmNavigationTarget("create", "/publish/mobile");
    });
    return () => cancelIdleCallback(idleId);
  }, [hasClientAccessToken, warmNavigationTarget]);
  useEffect(() => {
    const error = feedQuery.error;
    if (!error || error.name === "AbortError") {
      return;
    }
    toast.error(error.message || t("feed.loadFailed"));
  }, [feedQuery.error, t]);
  function loadMode(
    mode: FeedMode,
    categoryId: number | "all" = "all",
    sort?: FollowingSort,
  ) {
    void queryClient.cancelQueries({ queryKey: [FEED_QUERY_KEY] });
    const nextFollowingSort =
      mode === "following" ? (sort ?? followingSort) : followingSort;
    setActiveSearch("");
    setActiveMode(mode);
    setActiveCategory(categoryId);
    if (mode === "following") {
      setFollowingSort(nextFollowingSort);
    }
  }
  function loadCategory(categoryId: number | "all") {
    loadMode(activeMode, categoryId);
  }
  function scrollCategoryIntoView(category: Category) {
    const strip = document.getElementById("category-strip");
    const categoryButton = strip?.querySelector<HTMLElement>(
      `[data-category-id="${category.id}"]`,
    );
    categoryButton?.scrollIntoView({
      behavior: "smooth",
      block: "nearest",
      inline: "center",
    });
  }
  function getActiveCategoryIndex() {
    const index = categories.findIndex((category) => {
      const isAll = category.id === -1;
      return (
        (isAll && activeCategory === "all") || activeCategory === category.id
      );
    });
    return index >= 0 ? index : 0;
  }
  function switchCategoryByOffset(offset: number) {
    if (categories.length <= 1) {
      return;
    }
    const currentIndex = getActiveCategoryIndex();
    const nextIndex = Math.min(
      Math.max(currentIndex + offset, 0),
      categories.length - 1,
    );
    if (nextIndex === currentIndex) {
      return;
    }
    const nextCategory = categories[nextIndex];
    loadCategory(nextCategory.id === -1 ? "all" : nextCategory.id);
    window.requestAnimationFrame(() => scrollCategoryIntoView(nextCategory));
  }
  function handleCategoryPointerStart(event: PointerEvent<HTMLDivElement>) {
    categorySwipeStartRef.current = { x: event.clientX, y: event.clientY };
  }
  function handleCategoryPointerEnd(event: PointerEvent<HTMLDivElement>) {
    const start = categorySwipeStartRef.current;
    categorySwipeStartRef.current = null;
    if (!start) {
      return;
    }
    const offset = getSwipeCategoryOffset(start, { x: event.clientX, y: event.clientY });
    if (offset !== null) {
      switchCategoryByOffset(offset);
    }
  }
  function handleFeedPointerStart(event: PointerEvent<HTMLElement>) {
    if (activeMode === "following" || event.pointerType === "mouse") {
      return;
    }
    feedSwipeStartRef.current = { x: event.clientX, y: event.clientY };
  }
  function handleFeedPointerEnd(event: PointerEvent<HTMLElement>) {
    const start = feedSwipeStartRef.current;
    feedSwipeStartRef.current = null;
    if (!start || activeMode === "following") {
      return;
    }
    const offset = getSwipeCategoryOffset(start, { x: event.clientX, y: event.clientY });
    if (offset === null) {
      return;
    }
    feedSwipeSuppressClickRef.current = true;
    switchCategoryByOffset(offset);
    window.setTimeout(() => {
      feedSwipeSuppressClickRef.current = false;
    }, 0);
  }
  function handleFeedClickCapture(event: MouseEvent<HTMLElement>) {
    if (!feedSwipeSuppressClickRef.current) {
      return;
    }
    event.preventDefault();
    event.stopPropagation();
    feedSwipeSuppressClickRef.current = false;
  }
  function runSearch(rawKeyword: string) {
    const keyword = rawKeyword.trim();
    if (!keyword) {
      loadMode(activeMode, activeCategory);
      return;
    }
    void queryClient.cancelQueries({ queryKey: [FEED_QUERY_KEY] });
    setActiveSearch(keyword);
    setActiveCategory("all");
  }
  function handleSearchSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    runSearch(searchInput);
  }
  function clearSearch() {
    setSearchInput("");
    loadMode(activeMode, "all");
  }
  async function handleLogout() {
    if (isLoggingOut) {
      return;
    }
    setIsLoggingOut(true);
    try {
      await logout();
    } catch (error) {
      console.warn("Logout request failed after local session cleanup.", error);
    } finally {
      setHasClientAccessToken(false);
      setMobileMoreOpen(false);
      setThemeSettingsOpen(false);
      setIsLoggingOut(false);
      toast.success(t("nav.logoutSuccess"));
      router.refresh();
    }
  }
  async function fetchPageWithAnchor(direction: "next" | "previous") {
    if (
      feedQuery.isPlaceholderData ||
      isReplacingFeed ||
      feedQuery.isFetchingNextPage ||
      feedQuery.isFetchingPreviousPage
    ) {
      return;
    }
    const anchor = getVisibleFeedAnchor();
    try {
      if (direction === "next") {
        if (!feedQuery.hasNextPage) {
          return;
        }
        await feedQuery.fetchNextPage({ cancelRefetch: false });
      } else {
        if (!feedQuery.hasPreviousPage) {
          return;
        }
        await feedQuery.fetchPreviousPage({ cancelRefetch: false });
      }
    } catch {
      // React Query exposes the directional error state and retry action in the feed UI.
    }
    restoreFeedAnchor(anchor);
  }
  async function refreshFeed() {
    await queryClient.cancelQueries({ queryKey: feedQueryKey, exact: true });
    queryClient.setQueryData<InfiniteData<FeedPayload, number>>(
      feedQueryKey,
      (current) =>
        current?.pages.length
          ? {
              pages: [current.pages[0]],
              pageParams: [current.pageParams[0] ?? 1],
            }
          : current,
    );
    try {
      await feedQuery.refetch();
    } catch {
      // The feed keeps its existing page while the shared query error UI handles retry.
    }
  }
  const topSentinelRef = useFeedSentinel({
    enabled:
      !feedQuery.isPlaceholderData &&
      posts.length > 0 &&
      feedQuery.hasPreviousPage,
    onEnter: () => void fetchPageWithAnchor("previous"),
    resetKey: feedQueryKey.join(":"),
    rootMargin: FEED_TOP_SENTINEL_ROOT_MARGIN,
  });
  const bottomSentinelRef = useFeedSentinel({
    enabled:
      !feedQuery.isPlaceholderData &&
      posts.length > 0 &&
      feedQuery.hasNextPage,
    onEnter: () => void fetchPageWithAnchor("next"),
    resetKey: feedQueryKey.join(":"),
    rootMargin: FEED_BOTTOM_SENTINEL_ROOT_MARGIN,
  });
  const {
    loginUnlockOpen,
    noMoreUnlockSentinelRef,
    openLoginUnlock,
    requireLogin,
    setLoginUnlockOpen,
  } = useLoginUnlockGate({
    feedQueryKey,
    hasClientAccessToken,
    hasNextPage: feedQuery.hasNextPage,
    isFeedFetching: feedQuery.isFetching,
    isPlaceholderData: feedQuery.isPlaceholderData,
    postsLength: posts.length,
  });
  const { handleLike } = useExploreFeedActions({
    openLoginUnlock,
    queryClient,
    requireLogin,
    t,
  });
  return {
    activeCategory,
    activeMode,
    activeSearch,
    bottomSentinelRef,
    canLoadMore,
    categories,
    categorySwipeStartRef,
    categoryVirtualizer,
    clearSearch,
    exploreTheme,
    exploreThemePreference,
    exploreThemeVars,
    feedQuery,
    feedSwipeStartRef,
    fetchPageWithAnchor,
    followingSort,
    handleCategoryPointerEnd,
    handleCategoryPointerStart,
    handleFeedClickCapture,
    handleFeedPointerEnd,
    handleFeedPointerStart,
    handleLike,
    handleLogout,
    handleSearchSubmit,
    hasClientAccessToken,
    isLightTheme,
    isLoggingOut,
    isPending,
    isReplacingFeed,
    isWalletBalanceLoading,
    lightboxIndex,
    lightboxSlides,
    loadCategory,
    loadMode,
    loginUnlockOpen,
    messageBadgeCount,
    mobileMoreOpen,
    mobileSearchOpen,
    noMoreUnlockSentinelRef,
    posts,
    priorityPostIds,
    refreshFeed,
    scrollCategoryIntoView,
    searchActive,
    searchInput,
    setExploreTheme,
    setExploreThemePreference,
    setLightboxIndex,
    setLoginUnlockOpen,
    setMobileMoreOpen,
    setMobileSearchOpen,
    setSearchInput,
    setThemeSettingsOpen,
    showNextLoading,
    showPreviousLoading,
    siteProfile,
    t,
    themeSettingsOpen,
    toolbarItems,
    topSentinelRef,
    visibleDesktopNavItems,
    walletBalance,
    walletBalanceError,
    warmNavigationTarget,
  };
}
