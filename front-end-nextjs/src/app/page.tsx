import type { Metadata } from "next";
import { headers } from "next/headers";
import { ExploreFeed } from "@/components/feed/explore-feed";
import { apiRequestContextFromHeaders, getAuthConfig, getInitialFeedData } from "@/lib/api";
import { absoluteMediaUrl, normalizeSiteProfile } from "@/lib/seo";
import type { InitialFeedData } from "@/lib/types";
import type { SiteProfile } from "@/lib/types/site";

type HomeProps = {
  searchParams?: Promise<Record<string, string | string[] | undefined>>;
};

export async function generateMetadata(): Promise<Metadata> {
  const headerStore = await headers();
  const siteProfile = await loadSiteProfile(
    apiRequestContextFromHeaders(headerStore),
  );
  const image = absoluteMediaUrl(siteProfile.avatarUrl);

  return {
    title: {
      absolute: siteProfile.title,
    },
    description: siteProfile.description,
    applicationName: siteProfile.title,
    icons: siteProfile.avatarUrl
      ? {
          icon: siteProfile.avatarUrl,
          apple: siteProfile.avatarUrl,
        }
      : undefined,
    openGraph: {
      type: "website",
      siteName: siteProfile.title,
      title: siteProfile.title,
      description: siteProfile.description,
      url: "/",
      images: image ? [{ url: image, alt: siteProfile.title }] : undefined,
    },
  };
}

export default async function Home({ searchParams }: HomeProps) {
  const headerStore = await headers();
  const resolvedSearchParams = await searchParams;
  const initialData = await loadInitialData(
    apiRequestContextFromHeaders(headerStore),
    isOAuthSuccessCallback(resolvedSearchParams),
  );

  return <ExploreFeed initialData={initialData} />;
}

async function loadSiteProfile(context: ReturnType<typeof apiRequestContextFromHeaders>): Promise<SiteProfile> {
  try {
    const authConfig = await getAuthConfig(context);
    return normalizeSiteProfile(authConfig.siteProfile);
  } catch {
    return normalizeSiteProfile();
  }
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
  siteProfile: normalizeSiteProfile(),
};
