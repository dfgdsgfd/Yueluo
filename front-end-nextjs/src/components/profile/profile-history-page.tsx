"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Masonry } from "react-plock";
import { ArrowLeft } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { TooltipProvider } from "@/components/ui/tooltip";
import { PostCard } from "@/components/feed/post-card";
import {
  getUserHistoryPosts,
  toggleLike,
} from "@/lib/api";
import type { FeedPost } from "@/lib/types";

type HistoryCacheEntry = {
  posts: FeedPost[];
  updatedAt: number;
};

const HISTORY_CACHE_TTL_MS = 2 * 60 * 1000;
let historyCache: HistoryCacheEntry | null = null;
let historyRequest: Promise<FeedPost[]> | null = null;

export function ProfileHistoryDataPage() {
  const t = useTranslations();
  const cachedPosts = getCachedHistoryPosts();
  const [posts, setPosts] = useState<FeedPost[]>(() => cachedPosts ?? []);
  const [status, setStatus] = useState<"loading" | "loaded" | "error">(
    cachedPosts ? "loaded" : "loading",
  );

  useEffect(() => {
    let mounted = true;

    loadCachedHistoryPosts()
      .then((nextPosts) => {
        if (!mounted) {
          return;
        }

        setPosts(nextPosts);
        setStatus("loaded");
      })
      .catch((error) => {
        if (!mounted) {
          return;
        }

        setStatus("error");
        toast.error(error instanceof Error ? error.message : t("feed.loadFailed"));
      });

    return () => {
      mounted = false;
    };
  }, [t]);

  function updatePostState(postId: FeedPost["id"], updater: (post: FeedPost) => FeedPost) {
    setPosts((currentPosts) => {
      const nextPosts = currentPosts.map((item) =>
        item.id === postId ? updater(item) : item,
      );

      if (historyCache) {
        historyCache = {
          ...historyCache,
          posts: nextPosts,
        };
      }

      return nextPosts;
    });
  }

  async function handleLike(post: FeedPost) {
    const nextLiked = !post.liked;
    updatePostState(post.id, (item) => ({
      ...item,
      liked: nextLiked,
      like_count: Math.max(0, item.like_count + (nextLiked ? 1 : -1)),
    }));

    try {
      const result = await toggleLike(post.id);
      updatePostState(post.id, (item) => ({
        ...item,
        liked: result.liked,
        like_count: Math.max(
          0,
          item.like_count + (result.liked === nextLiked ? 0 : result.liked ? 1 : -1),
        ),
      }));
    } catch (error) {
      updatePostState(post.id, (item) => ({
        ...item,
        liked: post.liked,
        like_count: post.like_count,
      }));
      toast.error(error instanceof Error ? error.message : t("feed.loadFailed"));
    }
  }

  const loading = status === "loading" && posts.length === 0;
  const title = t("profile.quickActions.history");

  return (
    <div className="theme-adaptive min-h-screen bg-[#121212] text-[#e0e0e0]">
      <header className="fixed inset-x-0 top-0 z-40 h-14 border-b border-white/[0.07] bg-[#121212]/96 backdrop-blur lg:hidden">
        <div className="flex h-full items-center gap-2 px-3">
          <Button
            asChild
            variant="ghost"
            size="icon"
            aria-label={t("nav.home")}
            className="size-10 text-white hover:bg-white/[0.06]"
          >
            <Link href="/">
              <ArrowLeft className="size-5" />
            </Link>
          </Button>
          <h1 className="min-w-0 flex-1 truncate text-[15px] font-semibold text-white">
            {title}
          </h1>
        </div>
      </header>

      <main className="mx-auto min-h-screen w-full max-w-[1120px] px-4 pb-24 pt-[72px] sm:px-6 lg:px-8 lg:pb-14 lg:pt-10">
        <div className="mb-5 hidden h-10 items-center gap-3 lg:flex">
          <Button
            asChild
            variant="ghost"
            className="h-10 px-3 text-white/64 hover:bg-white/[0.06] hover:text-white"
          >
            <Link href="/">
              <ArrowLeft className="size-4" />
              {t("nav.home")}
            </Link>
          </Button>
          <h1 className="text-xl font-bold text-white">{title}</h1>
        </div>

        {loading ? (
          <ProfileHistorySkeleton />
        ) : posts.length > 0 ? (
          <TooltipProvider>
            <Masonry
              items={posts}
              config={{
                columns: [2, 3, 4, 5],
                gap: [10, 18, 22, 26],
                media: [640, 920, 1200, 1500],
                useBalancedLayout: true,
              }}
              render={(post, index) => (
                <PostCard
                  key={`${post.id}-${index}`}
                  post={post}
                  index={index}
                  transitionScope="profile-history"
                  onLike={handleLike}
                />
              )}
            />
          </TooltipProvider>
        ) : (
          <div className="flex min-h-[42vh] flex-col items-center justify-center rounded-2xl border border-dashed border-white/10 px-6 text-center">
            <p className="text-base font-semibold text-white">
              {t("profile.emptyHistoryTitle")}
            </p>
            <p className="mt-2 text-sm text-white/45">
              {status === "error"
                ? t("feed.emptyDescription")
                : t("profile.emptyHistoryDescription")}
            </p>
          </div>
        )}
      </main>

    </div>
  );
}

function ProfileHistorySkeleton() {
  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
      {Array.from({ length: 8 }).map((_, index) => (
        <div key={index} className="overflow-hidden rounded-lg bg-white/[0.045]">
          <div className="aspect-[3/4] animate-pulse bg-white/[0.08]" />
          <div className="space-y-2 p-2">
            <div className="h-3 w-4/5 animate-pulse rounded bg-white/[0.08]" />
            <div className="h-3 w-1/2 animate-pulse rounded bg-white/[0.08]" />
          </div>
        </div>
      ))}
    </div>
  );
}

function getCachedHistoryPosts() {
  if (!historyCache) {
    return null;
  }

  if (Date.now() - historyCache.updatedAt > HISTORY_CACHE_TTL_MS) {
    historyCache = null;
    return null;
  }

  return historyCache.posts;
}

function loadCachedHistoryPosts() {
  const cachedPosts = getCachedHistoryPosts();
  if (cachedPosts) {
    return Promise.resolve(cachedPosts);
  }

  if (historyRequest) {
    return historyRequest;
  }

  historyRequest = getUserHistoryPosts()
    .then((posts) => {
      historyCache = {
        posts,
        updatedAt: Date.now(),
      };

      return posts;
    })
    .finally(() => {
      historyRequest = null;
    });

  return historyRequest;
}
