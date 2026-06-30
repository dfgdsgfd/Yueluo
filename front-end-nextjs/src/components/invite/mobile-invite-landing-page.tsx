"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { CheckCircle2, Gift, Loader2, UserPlus } from "lucide-react";
import { getInviteInfo, recordInviteClick } from "@/lib/api";
import type { InviteInfoPayload } from "@/lib/types";

export function MobileInviteLandingPage({ code }: { code: string }) {
  const [info, setInfo] = useState<InviteInfoPayload | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);

  useEffect(() => {
    let mounted = true;

    Promise.allSettled([getInviteInfo(code), recordInviteClick(code)])
      .then(([infoResult]) => {
        if (!mounted) {
          return;
        }
        if (infoResult.status === "fulfilled") {
          setInfo(infoResult.value);
        } else {
          setNotFound(true);
        }
      })
      .finally(() => {
        if (mounted) {
          setIsLoading(false);
        }
      });

    return () => {
      mounted = false;
    };
  }, [code]);

  return (
    <main className="min-h-dvh bg-[#fbfbff] text-[#22222a]">
      <div className="mx-auto flex min-h-dvh w-full max-w-[430px] flex-col px-4 pb-6 pt-[calc(1.25rem+env(safe-area-inset-top))]">
        <section className="relative mt-8 overflow-hidden rounded-[8px] bg-[#7569df] px-5 py-6 text-white shadow-[0_16px_38px_rgba(110,88,210,0.24)]">
          <div className="absolute -right-8 -top-16 size-40 rounded-full bg-white/16" />
          <div className="absolute bottom-0 right-10 size-20 rounded-full bg-[#c9a7ff]/24" />
          <div className="relative z-10">
            <div className="flex size-12 items-center justify-center rounded-full bg-white/18">
              {isLoading ? <Loader2 className="size-6 animate-spin" /> : <Gift className="size-6" />}
            </div>
            <h1 className="mt-5 text-[28px] font-black leading-tight">邀请注册</h1>
            <p className="mt-3 text-[14px] font-bold leading-6 text-white/76">
              {isLoading
                ? "正在确认邀请信息..."
                : notFound
                  ? "这个邀请码暂不可用。"
                  : `${info?.nickname || "好友"} 邀请你加入 YueM。`}
            </p>
            <div className="mt-5 rounded-[8px] bg-white/16 px-3 py-3">
              <p className="text-[12px] font-bold text-white/68">邀请码</p>
              <p className="mt-1 text-[24px] font-black tracking-[0.08em]">{code}</p>
            </div>
          </div>
        </section>

        <section className="mt-4 rounded-[8px] bg-white px-4 py-4 shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
          <div className="flex items-center gap-3">
            <span className="flex size-10 shrink-0 items-center justify-center rounded-full bg-[#f4efff] text-[#765eda]">
              {notFound ? <CheckCircle2 className="size-5" /> : <UserPlus className="size-5" />}
            </span>
            <div className="min-w-0">
              <h2 className="truncate text-[16px] font-black text-[#1f1f27]">
                {notFound ? "邀请码不可用" : "完成注册后即可加入"}
              </h2>
              <p className="mt-1 text-[12px] font-bold text-[#9a96a5]">
                注册或登录后，即可继续浏览内容。
              </p>
            </div>
          </div>
          <Link
            href={`/login?invite_code=${encodeURIComponent(code)}`}
            className="mt-5 flex h-11 items-center justify-center rounded-full bg-[#8069dd] text-[14px] font-black text-white"
          >
            去注册 / 登录
          </Link>
        </section>
      </div>
    </main>
  );
}
