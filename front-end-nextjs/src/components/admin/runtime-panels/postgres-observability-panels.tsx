"use client";

import { useTranslations } from "next-intl";
import { Database, Radio, Search, Server } from "lucide-react";

import { EmptyBlock, InfoTile, Panel, StatusPill, formatBytes, formatCompact, formatMs, formatPercent, readableValue, recordArray, recordObject } from "./shared";

export function PostgresAdvancedPanel({
  activity,
  locks,
  tableHealth,
  wal,
  io,
  checkpointer,
  deadlocks,
  topSQL,
  topDead,
}: {
  activity: Record<string, unknown>;
  locks: Record<string, unknown>;
  tableHealth: Record<string, unknown>;
  wal: Record<string, unknown>;
  io: Record<string, unknown>;
  checkpointer: Record<string, unknown>;
  deadlocks: unknown;
  topSQL: Array<Record<string, unknown>>;
  topDead: Array<Record<string, unknown>>;
}) {
  const t = useTranslations("adminObservability");
  return (
    <div className="grid min-w-0 gap-4 xl:grid-cols-2">
      <Panel title={t("pgActivityTitle")} icon={Database}>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <InfoTile label={t("pgTotalConnections")} value={formatCompact(activity.total_connections)} />
          <InfoTile label={t("pgActiveConnections")} value={formatCompact(activity.active_connections)} />
          <InfoTile label={t("pgIdleTransaction")} value={formatCompact(activity.idle_in_transaction)} />
          <InfoTile label={t("pgLongTransactions")} value={formatCompact(activity.long_transactions)} />
          <InfoTile label={t("pgWaitingLocks")} value={formatCompact(locks.waiting_locks)} />
          <InfoTile label={t("pgDeadlocks")} value={formatCompact(deadlocks ?? 0)} />
          <InfoTile label={t("pgMaxTxAge")} value={formatMs(activity.max_transaction_age_ms)} />
          <InfoTile label={t("pgMaxQueryAge")} value={formatMs(activity.max_query_age_ms)} />
        </div>
      </Panel>
      <Panel title={t("pgTableHealthTitle")} icon={Database}>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <InfoTile label={t("pgLiveTuples")} value={formatCompact(tableHealth.live_tuples)} />
          <InfoTile label={t("pgDeadTuples")} value={formatCompact(tableHealth.dead_tuples)} />
          <InfoTile label={t("pgDeadRatio")} value={formatPercent(tableHealth.dead_ratio)} />
          <InfoTile label={t("pgStaleAnalyze")} value={formatCompact(tableHealth.stale_analyze)} />
          <InfoTile label={t("pgAutoVacuum")} value={formatCompact(tableHealth.auto_vacuum)} />
          <InfoTile label={t("pgManualVacuum")} value={formatCompact(tableHealth.manual_vacuum)} />
          <InfoTile label={t("pgAutoAnalyze")} value={formatCompact(tableHealth.auto_analyze)} />
          <InfoTile label={t("pgManualAnalyze")} value={formatCompact(tableHealth.manual_analyze)} />
        </div>
      </Panel>
      <Panel title={t("pgWalIoTitle")} icon={Radio}>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <InfoTile label={t("pgWalBytes")} value={formatBytes(wal.wal_bytes)} />
          <InfoTile label={t("pgWalRecords")} value={formatCompact(wal.wal_records)} />
          <InfoTile label={t("pgWalSync")} value={formatCompact(wal.wal_sync)} />
          <InfoTile label={t("pgWalSyncTime")} value={formatMs(wal.wal_sync_ms)} />
          <InfoTile label={t("pgIoReads")} value={formatCompact(io.reads)} />
          <InfoTile label={t("pgIoWrites")} value={formatCompact(io.writes)} />
          <InfoTile label={t("pgIoHits")} value={formatCompact(io.hits)} />
          <InfoTile label={t("pgIoFsyncs")} value={formatCompact(io.fsyncs)} />
        </div>
        {io.message ? <p className="mt-3 break-all text-xs text-amber-800">{readableValue(io.message)}</p> : null}
      </Panel>
      <Panel title={t("pgCheckpointTitle")} icon={Server}>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <InfoTile label={t("pgCheckpointSource")} value={readableValue(checkpointer.source)} />
          <InfoTile label={t("pgCheckpointTimed")} value={formatCompact(checkpointer.num_timed)} />
          <InfoTile label={t("pgCheckpointRequested")} value={formatCompact(checkpointer.num_requested)} />
          <InfoTile label={t("pgBuffersWritten")} value={formatCompact(checkpointer.buffers_written)} />
          <InfoTile label={t("pgCheckpointWriteTime")} value={formatMs(checkpointer.write_time_ms)} />
          <InfoTile label={t("pgCheckpointSyncTime")} value={formatMs(checkpointer.sync_time_ms)} />
          <InfoTile label={t("pgBuffersClean")} value={formatCompact(checkpointer.buffers_clean)} />
          <InfoTile label={t("pgBuffersBackend")} value={formatCompact(checkpointer.buffers_backend)} />
        </div>
      </Panel>
      <Panel title={t("pgTopSqlTitle")} icon={Search}>
        {topSQL.length ? (
          <div className="grid max-h-[360px] gap-2 overflow-y-auto pr-1">
            {topSQL.map((item, index) => (
              <div key={String(item.queryid ?? index)} className="rounded-lg border border-black/[0.06] bg-[#f8fafc] p-3">
                <div className="flex flex-wrap items-center gap-2 text-xs text-[#7b808c]">
                  <span className="font-semibold text-[#252932]">{t("pgCalls", { value: formatCompact(item.calls) })}</span>
                  <span>{formatMs(item.mean_exec_time_ms)}</span>
                  <span>{formatMs(item.total_exec_time_ms)}</span>
                  <span>{t("pgRows", { value: formatCompact(item.rows) })}</span>
                </div>
                <p className="mt-1 line-clamp-2 break-all text-xs font-semibold text-[#252932]">{readableValue(item.query)}</p>
              </div>
            ))}
          </div>
        ) : (
          <EmptyBlock icon={Search} label={t("pgNoTopSql")} />
        )}
      </Panel>
      <Panel title={t("pgTopDeadTitle")} icon={Database}>
        {topDead.length ? (
          <div className="grid max-h-[360px] gap-2 overflow-y-auto pr-1">
            {topDead.map((item, index) => (
              <div key={String(item.table ?? index)} className="rounded-lg border border-black/[0.06] bg-[#f8fafc] p-3">
                <div className="flex min-w-0 flex-wrap items-center gap-2 text-xs text-[#7b808c]">
                  <span className="min-w-0 truncate font-semibold text-[#252932]">{readableValue(item.table)}</span>
                  <span>{t("pgDeadTuples")} {formatCompact(item.dead_tuples)}</span>
                  <span>{formatPercent(item.dead_ratio)}</span>
                </div>
                <p className="mt-1 text-xs text-[#7b808c]">{t("pgLastVacuum")} {readableValue(item.last_vacuum ?? item.last_autovacuum)} · {t("pgLastAnalyze")} {readableValue(item.last_analyze ?? item.last_autoanalyze)}</p>
              </div>
            ))}
          </div>
        ) : (
          <EmptyBlock icon={Database} label={t("pgNoTableHealth")} />
        )}
      </Panel>
    </div>
  );
}

