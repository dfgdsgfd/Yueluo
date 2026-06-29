import type { Metadata, Viewport } from "next";
import { cookies } from "next/headers";
import { getMessages } from "next-intl/server";
import { Toaster } from "sonner";
import { OAuthCallbackBootstrap } from "@/components/auth/oauth-callback-bootstrap";
import { OAuthCallbackHandler } from "@/components/auth/oauth-callback-handler";
import { PostDetailInstantHost } from "@/components/feed/post-detail-instant-host";
import { WebVitalsReporter } from "@/components/performance/web-vitals-reporter";
import { DeferredGlobalFeatures } from "@/components/providers/deferred-global-features";
import { LocaleProvider } from "@/components/providers/locale-provider";
import { NativeAppProvider } from "@/components/providers/native-app-provider";
import { QueryProvider } from "@/components/providers/query-provider";
import {
  DEFAULT_LOCALE,
  LOCALE_COOKIE_NAME,
  isAppLocale,
} from "@/i18n/locales";
import { getSiteUrl } from "@/lib/seo";
import "./globals.css";

export const unstable_instant = false;

const themeInitScript = `
(() => {
  try {
    const stored = window.localStorage.getItem("yuem_explore_theme");
    const prefersLight = window.matchMedia && window.matchMedia("(prefers-color-scheme: light)").matches;
    const theme = stored === "light" || stored === "dark" ? stored : prefersLight ? "light" : "dark";
    document.documentElement.dataset.yuemTheme = theme;
  } catch {
    document.documentElement.dataset.yuemTheme = "dark";
  }
})();
`;

export const metadata: Metadata = {
  metadataBase: new URL(getSiteUrl()),
  title: {
    default: "Yuem",
    template: "%s | Yuem",
  },
  description: "Discover, create, and share moments with Yuem.",
  applicationName: "Yuem",
  alternates: {
    canonical: "/",
  },
  openGraph: {
    type: "website",
    siteName: "Yuem",
    title: "Yuem",
    description: "Discover, create, and share moments with Yuem.",
    url: "/",
  },
  robots: {
    index: true,
    follow: true,
  },
  formatDetection: {
    address: false,
    email: false,
    telephone: false,
  },
};

export const viewport: Viewport = {
  viewportFit: "cover",
  colorScheme: "dark light",
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#f6f6f7" },
    { media: "(prefers-color-scheme: dark)", color: "#121212" },
  ],
};

export default async function RootLayout({
  children,
  modal,
}: Readonly<{
  children: React.ReactNode;
  modal: React.ReactNode;
}>) {
  const cookieStore = await cookies();
  const cookieLocale = cookieStore.get(LOCALE_COOKIE_NAME)?.value;
  const initialLocale = isAppLocale(cookieLocale) ? cookieLocale : DEFAULT_LOCALE;
  const initialMessages = await getMessages();

  return (
    <html lang={initialLocale} className="h-full antialiased" suppressHydrationWarning>
      <body className="flex min-h-dvh flex-col">
        <script dangerouslySetInnerHTML={{ __html: themeInitScript }} />
        <OAuthCallbackBootstrap />
        <LocaleProvider
          initialLocale={initialLocale}
          initialMessages={initialMessages}
        >
          <QueryProvider>
            <NativeAppProvider>
              <WebVitalsReporter />
              <OAuthCallbackHandler />
              {children}
              {modal}
              <PostDetailInstantHost />
              <DeferredGlobalFeatures />
              <Toaster richColors position="top-center" />
            </NativeAppProvider>
          </QueryProvider>
        </LocaleProvider>
      </body>
    </html>
  );
}
