"use client";

import {
detectBrowserLocale,
isAppLocale,
LEGACY_LOCALE_STORAGE_KEY,
LOCALE_COOKIE_NAME,
LOCALES,
type AppLocale,
} from "@/i18n/locales";
import { loadMessages } from "@/i18n/messages";
import { NextIntlClientProvider,type AbstractIntlMessages } from "next-intl";
import { createContext,useContext,useEffect,useMemo,useState } from "react";

type LocaleContextValue = {
  locale: AppLocale;
  setLocale: (locale: AppLocale) => Promise<void>;
  locales: readonly AppLocale[];
};

const LocaleContext = createContext<LocaleContextValue | null>(null);

function writeLocaleCookie(locale: AppLocale) {
  document.cookie = `${LOCALE_COOKIE_NAME}=${locale}; path=/; max-age=31536000; samesite=lax`;
}

function readLocaleCookie(): AppLocale | null {
  const match = document.cookie
    .split(";")
    .map((part) => part.trim())
    .find((part) => part.startsWith(`${LOCALE_COOKIE_NAME}=`));
  if (!match) {
    return null;
  }

  const rawValue = match.slice(LOCALE_COOKIE_NAME.length + 1);
  let value = rawValue;
  try {
    value = decodeURIComponent(rawValue);
  } catch {}

  return isAppLocale(value) ? value : null;
}

/**
 * Resolve the user's preferred locale on the client, applying any required side
 * effects (cookie persistence, legacy localStorage migration). This must only
 * run after mount — never during the initial render — so the first client render
 * matches the server HTML and avoids hydration mismatches.
 */
function resolvePreferredLocale(initialLocale: AppLocale): AppLocale {
  // 1. Honour legacy localStorage value (migrate to cookie, then drop it).
  const legacyLocale = window.localStorage.getItem(LEGACY_LOCALE_STORAGE_KEY);
  if (isAppLocale(legacyLocale)) {
    writeLocaleCookie(legacyLocale);
    window.localStorage.removeItem(LEGACY_LOCALE_STORAGE_KEY);
    return legacyLocale;
  }

  if (legacyLocale !== null) {
    window.localStorage.removeItem(LEGACY_LOCALE_STORAGE_KEY);
  }

  // 2. Honour an explicit locale cookie, including the default locale. Without
  //    this check a Chinese browser would immediately override a manual switch
  //    back to English because `en` is also the default fallback locale.
  const cookieLocale = readLocaleCookie();
  if (cookieLocale) {
    return cookieLocale;
  }

  // 3. Auto-detect from browser language preferences and persist for SSR so
  //    subsequent server renders match without an extra client switch.
  const browserLocale = detectBrowserLocale();
  if (browserLocale && browserLocale !== initialLocale) {
    writeLocaleCookie(browserLocale);
    return browserLocale;
  }

  return initialLocale;
}

export function LocaleProvider({
  children,
  initialLocale,
  initialMessages,
}: {
  children: React.ReactNode;
  initialLocale: AppLocale;
  initialMessages: AbstractIntlMessages;
}) {
  // Always start from the server-resolved locale so the first client render
  // matches the server-rendered HTML. Detecting the browser locale here would
  // make the initial render non-deterministic (e.g. a Chinese mobile browser
  // reports `zh-CN` while the server rendered `en`), causing hydration
  // mismatches across the whole translated tree. Preferred-locale resolution is
  // deferred to the post-mount effect below.
  const [locale, setLocaleState] = useState<AppLocale>(initialLocale);
  const [messages, setMessages] = useState<AbstractIntlMessages>(initialMessages);

  // Resolve the user's preferred locale *after* hydration. Updating state here
  // is safe because it runs post-mount and cannot cause a hydration mismatch.
  useEffect(() => {
    const resolved = resolvePreferredLocale(initialLocale);
    if (resolved !== initialLocale) {
      void loadMessages(resolved).then((resolvedMessages) => {
        setMessages(resolvedMessages);
        setLocaleState(resolved);
      });
    }
  }, [initialLocale]);

  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  const value = useMemo<LocaleContextValue>(
    () => ({
      locale,
      locales: LOCALES,
      async setLocale(nextLocale) {
        const nextMessages = await loadMessages(nextLocale);
        writeLocaleCookie(nextLocale);
        setMessages(nextMessages);
        setLocaleState(nextLocale);
      },
    }),
    [locale],
  );

  return (
    <LocaleContext.Provider value={value}>
      <NextIntlClientProvider
        locale={locale}
        messages={messages}
        timeZone="Asia/Shanghai"
      >
        {children}
      </NextIntlClientProvider>
    </LocaleContext.Provider>
  );
}

export function useLocaleContext() {
  const value = useContext(LocaleContext);

  if (!value) {
    throw new Error("useLocaleContext must be used within LocaleProvider.");
  }

  return value;
}
