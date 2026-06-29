export const POINTS_AWARD_EVENT = "yuem:points-award";

export type PointsAwardEventPayload = {
  amount: number;
  balanceAfter?: number;
  dailyAwarded?: number;
  message?: string;
  reason?: string;
};

const recentAwards = new Map<string, number>();
const DUPLICATE_AWARD_WINDOW_MS = 300;
const pendingAwards: PointsAwardEventPayload[] = [];

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function numberFromValue(value: unknown) {
  const numeric = typeof value === "number" ? value : Number(value);
  return Number.isFinite(numeric) ? numeric : undefined;
}

function stringFromValue(value: unknown) {
  return typeof value === "string" && value.trim() ? value.trim() : undefined;
}

export function normalizePointsAward(value: unknown): PointsAwardEventPayload | null {
  if (!isRecord(value)) {
    return null;
  }

  const awardedValue = value.awarded ?? value.Awarded;
  const amount = numberFromValue(value.amount ?? value.Amount ?? value.points ?? value.awarded_points ?? value.awardedPoints);
  const awarded = typeof awardedValue === "boolean" ? awardedValue : Boolean(amount && amount > 0);

  if (!awarded || !amount || amount <= 0) {
    return null;
  }

  const balanceAfter = numberFromValue(value.balance_after ?? value.balanceAfter ?? value.BalanceAfter);
  const dailyAwarded = numberFromValue(value.daily_awarded ?? value.dailyAwarded ?? value.DailyAwarded);
  const message = stringFromValue(value.message ?? value.Message);
  const reason = stringFromValue(value.reason ?? value.Reason);

  return {
    amount,
    ...(balanceAfter === undefined ? {} : { balanceAfter }),
    ...(dailyAwarded === undefined ? {} : { dailyAwarded }),
    ...(message ? { message } : {}),
    ...(reason ? { reason } : {}),
  };
}

function normalizePointsAwardList(value: unknown): PointsAwardEventPayload[] {
  if (Array.isArray(value)) {
    return value.flatMap((item) => {
      const award = normalizePointsAward(item);
      return award ? [award] : [];
    });
  }

  const award = normalizePointsAward(value);
  return award ? [award] : [];
}

function collectPointsAwards(value: unknown, depth = 0): PointsAwardEventPayload[] {
  if (!isRecord(value)) {
    return [];
  }

  const directAwards = [
    ...normalizePointsAwardList(value.points_award ?? value.pointsAward ?? value.PointsAward),
    ...normalizePointsAwardList(value.points_awards ?? value.pointsAwards ?? value.PointsAwards),
  ];
  if (directAwards.length || depth >= 2) {
    return directAwards;
  }

  return collectPointsAwards(value.data ?? value.Data, depth + 1);
}

export function dispatchPointsAward(award: PointsAwardEventPayload) {
  if (typeof window === "undefined") {
    return;
  }

  const key = `${award.amount}:${award.reason ?? ""}:${award.balanceAfter ?? ""}`;
  const now = Date.now();
  const lastDispatchedAt = recentAwards.get(key);
  if (lastDispatchedAt && now - lastDispatchedAt < DUPLICATE_AWARD_WINDOW_MS) {
    return;
  }
  recentAwards.set(key, now);
  pendingAwards.push(award);
  if (pendingAwards.length > 8) {
    pendingAwards.splice(0, pendingAwards.length - 8);
  }

  window.dispatchEvent(new CustomEvent<PointsAwardEventPayload>(POINTS_AWARD_EVENT, { detail: award }));
}

export function consumePendingPointsAwards() {
  return pendingAwards.splice(0, pendingAwards.length);
}

function dispatchPointsAwards(awards: PointsAwardEventPayload[]) {
  for (const award of awards) {
    dispatchPointsAward(award);
  }
}

export function emitPointsAwardFromResponsePayload(payload: unknown) {
  if (isRecord(payload)) {
    const code = numberFromValue(payload.code);
    if (code !== undefined && code !== 200) {
      return;
    }
    if (payload.success === false) {
      return;
    }
  }

  dispatchPointsAwards(collectPointsAwards(payload));
}
