export const LOCALES = ["en", "zh-CN", "zh-TW", "vi", "ja", "ko"] as const;

export type AppLocale = (typeof LOCALES)[number];

export const DEFAULT_LOCALE: AppLocale = "en";
export const LOCALE_COOKIE_NAME = "xse.locale";
export const LEGACY_LOCALE_STORAGE_KEY = "xse.locale";

export function isAppLocale(value: string | undefined | null): value is AppLocale {
  return LOCALES.includes(value as AppLocale);
}

/**
 * Map from BCP-47 language tags (or prefixes) reported by `navigator.language`
 * to the application locale.  Matching is tried in order:
 *   1. exact match (e.g. "zh-TW")
 *   2. language-only prefix (e.g. "zh" → "zh-CN")
 */
const BROWSER_LANG_MAP: Record<string, AppLocale> = {
  en: "en",
  "zh-cn": "zh-CN",
  "zh-tw": "zh-TW",
  "zh-hk": "zh-TW",
  zh: "zh-CN",
  vi: "vi",
  ja: "ja",
  ko: "ko",
};

/**
 * Detect the best matching app locale from the browser's language preferences.
 * Returns `undefined` when no supported locale can be inferred (caller should
 * fall back to `DEFAULT_LOCALE`).
 */
export function detectBrowserLocale(): AppLocale | undefined {
  if (typeof navigator === "undefined") {
    return undefined;
  }

  const languages = navigator.languages?.length ? navigator.languages : [navigator.language];

  for (const lang of languages) {
    if (!lang) continue;
    const tag = lang.toLowerCase();

    // Try exact match first (e.g. "zh-TW", "zh-CN").
    if (BROWSER_LANG_MAP[tag]) {
      return BROWSER_LANG_MAP[tag];
    }

    // Try language-only prefix (e.g. "en-US" → "en").
    const prefix = tag.split("-")[0];
    if (BROWSER_LANG_MAP[prefix]) {
      return BROWSER_LANG_MAP[prefix];
    }
  }

  return undefined;
}

