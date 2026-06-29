import type { useTranslations } from "next-intl";

export type LogRange = "1h" | "3h" | "6h" | "12h" | "today" | "3d" | "7d" | "30d" | "365d";
export type VisitorMode = "all" | "exclude" | "only";
export type LogTab = "access" | "operation" | "security" | "system" | "points" | "balance";
export type AdminLogsTranslator = ReturnType<typeof useTranslations>;

export const rangeOptions: LogRange[] = ["1h", "3h", "6h", "12h", "today", "3d", "7d", "30d", "365d"];
export const pageSizeOptions = [10, 20, 30, 50, 100];

export function initialLogPages(): Record<LogTab, number> {
  return { access: 1, operation: 1, security: 1, system: 1, points: 1, balance: 1 };
}
