export type SystemLogsPayload = {
  enabled?: boolean;
  items?: Array<Record<string, unknown>>;
  nextCursor?: string;
  hasMore?: boolean;
  retention_hours?: number;
};

export type PerformancePayload = Record<string, unknown> & {
  enabled?: boolean;
  retention_hours?: number;
  requests?: Record<string, unknown>;
  runtime?: Record<string, unknown>;
  runtime_series?: Array<Record<string, unknown>>;
  postgresql?: Record<string, unknown>;
  postgresql_series?: Array<Record<string, unknown>>;
  slow_requests?: { items?: Array<Record<string, unknown>> };
  slow_queries?: {
    items?: Array<Record<string, unknown>>;
    groups?: Array<Record<string, unknown>>;
  };
  redis?: Record<string, unknown>;
  versions?: Record<string, unknown>;
};

export type ObservabilityEventType = "errors" | "slow_requests" | "slow_queries";

export type ObservabilityRange = "1h" | "3h" | "6h" | "12h" | "24h";

export type ObservabilityEventsPayload = {
  enabled?: boolean;
  type?: ObservabilityEventType;
  items?: Array<Record<string, unknown>>;
  pagination?: {
    page?: number;
    limit?: number;
    total?: number;
    pages?: number;
    totalPages?: number;
    hasNextPage?: boolean;
  };
};

export type AccessLogPayload = {
  enabled?: boolean;
  items?: Array<Record<string, unknown>>;
  limit?: number;
};

export const observabilityRanges: ObservabilityRange[] = ["1h", "3h", "6h", "12h", "24h"];

export const observabilityBuckets: Record<ObservabilityRange, string> = {
  "1h": "1m",
  "3h": "1m",
  "6h": "5m",
  "12h": "10m",
  "24h": "15m",
};
