"use client";

import { useEffect, useState } from "react";
import { Loader2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import {
  getCurrentAdmin,
  getStoredAdminAccessToken,
  getStoredAdminUser,
  logoutAdmin,
} from "@/lib/api";
import type { AdminUser } from "@/lib/types";
import { AdminAuthenticatedShell } from "./admin-authenticated-shell";
import { AdminPublicShell } from "./admin-public-shell";

export function AdminConsolePage({
  backendApiEntryPath,
}: {
  backendApiEntryPath: string;
}) {
  const adminT = useTranslations("adminPortal");
  const router = useRouter();
  const [token] = useState<string | null>(() => getStoredAdminAccessToken());
  const [admin, setAdmin] = useState<AdminUser | null>(() => getStoredAdminUser());

  useEffect(() => {
    let cancelled = false;
    if (!token) {
      router.replace("/admin");
      return;
    }

    void getCurrentAdmin(token)
      .then((currentAdmin) => {
        if (!cancelled) {
          setAdmin(currentAdmin);
        }
      })
      .catch(() => {
        logoutAdmin();
        router.replace("/admin");
      });

    return () => {
      cancelled = true;
    };
  }, [router, token]);

  if (!token || !admin) {
    return (
      <AdminPublicShell>
        <AdminPanelLoading label={adminT("loading")} />
      </AdminPublicShell>
    );
  }

  return (
    <AdminAuthenticatedShell
      admin={admin}
      backendApiEntryPath={backendApiEntryPath}
      onLogout={() => {
        logoutAdmin();
        router.replace("/admin");
      }}
      token={token}
    />
  );
}

function AdminPanelLoading({ label }: { label: string }) {
  return (
    <div className="flex min-h-48 items-center justify-center rounded-xl border border-black/[0.06] bg-white/80">
      <Loader2 className="size-5 animate-spin text-[#1d4ed8]" aria-hidden="true" />
      <span className="ml-2 text-sm text-[#6b7280]">{label}</span>
    </div>
  );
}
