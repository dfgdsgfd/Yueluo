export type PurchaseShortageKind = "balance" | "points";

const pointsShortageMarkers = [
  "insufficient_points",
  "points_insufficient",
  "error.insufficient_points",
  "error.points_insufficient",
  "积分不足",
  "积分余额不足",
  "points insufficient",
  "insufficient points",
];

const balanceShortageMarkers = [
  "insufficient_balance",
  "balance_insufficient",
  "error.insufficient_balance",
  "error.balance_insufficient",
  "月币不足",
  "余额不足",
  "balance insufficient",
  "insufficient balance",
];

export function purchaseShortageFromError(error: unknown): PurchaseShortageKind | null {
  if (!error || typeof error !== "object") {
    return null;
  }

  const errorRecord = error as Record<string, unknown>;
  const haystack = collectStrings([errorRecord.message, errorRecord.details])
    .join("\n")
    .toLowerCase();

  if (pointsShortageMarkers.some((marker) => haystack.includes(marker.toLowerCase()))) {
    return "points";
  }

  if (balanceShortageMarkers.some((marker) => haystack.includes(marker.toLowerCase()))) {
    return "balance";
  }

  return null;
}

function collectStrings(values: unknown[]): string[] {
  const strings: string[] = [];
  const seen = new Set<unknown>();

  function visit(value: unknown) {
    if (value == null) {
      return;
    }
    if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
      strings.push(String(value));
      return;
    }
    if (typeof value !== "object" || seen.has(value)) {
      return;
    }
    seen.add(value);
    if (Array.isArray(value)) {
      for (const item of value) {
        visit(item);
      }
      return;
    }
    for (const item of Object.values(value as Record<string, unknown>)) {
      visit(item);
    }
  }

  for (const value of values) {
    visit(value);
  }

  return strings;
}
