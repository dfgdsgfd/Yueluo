import type { MetadataRoute } from "next";
import { getFeedPage } from "@/lib/api";
import { absoluteSiteUrl } from "@/lib/seo";

const sitemapRequestContext = {
  forwardedHeaders: {
    "user-agent": "Yuem-NextJS-Sitemap/1.0",
    "x-internal-request": "nextjs-sitemap",
  },
};

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const now = new Date();
  const staticRoutes: MetadataRoute.Sitemap = [
    { url: absoluteSiteUrl("/"), lastModified: now, changeFrequency: "hourly", priority: 1 },
    { url: absoluteSiteUrl("/explore"), lastModified: now, changeFrequency: "hourly", priority: 0.9 },
    { url: absoluteSiteUrl("/download"), lastModified: now, changeFrequency: "weekly", priority: 0.5 },
  ];

  const feed = await getFeedPage(
    "recommended",
    { ...sitemapRequestContext, signal: AbortSignal.timeout(8_000) },
    { page: 1, limit: 100 },
  ).catch(() => null);
  const postRoutes =
    feed?.posts
      .filter((post) => !post.is_draft && (post.visibility ?? "public") === "public")
      .map((post) => ({
        url: absoluteSiteUrl(`/post?id=${encodeURIComponent(String(post.id))}`),
        lastModified: post.created_at ? new Date(post.created_at) : now,
        changeFrequency: "weekly" as const,
        priority: post.public_access_exempt ? 0.9 : 0.7,
      })) ?? [];

  return [...staticRoutes, ...postRoutes];
}