export function PostgresDiagnosticsPanel({ diagnostics }: { diagnostics: Record<string, unknown> }) {
  const t = useTranslations("adminObservability");
  const pool = recordObject(diagnostics.pool);
  const waitEvents = recordArray(recordObject(diagnostics.wait_events).items);
  const blockingLocks = recordArray(recordObject(diagnostics.blocking_locks).items);
  const longTransactions = recordArray(recordObject(diagnostics.long_transactions).items);
  const pressure = String(pool.pressure ?? "low");
  return (
    <div className="grid min-w-0 gap-4 xl:grid-cols-2">
      <Panel title={t("pgPoolDiagnosticsTitle")} icon={Server}>
        <div className="mb-3 flex items-center justify-between gap-3">
          <span className="text-sm font-semibold text-[#252932]">{t("pgPoolPressure")}</span>
          <StatusPill value={t(`pressure.${pressure}`)} tone={pressure === "high" ? "red" : pressure === "medium" ? "amber" : "green"} />
        </div>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <InfoTile label={t("pgPoolUsage")} value={formatPercent(pool.usage_ratio)} />
          <InfoTile label={t("pgPoolWaitCount")} value={formatCompact(pool.wait_count)} />
          <InfoTile label={t("pgPoolWaitAvg")} value={formatMs(pool.wait_avg_ms)} />
          <InfoTile label={t("pgPoolWaitDuration")} value={formatMs(pool.wait_duration_ms)} />
          <InfoTile label={t("pgOpen")} value={formatCompact(pool.open_connections)} />
          <InfoTile label={t("pgInUse")} value={formatCompact(pool.in_use)} />
          <InfoTile label={t("pgIdle")} value={formatCompact(pool.idle)} />
          <InfoTile label={t("pgMaxOpen")} value={formatCompact(pool.max_open)} />
        </div>
      </Panel>
      <Panel title={t("pgWaitEventsTitle")} icon={Radio}>
        <DiagnosticList
          items={waitEvents}
          emptyLabel={t("pgNoWaitEvents")}
          primary={(item) => `${readableValue(item.wait_event_type)} / ${readableValue(item.wait_event)}`}
          secondary={(item) => `${t("count")} ${formatCompact(item.count)} · ${t("maxWait")} ${formatMs(item.max_wait_ms)}`}
        />
      </Panel>
      <Panel title={t("pgBlockingLocksTitle")} icon={Database}>
        <DiagnosticList
          items={blockingLocks}
          emptyLabel={t("pgNoBlockingLocks")}
          primary={(item) => `${t("blockedPid")} ${readableValue(item.blocked_pid)} · ${t("blockerPid")} ${readableValue(item.blocker_pid)}`}
          secondary={(item) => `${t("wait")} ${formatMs(item.wait_ms)} · ${readableValue(item.wait_event_type)} / ${readableValue(item.wait_event)}`}
          detail={(item) => readableValue(item.blocked_query || item.blocker_query)}
        />
      </Panel>
      <Panel title={t("pgLongTransactionsTitle")} icon={Search}>
        <DiagnosticList
          items={longTransactions}
          emptyLabel={t("pgNoLongTransactions")}
          primary={(item) => `${readableValue(item.state)} · PID ${readableValue(item.pid)}`}
          secondary={(item) => `${t("transactionAge")} ${formatMs(item.xact_age_ms)} · ${t("queryAge")} ${formatMs(item.query_age_ms)}`}
          detail={(item) => readableValue(item.query)}
        />
      </Panel>
    </div>
  );
}

function DiagnosticList({
  items,
  emptyLabel,
  primary,
  secondary,
  detail,
}: {
  items: Array<Record<string, unknown>>;
  emptyLabel: string;
  primary: (item: Record<string, unknown>) => string;
  secondary: (item: Record<string, unknown>) => string;
  detail?: (item: Record<string, unknown>) => string;
}) {
  if (!items.length) return <EmptyBlock icon={Database} label={emptyLabel} />;
  return (
    <div className="grid max-h-[360px] gap-2 overflow-y-auto pr-1">
      {items.map((item, index) => (
        <div key={String(item.id ?? item.pid ?? item.wait_event ?? index)} className="rounded-lg border border-black/[0.06] bg-[#f8fafc] p-3">
          <p className="line-clamp-1 break-all text-xs font-semibold text-[#252932]">{primary(item)}</p>
          <p className="mt-1 text-xs text-[#7b808c]">{secondary(item)}</p>
          {detail ? <p className="mt-2 line-clamp-2 break-all text-xs text-[#59606c]">{detail(item)}</p> : null}
        </div>
      ))}
    </div>
  );
}
