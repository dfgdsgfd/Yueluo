"use client";

import { FileSearch, ShieldCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Panel } from "./layout-widgets";
import { EmptyBlock, KeyValueGrid } from "./resource-editor";
import type { HiddenWatermarkAccessController } from "./hidden-watermark-access-view-types";

export function WatermarkExtractSection({ controller }: { controller: HiddenWatermarkAccessController }) {
  const { extractWatermark, extracting, extractionProgress, setWatermarkFile, setWatermarkReferenceFile, setWatermarkResult, t, watermarkFile, watermarkResult } = controller;
  return (
      <Panel title={t("extract.title")} icon={ShieldCheck}>
        <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)]">
          <div className="min-w-0 rounded-lg border border-dashed border-black/10 bg-[#fafbfe] p-4">
            <label className="block text-sm font-semibold text-[#343944]" htmlFor="watermark-extract-file">
              {t("extract.fileLabel")}
            </label>
            <input
              id="watermark-extract-file"
              type="file"
              accept="image/jpeg,image/png,image/webp,image/avif"
              className="mt-3 block w-full min-w-0 text-sm text-[#5e6470] file:mr-3 file:rounded-lg file:border-0 file:bg-[#e9efff] file:px-3 file:py-2 file:font-semibold file:text-[#1d4ed8]"
              onChange={(event) => {
                setWatermarkFile(event.target.files?.[0] ?? null);
                setWatermarkResult(null);
              }}
            />
            <label className="mt-4 block text-sm font-semibold text-[#343944]" htmlFor="watermark-reference-file">
              {t("extract.referenceFileLabel")}
            </label>
            <input
              id="watermark-reference-file"
              type="file"
              accept="image/jpeg,image/png,image/webp,image/avif"
              className="mt-3 block w-full min-w-0 text-sm text-[#5e6470] file:mr-3 file:rounded-lg file:border-0 file:bg-[#e9efff] file:px-3 file:py-2 file:font-semibold file:text-[#1d4ed8]"
              onChange={(event) => {
                setWatermarkReferenceFile(event.target.files?.[0] ?? null);
                setWatermarkResult(null);
              }}
            />
            <p className="mt-2 text-xs leading-5 text-[#7b808c]">{t("extract.referenceDescription")}</p>
            <Button
              type="button"
              variant="outline"
              className="mt-3 h-9 rounded-lg"
              disabled={!watermarkFile || extracting}
              onClick={() => void extractWatermark()}
            >
              <FileSearch className="size-4" />
              <span>{extracting ? t("extract.extracting") : t("extract.action")}</span>
            </Button>
            {extracting ? (
              <div className="mt-3 rounded-lg border border-[#1d4ed8]/12 bg-[#f3f7ff] p-3">
                <div className="flex items-center justify-between gap-3 text-xs font-semibold text-[#334155]">
                  <span>{t(`extract.progress.${extractionProgress.stage}`)}</span>
                  <span>{t("extract.progress.elapsed", { seconds: Math.floor(extractionProgress.elapsedMs / 1000) })}</span>
                </div>
                <div className="mt-2 h-2 overflow-hidden rounded-full bg-[#1d4ed8]/10">
                  <div
                    className="h-full rounded-full bg-[#1d4ed8] transition-[width] duration-300"
                    style={{ width: `${extractionProgress.percent}%` }}
                  />
                </div>
                <p className="mt-2 text-xs leading-5 text-[#64748b]">
                  {extractionProgress.total
                    ? t("extract.progress.units", {
                        completed: extractionProgress.completed ?? 0,
                        total: extractionProgress.total,
                      })
                    : extractionProgress.heartbeat
                      ? t(
                          extractionProgress.source === "engine"
                            ? "extract.progress.heartbeatEngine"
                            : "extract.progress.heartbeatGateway",
                        )
                      : t("extract.progress.live")}
                </p>
              </div>
            ) : null}
            <p className="mt-3 text-xs leading-5 text-[#7b808c]">{t("extract.description")}</p>
          </div>
          {watermarkResult ? (
            <KeyValueGrid
              entries={[
                [t("extract.fields.found"), Boolean(watermarkResult.found)],
                [t("extract.fields.valid"), Boolean(watermarkResult.valid)],
                [t("extract.fields.version"), watermarkResult.version ?? ""],
                [t("extract.fields.traceToken"), watermarkResult.traceToken ?? ""],
                [t("extract.fields.traceType"), watermarkResult.traceType ?? ""],
                [t("extract.fields.traceResolved"), Boolean(watermarkResult.traceResolved)],
                [t("extract.fields.payloadBytes"), watermarkResult.payloadBytes ?? ""],
                [t("extract.fields.payloadBits"), watermarkResult.payloadBits ?? ""],
                [t("extract.fields.payloadFormat"), watermarkResult.payloadFormat ?? ""],
                [t("extract.fields.watermarkEngine"), watermarkResult.watermarkEngine ?? ""],
                [t("extract.fields.uid"), watermarkResult.uid ?? ""],
                [t("extract.fields.userId"), watermarkResult.userId ?? ""],
                [t("extract.fields.username"), watermarkResult.username ?? ""],
                [t("extract.fields.uploadedAt"), watermarkResult.uploadedAt ?? ""],
                [t("extract.fields.sourceHash"), watermarkResult.sourceHash ?? ""],
                [t("extract.fields.customText"), watermarkResult.customText ?? ""],
                [t("extract.fields.postId"), watermarkResult.postId ?? ""],
                [t("extract.fields.imageId"), watermarkResult.imageId ?? ""],
                [t("extract.fields.jobId"), watermarkResult.jobId ?? ""],
                [t("extract.fields.includedFields"), watermarkResult.includedFields?.join(", ") ?? ""],
              ]}
            />
          ) : (
            <EmptyBlock icon={ShieldCheck} label={t("extract.empty")} />
          )}
        </div>
      </Panel>
  );
}

