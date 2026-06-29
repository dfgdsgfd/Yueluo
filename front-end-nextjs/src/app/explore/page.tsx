import { headers } from "next/headers";
import { ExploreFeed } from "@/components/feed/explore-feed";
import { apiRequestContextFromHeaders, getInitialFeedData } from "@/lib/api";
import type { InitialFeedData } from "@/lib/types";

export const metadata = {
  title: "Explore",
};

type ExplorePageProps = {
  searchParams?: Promise<Record<string, string | string[] | undefined>>;
};

export default async function ExplorePage({ searchParams }: ExplorePageProps) {
  const headerStore = await headers();
  const resolvedSearchParams = await searchParams;
  const initialData = await loadInitialData(
    apiRequestContextFromHeaders(headerStore),
    isOAuthSuccessCallback(resolvedSearchParams),
  );

  return <ExploreFeed initialData={initialData} />;
}

async function loadInitialData(context: ReturnType<typeof apiRequestContextFromHeaders>, allowClientOAuthBootstrap: boolean) {
  if (allowClientOAuthBootstrap) {
    return emptyInitialFeedData;
  }

  try {
    return await getInitialFeedData(context);
  } catch {
    return emptyInitialFeedData;
  }
}

function isOAuthSuccessCallback(searchParams?: Record<string, string | string[] | undefined>) {
  return getSearchParam(searchParams, "oauth2_login") === "success";
}

function getSearchParam(
  searchParams: Record<string, string | string[] | undefined> | undefined,
  key: string,
) {
  const value = searchParams?.[key];
  return Array.isArray(value) ? value[0] : value;
}

const emptyInitialFeedData: InitialFeedData = {
  posts: [],
  categories: [],
  pagination: { page: 1, limit: 24, total: 0, pages: 0, hasNextPage: false },
  source: "backend",
};
