import { getRequestConfig } from "next-intl/server";
import { cookies } from "next/headers";
import {
DEFAULT_LOCALE,
LOCALE_COOKIE_NAME,
isAppLocale,
} from "./locales";
import { loadMessages } from "./messages";

export default getRequestConfig(async () => {
  const cookieStore = await cookies();
  const cookieLocale = cookieStore.get(LOCALE_COOKIE_NAME)?.value;
  const locale = isAppLocale(cookieLocale) ? cookieLocale : DEFAULT_LOCALE;

  return {
    locale,
    messages: await loadMessages(locale),
    timeZone: "Asia/Shanghai",
  };
});
