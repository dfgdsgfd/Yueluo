"use client";

import { useMemo, useState, type ReactNode } from "react";
import { CircleDollarSign, Eye, LockKeyhole, LockKeyholeOpen, Plus, Shield, X } from "lucide-react";
import { useTranslations } from "next-intl";
import { cn } from "@/lib/utils";
import {
  ImageManager,
  type ManagedImageItem,
} from "../shared/image-manager";
import type { ImageAccessPatch } from "../shared/image-access";
import type { MobilePaymentMethod } from "./mobile-publish-config";

export function MobileImageManager({
  disabled,
  imageLimit,
  imagePaymentMethod,
  imagePrice,
  items,
  mediaCount,
  mediaHelpText,
  onAddMedia,
  onBatchUpdate,
  onPaymentMethodChange,
  onPriceChange,
  onRemove,
  onReorder,
  onRetry,
  onSelectionChange,
  paidImageCount,
  paidContentEnabled,
  paymentMethodsEnabled,
  paymentMaxPrices,
  protectionEnabled,
  protectionNoticeEnabled,
  selectAllEnabled,
  selectedIds,
  showAddButton,
}: {
  disabled: boolean;
  imageLimit: number;
  imagePaymentMethod: "balance" | "points";
  imagePrice: string;
  items: ManagedImageItem[];
  mediaCount: number;
  mediaHelpText: string;
  onAddMedia: () => void;
  onBatchUpdate: (
    ids: string[],
    flags: ImageAccessPatch,
  ) => void;
  onPaymentMethodChange: (method: MobilePaymentMethod) => void;
  onPriceChange: (price: string) => void;
  onRemove: (ids: string[]) => void;
  onReorder: (ids: string[]) => void;
  onRetry: (id: string) => void;
  onSelectionChange: (ids: string[]) => void;
  paidImageCount: number;
  paidContentEnabled: boolean;
  paymentMethodsEnabled: Record<"balance" | "points", boolean>;
  paymentMaxPrices: Record<MobilePaymentMethod, number>;
  protectionEnabled: boolean;
  protectionNoticeEnabled: boolean;
  selectAllEnabled: boolean;
  selectedIds: string[];
  showAddButton: boolean;
}) {
  const t = useTranslations();
  const [editingImageId, setEditingImageId] = useState<string | null>(null);
  const editingImage = useMemo(
    () => items.find((item) => item.id === editingImageId) ?? null,
    [editingImageId, items],
  );
  const editingIsCover = editingImage?.id === items[0]?.id;

  return (
    <>
      <div className="mt-10">
        {items.length > 0 ? (
          <ImageManager
            variant="mobile"
            disabled={disabled}
            items={items}
            protectionEnabled={protectionEnabled}
            protectionNoticeEnabled={protectionNoticeEnabled}
            selectAllEnabled={selectAllEnabled}
            paidContentEnabled={paidContentEnabled}
            selectedIds={selectedIds}
            onOpenItem={setEditingImageId}
            onSelectionChange={onSelectionChange}
            onBatchUpdate={onBatchUpdate}
            onRemove={onRemove}
            onReorder={onReorder}
            onRetry={onRetry}
          />
        ) : null}

        {showAddButton && mediaCount < imageLimit ? (
          <button
            type="button"
            onClick={onAddMedia}
            disabled={disabled}
            className={cn(
              "mt-3 flex min-h-20 w-full items-center justify-center gap-3 rounded-[14px] border border-dashed border-[var(--mobile-publish-border)] bg-[var(--mobile-publish-card)] px-4 text-[var(--mobile-publish-accent-strong)] active:bg-[var(--mobile-publish-accent-soft)] disabled:opacity-60",
              mediaCount === 0 && "min-h-[118px] flex-col",
            )}
          >
            <Plus className={cn("shrink-0", mediaCount === 0 ? "size-12" : "size-7")} strokeWidth={2} />
            <span className="text-center">
              <span className="block text-[16px] font-bold leading-5">
                {t("publish.imageManager.addMedia")}
              </span>
              <span className="mt-1 block text-[13px] font-semibold text-[var(--mobile-publish-muted)]">
                {mediaHelpText}
              </span>
            </span>
          </button>
        ) : null}
      </div>

      {paidImageCount > 0 ? (
        <section className="mt-5 rounded-[16px] bg-[var(--mobile-publish-card)] p-4 shadow-[var(--mobile-publish-shadow)]">
          <div className="flex items-center justify-between gap-3">
            <div className="min-w-0">
              <h2 className="text-[15px] font-black text-[var(--mobile-publish-heading)]">
                {t("publish.protection.paymentTitle", { count: paidImageCount })}
              </h2>
              <p className="mt-1 text-xs leading-5 text-[var(--mobile-publish-muted)]">
                {t("publish.imageManager.paymentMethodHint")}
              </p>
            </div>
            <Shield className="size-6 shrink-0 text-[var(--mobile-publish-accent-strong)]" />
          </div>
          <div className="mt-3 grid grid-cols-2 gap-2">
            {(["balance", "points"] as const).map((method) => (
              <button
                key={method}
                type="button"
                onClick={() => onPaymentMethodChange(method)}
                disabled={!paymentMethodsEnabled[method]}
                className={cn(
                  "h-10 rounded-xl border text-sm font-bold disabled:opacity-40",
                  imagePaymentMethod === method
                    ? "border-[var(--mobile-publish-accent)] bg-[var(--mobile-publish-accent-soft)] text-[var(--mobile-publish-accent-strong)]"
                    : "border-[var(--mobile-publish-border-soft)] text-[var(--mobile-publish-muted)]",
                )}
              >
                {t(`publish.protection.${method}`)}
              </button>
            ))}
          </div>
          <label className="mt-3 block">
            <span className="mb-1 block text-xs font-bold text-[var(--mobile-publish-muted)]">
              {t("publish.protection.priceLabel")}
            </span>
            <input
              value={imagePrice}
              inputMode="decimal"
              max={paymentMaxPrices[imagePaymentMethod]}
              onChange={(event) => onPriceChange(event.target.value.replace(/[^\d.]/g, "").slice(0, 8))}
              className="h-11 w-full rounded-xl border border-[var(--mobile-publish-border-soft)] bg-[var(--mobile-publish-surface)] px-3 text-base font-bold text-[var(--mobile-publish-input)] outline-none focus:border-[var(--mobile-publish-accent)]"
            />
          </label>
        </section>
      ) : null}

      {editingImage ? (
        <div className="fixed inset-0 z-[70] flex items-end justify-center">
          <button
            type="button"
            aria-label={t("common.close")}
            onClick={() => setEditingImageId(null)}
            className="absolute inset-0 bg-black/35"
          />
          <section className="relative z-10 w-full max-w-[430px] rounded-t-[24px] bg-[var(--mobile-publish-surface)] px-4 pb-[calc(18px+env(safe-area-inset-bottom))] pt-4 shadow-2xl">
            <div className="flex items-center gap-3">
              <div className="min-w-0 flex-1">
                <h2 className="text-[18px] font-black text-[var(--mobile-publish-heading)]">
                  {t("publish.imageManager.editImage")}
                </h2>
              </div>
              <button
                type="button"
                aria-label={t("common.close")}
                onClick={() => setEditingImageId(null)}
                className="flex size-9 items-center justify-center rounded-full bg-[var(--mobile-publish-card)] text-[var(--mobile-publish-muted)]"
              >
                <X className="size-5" />
              </button>
            </div>
            <div className="mt-4 space-y-3">
              <div className="grid grid-cols-2 gap-2">
              <MobileImagePreset
                active={editingImage.isFreePreview}
                disabled={editingIsCover}
                icon={<Eye className="size-5" />}
                label={t("publish.protection.badgeFree")}
                onClick={() => onBatchUpdate([editingImage.id], { isFreePreview: true })}
              />
              <MobileImagePreset
                active={!editingImage.isFreePreview}
                disabled={editingIsCover}
                icon={<CircleDollarSign className="size-5" />}
                label={t("publish.protection.badgePaid")}
                onClick={() => onBatchUpdate([editingImage.id], { isFreePreview: false })}
              />
              </div>
              {protectionEnabled ? (
                <div className="grid grid-cols-2 gap-2">
                  <MobileImagePreset
                    active={!editingImage.isProtected}
                    disabled={editingIsCover}
                    icon={<LockKeyholeOpen className="size-5" />}
                    label={t("publish.protection.setDirect")}
                    onClick={() => onBatchUpdate([editingImage.id], { isProtected: false })}
                  />
                  <MobileImagePreset
                    active={editingImage.isProtected}
                    disabled={editingIsCover}
                    icon={<LockKeyhole className="size-5" />}
                    label={t("publish.protection.setProtected")}
                    onClick={() => onBatchUpdate([editingImage.id], { isProtected: true })}
                  />
                </div>
              ) : null}
              {editingIsCover ? (
                <p className="text-xs font-semibold leading-5 text-[var(--mobile-publish-muted)]">
                  {t("publish.imageManager.coverNotice")}
                </p>
              ) : null}
            </div>
          </section>
        </div>
      ) : null}
    </>
  );
}

function MobileImagePreset({
  active,
  disabled = false,
  icon,
  label,
  onClick,
}: {
  active: boolean;
  disabled?: boolean;
  icon: ReactNode;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className={cn(
        "flex h-12 items-center justify-center gap-2 rounded-xl border text-sm font-black disabled:cursor-not-allowed disabled:opacity-45",
        active
          ? "border-[var(--mobile-publish-accent)] bg-[var(--mobile-publish-accent-soft)] text-[var(--mobile-publish-accent-strong)]"
          : "border-[var(--mobile-publish-border-soft)] bg-[var(--mobile-publish-card)] text-[var(--mobile-publish-muted)]",
      )}
    >
      {icon}
      <span>{label}</span>
    </button>
  );
}
