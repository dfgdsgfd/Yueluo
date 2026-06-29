import type { AdminListResource } from "@/lib/types";
import type { AdminView } from "./types";

export const ADMIN_VIEW_SEARCH_PARAM = "view";

type SearchParamsLike = {
  get(name: string): string | null;
  toString(): string;
};

type NonResourceAdminViewKind = Exclude<AdminView["kind"], "resource">;

const adminResourceSlugs = [
  "admins",
  "ai-moderation-logs",
  "announcements",
  "app-versions",
  "audit",
  "banned-word-categories",
  "banned-words",
  "categories",
  "collections",
  "comments",
  "users",
  "feedback",
  "file-recycle-bin",
  "follows",
  "licenses",
  "likes",
  "media-library",
  "notification-templates",
  "open-apis",
  "posts",
  "posts-quality",
  "post-configs",
  "quality-reward-settings",
  "content-review",
  "reports",
  "sessions",
  "system-notifications",
  "tags",
  "user-configs",
  "user-toolbar",
] as const satisfies readonly AdminListResource[];

const adminResourceSlugSet = new Set<string>(adminResourceSlugs);

const adminViewBySlug = {
  dashboard: { kind: "dashboard" },
  withdraw: { kind: "withdraw" },
  invite: { kind: "invite" },
  coupon: { kind: "coupon" },
  points: { kind: "points" },
  "onboarding-settings": { kind: "onboarding-settings" },
  "ai-agent": { kind: "ai-agent" },
  "access-block": { kind: "access-block" },
  "hidden-watermark-access": { kind: "hidden-watermark-access" },
  settings: { kind: "settings" },
  "system-update": { kind: "system-update" },
  queues: { kind: "queues" },
  logs: { kind: "logs" },
  observability: { kind: "observability" },
  maintenance: { kind: "maintenance" },
  database: { kind: "database" },
  operations: { kind: "operations" },
  "component-check": { kind: "component-check" },
} as const satisfies Record<NonResourceAdminViewKind, AdminView>;

export function parseAdminViewSearchParams(searchParams: SearchParamsLike | null): AdminView {
  return parseAdminViewSlug(searchParams?.get(ADMIN_VIEW_SEARCH_PARAM));
}

export function parseAdminViewSlug(value: string | null | undefined): AdminView {
  const slug = value?.trim();
  if (!slug) {
    return adminViewBySlug.dashboard;
  }
  if (adminResourceSlugSet.has(slug)) {
    return { kind: "resource", resource: slug as AdminListResource };
  }
  if (slug in adminViewBySlug) {
    return adminViewBySlug[slug as NonResourceAdminViewKind];
  }
  return adminViewBySlug.dashboard;
}

export function adminViewToSlug(view: AdminView) {
  return view.kind === "resource" ? view.resource : view.kind;
}

export function adminViewSearchSuffix(
  view: AdminView,
  currentSearchParams: Pick<SearchParamsLike, "toString"> | null,
) {
  const params = new URLSearchParams(currentSearchParams?.toString() ?? "");
  const slug = adminViewToSlug(view);
  if (slug === "dashboard") {
    params.delete(ADMIN_VIEW_SEARCH_PARAM);
  } else {
    params.set(ADMIN_VIEW_SEARCH_PARAM, slug);
  }
  const query = params.toString();
  return query ? `?${query}` : "";
}
