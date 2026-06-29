"use client";

import { History } from "lucide-react";
import { metadataRecord, metadataText, formatUsageSize, formatUsageTime } from "./hidden-watermark-access-model";
import { Panel } from "./layout-widgets";
import { EmptyBlock, LoadingBlock } from "./resource-editor";
import type { HiddenWatermarkAccessController } from "./hidden-watermark-access-view-types";

export function WatermarkUsageSection({ controller }: { controller: HiddenWatermarkAccessController }) {
  const { loading, t, usageRecords } = controller;
  return (
      <Panel title={t("usage.title")} icon={History}>
        {loading ? (
          <LoadingBlock label={t("usage.loading")} />
        ) : usageRecords.length ? (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[1080px] text-left text-sm">
              <thead className="text-xs uppercase tracking-normal text-[#7a8190]">
                <tr>
                  <th className="px-3 py-2">{t("usage.time")}</th>
                  <th className="px-3 py-2">{t("usage.actor")}</th>
                  <th className="px-3 py-2">{t("usage.result")}</th>
                  <th className="px-3 py-2">{t("usage.file")}</th>
                  <th className="px-3 py-2">{t("usage.watermark")}</th>
                  <th className="px-3 py-2">{t("usage.reason")}</th>
                  <th className="px-3 py-2">{t("usage.ip")}</th>
                  <th className="px-3 py-2">{t("usage.path")}</th>
                </tr>
              </thead>
              <tbody>
                {usageRecords.map((record) => {
                  const metadata = metadataRecord(record.metadata);
                  const failed = record.outcome !== "success";
                  return (
                    <tr key={String(record.id)} className={failed ? "border-t border-black/[0.05] bg-red-50/70" : "border-t border-black/[0.05]"}>
                      <td className="whitespace-nowrap px-3 py-2 text-[#59606c]">{formatUsageTime(record.created_at)}</td>
                      <td className="px-3 py-2">
                        <span className="block font-semibold text-[#252932]">{record.actor_display_id || record.actor_type || "-"}</span>
                        <span className="text-xs text-[#8a90a0]">{record.actor_id ?? "-"}</span>
                      </td>
                      <td className="px-3 py-2">
                        <span className={failed ? "inline-flex rounded-full bg-red-100 px-2 py-1 text-xs font-semibold text-red-700" : "inline-flex rounded-full bg-emerald-100 px-2 py-1 text-xs font-semibold text-emerald-700"}>
                          {failed ? t("usage.failed") : t("usage.success")}
                        </span>
                      </td>
                      <td className="max-w-[220px] px-3 py-2 text-[#59606c]">
                        <span className="block truncate" title={metadataText(metadata.filename)}>{metadataText(metadata.filename) || "-"}</span>
                        <span className="text-xs text-[#8a90a0]">{formatUsageSize(metadata.file_size)}</span>
                      </td>
                      <td className="px-3 py-2 text-[#59606c]">
                        {metadata.found === true
                          ? metadata.valid === true
                            ? t("usage.valid")
                            : t("usage.invalid")
                          : t("usage.notFound")}
                      </td>
                      <td className="max-w-[220px] break-words px-3 py-2 text-[#59606c]">{record.reason_code || "-"}</td>
                      <td className="px-3 py-2 text-[#59606c]">{record.ip || "-"}</td>
                      <td className="max-w-[240px] truncate px-3 py-2 text-[#59606c]" title={record.path || ""}>{record.path || "-"}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        ) : (
          <EmptyBlock icon={History} label={t("usage.empty")} />
        )}
      </Panel>
  );
}

