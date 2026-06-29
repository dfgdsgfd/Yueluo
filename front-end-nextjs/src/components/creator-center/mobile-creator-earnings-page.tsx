"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { ArrowLeft, Loader2, RefreshCw, Wallet } from "lucide-react";
import { toast } from "sonner";
import {
  buildCreatorActivities,
  CreatorEarningListItem,
  type ActivityItem,
} from "@/components/creator-center/mobile-creator-center-page";
import { getCreatorEarningsLog, getStoredAccessToken } from "@/lib/api";
import { cn } from "@/lib/utils";

const PAGE_SIZE = 20;

export function MobileCreatorEarningsPage() {
  const router = useRouter();
  const [authToken, setAuthToken] = useState<string | null | undefined>(undefined);
  const [items, setItems] = useState<ActivityItem[]>([]);
  const [page, setPage] = useState(1);
  const [hasNext, setHasNext] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [isLoadingMore, setIsLoadingMore] = useState(false);

  const loadEarnings = useCallback(async (nextPage = 1, options: { append?: boolean; silent?: boolean } = {}) => {
    const token = getStoredAccessToken();
    setAuthToken(token);
    if (!token) {
      setIsLoading(false);
      setIsRefreshing(false);
      setIsLoadingMore(false);
      return;
    }

    if (options.append) {
      setIsLoadingMore(true);
    } else if (options.silent) {
      setIsRefreshing(true);
    } else {
      setIsLoading(true);
    }

    try {
      const payload = await getCreatorEarningsLog({ page: nextPage, limit: PAGE_SIZE });
      const nextItems = buildCreatorActivities(payload.list);
      setItems((current) => (options.append ? [...current, ...nextItems] : nextItems));
      setPage(payload.pagination.page ?? nextPage);
      setHasNext(hasNextPage(payload.pagination, payload.list.length));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "收益记录加载失败");
    } finally {
      setIsLoading(false);
      setIsRefreshing(false);
      setIsLoadingMore(false);
    }
  }, []);

  useEffect(() => {
    queueMicrotask(() => {
      void loadEarnings();
    });
  }, [loadEarnings]);

  const handleBack = useCallback(() => {
    if (window.history.length > 1) {
      router.back();
      return;
    }
    router.push("/creator-center");
  }, [router]);

  if (authToken === null) {
    return (
      <main className="flex min-h-dvh items-center justify-center bg-[#fbfbff] px-4 text-[#22222a]">
        <section className="w-full max-w-[360px] rounded-[8px] bg-white px-5 py-6 text-center shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
          <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-[#f4efff] text-[#765eda]">
            <Wallet className="size-6" />
          </div>
          <h1 className="mt-4 text-[18px] font-black">登录后查看收益</h1>
          <p className="mt-2 text-sm font-bold leading-6 text-[#9a96a5]">收益记录需要使用当前账号身份。</p>
          <div className="mt-5 grid grid-cols-2 gap-2">
            <Link href="/login" className="flex h-10 items-center justify-center rounded-full bg-[#8069dd] text-sm font-black text-white">
              去登录
            </Link>
            <Link href="/creator-center" className="flex h-10 items-center justify-center rounded-full bg-[#f3f0ff] text-sm font-black text-[#765eda]">
              返回
            </Link>
          </div>
        </section>
      </main>
    );
  }

  return (
    <main className="min-h-dvh bg-[#fbfbff] text-[#24242c]">
      <div className="mx-auto min-h-dvh w-full max-w-[430px] bg-[#fbfbff] pb-6 shadow-[0_0_38px_rgba(112,94,173,0.08)]">
        <header className="sticky top-0 z-30 bg-[#fbfbff]/92 px-4 pb-3 pt-[calc(0.75rem+env(safe-area-inset-top))] backdrop-blur">
          <div className="grid h-11 grid-cols-[40px_minmax(0,1fr)_40px] items-center min-[390px]:grid-cols-[44px_minmax(0,1fr)_44px]">
            <button
              type="button"
              aria-label="返回上一页"
              onClick={handleBack}
              className="flex size-10 items-center justify-center rounded-full text-[#15151b] active:bg-[#f0eef8] min-[390px]:size-11"
            >
              <ArrowLeft className="size-6" strokeWidth={2.4} />
            </button>
            <h1 className="min-w-0 truncate text-center text-[21px] font-black tracking-normal text-[#16161d]">
              近期收益
            </h1>
            <button
              type="button"
              aria-label="刷新近期收益"
              onClick={() => void loadEarnings(1, { silent: true })}
              className="flex size-10 items-center justify-center justify-self-end rounded-full text-[#8b7adf] active:bg-[#f0eef8] min-[390px]:size-11"
            >
              <RefreshCw className={cn("size-5.5", isRefreshing && "animate-spin")} strokeWidth={2.4} />
            </button>
          </div>
        </header>

        {isLoading ? (
          <div className="flex min-h-[520px] items-center justify-center gap-2 text-sm font-bold text-[#9a96a5]">
            <Loader2 className="size-4 animate-spin" />
            正在加载收益记录
          </div>
        ) : (
          <section className="mx-4 rounded-[8px] bg-white px-4 py-4 shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
            {items.length ? (
              <>
                <div className="divide-y divide-[#f0eef7]">
                  {items.map((item) => (
                    <CreatorEarningListItem key={item.id} item={item} />
                  ))}
                </div>
                {hasNext ? (
                  <button
                    type="button"
                    onClick={() => void loadEarnings(page + 1, { append: true })}
                    disabled={isLoadingMore}
                    className="mt-4 flex h-11 w-full items-center justify-center gap-2 rounded-full bg-[#f3f0ff] text-[13px] font-black text-[#765eda] disabled:opacity-60"
                  >
                    {isLoadingMore ? <Loader2 className="size-4 animate-spin" /> : null}
                    加载更多
                  </button>
                ) : null}
              </>
            ) : (
              <div className="flex min-h-[180px] items-center justify-center text-[13px] font-bold text-[#9a96a5]">
                暂无真实收益记录
              </div>
            )}
          </section>
        )}
      </div>
    </main>
  );
}

function hasNextPage(
  pagination: { hasNextPage?: boolean; limit?: number; page?: number; pageSize?: number; pages?: number; total?: number; totalPages?: number },
  listLength: number,
) {
  if (typeof pagination.hasNextPage === "boolean") {
    return pagination.hasNextPage;
  }
  const page = pagination.page ?? 1;
  const totalPages = pagination.totalPages ?? pagination.pages;
  if (typeof totalPages === "number") {
    return page < totalPages;
  }
  const total = pagination.total;
  const limit = pagination.limit ?? pagination.pageSize ?? PAGE_SIZE;
  if (typeof total === "number" && limit > 0) {
    return page * limit < total;
  }
  return listLength >= PAGE_SIZE;
}
