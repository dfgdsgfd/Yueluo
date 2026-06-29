import type { AbstractIntlMessages } from "next-intl";
import type { AppLocale } from "./locales";

type MessageModule = { default: AbstractIntlMessages };
type MessageLoader = () => Promise<MessageModule>;

const loaders: Record<AppLocale, readonly MessageLoader[]> = {
  en: [
    () => import("@/messages/en/common.json"),
    () => import("@/messages/en/network.json"),
    () => import("@/messages/en/watermark.json"),
    () => import("@/messages/en/wallet.json"),
    () => import("@/messages/en/feed.json"),
    () => import("@/messages/en/profile.json"),
    () => import("@/messages/en/notifications.json"),
    () => import("@/messages/en/publish.json"),
    () => import("@/messages/en/admin.json"),
    () => import("@/messages/en/admin-observability.json"),
    () => import("@/messages/en/admin-logs.json"),
  ],
  "zh-CN": [
    () => import("@/messages/zh-CN/common.json"),
    () => import("@/messages/zh-CN/network.json"),
    () => import("@/messages/zh-CN/watermark.json"),
    () => import("@/messages/zh-CN/wallet.json"),
    () => import("@/messages/zh-CN/feed.json"),
    () => import("@/messages/zh-CN/profile.json"),
    () => import("@/messages/zh-CN/notifications.json"),
    () => import("@/messages/zh-CN/publish.json"),
    () => import("@/messages/zh-CN/admin.json"),
    () => import("@/messages/zh-CN/admin-observability.json"),
    () => import("@/messages/zh-CN/admin-logs.json"),
  ],
  "zh-TW": [
    () => import("@/messages/zh-TW/common.json"),
    () => import("@/messages/zh-TW/network.json"),
    () => import("@/messages/zh-TW/watermark.json"),
    () => import("@/messages/zh-TW/wallet.json"),
    () => import("@/messages/zh-TW/feed.json"),
    () => import("@/messages/zh-TW/profile.json"),
    () => import("@/messages/zh-TW/notifications.json"),
    () => import("@/messages/zh-TW/publish.json"),
    () => import("@/messages/zh-TW/admin.json"),
    () => import("@/messages/zh-TW/admin-observability.json"),
    () => import("@/messages/zh-TW/admin-logs.json"),
  ],
  vi: [
    () => import("@/messages/vi/common.json"),
    () => import("@/messages/vi/network.json"),
    () => import("@/messages/vi/watermark.json"),
    () => import("@/messages/vi/wallet.json"),
    () => import("@/messages/vi/feed.json"),
    () => import("@/messages/vi/profile.json"),
    () => import("@/messages/vi/notifications.json"),
    () => import("@/messages/vi/publish.json"),
    () => import("@/messages/vi/admin.json"),
    () => import("@/messages/vi/admin-observability.json"),
    () => import("@/messages/vi/admin-logs.json"),
  ],
  ja: [
    () => import("@/messages/ja/common.json"),
    () => import("@/messages/ja/network.json"),
    () => import("@/messages/ja/watermark.json"),
    () => import("@/messages/ja/wallet.json"),
    () => import("@/messages/ja/feed.json"),
    () => import("@/messages/ja/profile.json"),
    () => import("@/messages/ja/notifications.json"),
    () => import("@/messages/ja/publish.json"),
    () => import("@/messages/ja/admin.json"),
    () => import("@/messages/ja/admin-observability.json"),
    () => import("@/messages/ja/admin-logs.json"),
  ],
  ko: [
    () => import("@/messages/ko/common.json"),
    () => import("@/messages/ko/network.json"),
    () => import("@/messages/ko/watermark.json"),
    () => import("@/messages/ko/wallet.json"),
    () => import("@/messages/ko/feed.json"),
    () => import("@/messages/ko/profile.json"),
    () => import("@/messages/ko/notifications.json"),
    () => import("@/messages/ko/publish.json"),
    () => import("@/messages/ko/admin.json"),
    () => import("@/messages/ko/admin-observability.json"),
    () => import("@/messages/ko/admin-logs.json"),
  ],
};

export async function loadMessages(locale: AppLocale): Promise<AbstractIntlMessages> {
  const modules = await Promise.all(loaders[locale].map((load) => load()));
  return Object.assign({}, ...modules.map((module) => module.default));
}
