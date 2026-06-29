import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";

function parseAllowedDevOrigins(value = "") {
  return value
    .split(",")
    .map((origin) => origin.trim())
    .filter(Boolean);
}

const nextConfig: NextConfig = {
  cacheComponents: true,

  allowedDevOrigins: [
    "127.0.0.1",
    ...parseAllowedDevOrigins(process.env.NEXT_ALLOWED_DEV_ORIGINS),
  ],

  devIndicators: false,
  reactCompiler: true,

  experimental: {
    exposeTestingApiInProductionBuild:
      process.env.NEXT_EXPOSE_TESTING_API === "1",
    instantNavigationDevToolsToggle: true,
    viewTransition: true,
  },

  images: {
    // 允许正式环境中的图片优化器访问私有 IP。
    // Next.js 不支持只配置 192.168.0.0/16，会允许全部私有 IP。
    dangerouslyAllowLocalIP: true,

    // 正式环境继续使用 Next.js 图片优化器。
    unoptimized: false,

    minimumCacheTTL: 14_400,

    imageSizes: [32, 48, 64, 96, 128, 192, 256, 320, 384, 512],
    qualities: [70, 75],

    localPatterns: [
      {
        pathname: "/api/file/**",
      },
      {
        pathname: "/creator-center/**",
      },
    ],

    remotePatterns: [
      {
        protocol: "https",
        hostname: "cs.yuelk.com",
        pathname: "/api/file/**",
      },
      {
        protocol: "https",
        hostname: "**.yuelk.com",
        pathname: "/api/file/**",
      },
      {
        protocol: "https",
        hostname: "yuelk.com",
        pathname: "/api/file/**",
      },
      {
        protocol: "https",
        hostname: "images.unsplash.com",
        pathname: "/**",
      },
      {
        protocol: "http",
        hostname: "localhost",
        pathname: "/**",
      },
      {
        protocol: "http",
        hostname: "127.0.0.1",
        pathname: "/**",
      },
    ],

    dangerouslyAllowSVG: true,
    contentSecurityPolicy:
      "default-src 'self'; script-src 'none'; sandbox;",
  },

  async rewrites() {
    return [];
  },
};

const withNextIntl = createNextIntlPlugin("./src/i18n/request.ts");

export default withNextIntl(nextConfig);
