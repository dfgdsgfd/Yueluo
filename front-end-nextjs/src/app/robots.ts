import type { MetadataRoute } from "next";
import { absoluteSiteUrl } from "@/lib/seo";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      {
        userAgent: "*",
        allow: ["/", "/explore", "/post", "/download", "/api/file/"],
        disallow: [
          "/admin",
          "/messages",
          "/notifications",
          "/profile",
          "/settings",
          "/wallet",
          "/publish",
          "/api/admin/",
          "/api/messages/",
          "/api/notifications/",
          "/api/users/me",
        ],
      },
    ],
    sitemap: absoluteSiteUrl("/sitemap.xml"),
  };
}
