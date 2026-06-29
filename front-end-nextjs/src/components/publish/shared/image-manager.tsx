"use client";

import { useMemo, type ReactNode } from "react";
import {
  closestCenter,
  DndContext,
  KeyboardSensor,
  PointerSensor,
  TouchSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  arrayMove,
  rectSortingStrategy,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import {
  CircleDollarSign,
  Eye,
  ImageIcon,
  LockKeyhole,
  LockKeyholeOpen,
  RefreshCw,
  ShieldAlert,
  Trash2,
} from "lucide-react";
import { useTranslations } from "next-intl";
import { cn } from "@/lib/utils";
import type { ImageAccessPatch } from "./image-access";

export type ManagedImageItem = {
  error?: string | null;
  id: string;
  isFreePreview: boolean;
  isProtected: boolean;
  name: string;
  previewUrl: string;
  progress?: number;
  size?: number;
  status?: "queued" | "uploading" | "succeeded" | "failed";
};

export function ImageManager({
  disabled = false,
  items,
  onBatchUpdate,
  onOpenItem,
  onRemove,
  onReorder,
  onRetry,
  onSelectionChange,
  paidContentEnabled = true,
  protectionEnabled,
  protectionNoticeEnabled = true,
  selectAllEnabled = true,
  selectedIds,
  variant = "desktop",
}: {
  disabled?: boolean;
  items: ManagedImageItem[];
  onBatchUpdate: (ids: string[], flags: ImageAccessPatch) => void;
  onOpenItem?: (id: string) => void;
  onRemove: (ids: string[]) => void;
  onReorder: (ids: string[]) => void;
  onRetry?: (id: string) => void;
  onSelectionChange: (ids: string[]) => void;
  paidContentEnabled?: boolean;
  protectionEnabled: boolean;
  protectionNoticeEnabled?: boolean;
  selectAllEnabled?: boolean;
  selectedIds: string[];
  variant?: "desktop" | "mobile";
}) {
  const t = useTranslations();
  const selected = useMemo(() => new Set(selectedIds), [selectedIds]);
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 7 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 250, tolerance: 6 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );
  const targetIds = selectedIds.length > 0
    ? selectedIds
    : selectAllEnabled
      ? items.map((item) => item.id)
      : [];
  const coverId = items[0]?.id;
  const nonCoverTargetIds = targetIds.filter((id) => id !== coverId);

  function handleDragEnd(event: DragEndEvent) {
    const activeId = String(event.active.id);
    const overId = event.over ? String(event.over.id) : null;
    if (!overId || activeId === overId) {
      return;
    }
    const oldIndex = items.findIndex((item) => item.id === activeId);
    const newIndex = items.findIndex((item) => item.id === overId);
    if (oldIndex < 0 || newIndex < 0) {
      return;
    }
    onReorder(arrayMove(items, oldIndex, newIndex).map((item) => item.id));
  }

  function toggleSelection(id: string) {
    const next = new Set(selected);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    onSelectionChange(Array.from(next));
  }

  if (items.length === 0) {
    return null;
  }

  return (
    <div
      className={cn(
        "w-full text-left",
        variant === "mobile" &&
          selectAllEnabled &&
          selectedIds.length > 0 &&
          "pb-[calc(148px+env(safe-area-inset-bottom))]",
      )}
    >
      {selectAllEnabled ? (
        <div
          className={cn(
            "z-10 mb-3 flex flex-wrap items-center gap-2 rounded-2xl border border-black/[0.06] bg-white/95 p-2 shadow-sm backdrop-blur",
            variant === "desktop" && "sticky top-2",
            variant === "mobile" && selectedIds.length === 0 && "sticky top-0",
            variant === "mobile" &&
              selectedIds.length > 0 &&
              "fixed inset-x-3 bottom-[calc(10px+env(safe-area-inset-bottom))] z-50 mx-auto max-w-[406px] border-black/10 shadow-xl",
          )}
        >
          <button
            type="button"
            disabled={disabled}
            onClick={() => onSelectionChange(selectedIds.length === items.length ? [] : items.map((item) => item.id))}
            className={cn(
              "rounded-xl border border-black/[0.08] font-bold text-[#55555d] disabled:opacity-50",
              variant === "mobile" ? "h-8 px-2 text-[11px]" : "h-9 px-3 text-xs",
            )}
          >
            {selectedIds.length === items.length
              ? t("publish.imageManager.clearSelection")
              : t("publish.imageManager.selectAll")}
          </button>
          <span className="mr-auto text-xs font-semibold text-[#777780]">
            {t("publish.imageManager.selectionCount", {
              count: selectedIds.length,
              total: items.length,
            })}
          </span>
          <ImagePresetButton
            disabled={disabled || targetIds.length === 0}
            label={t("publish.imageManager.freeDirect")}
            icon={<Eye className="size-4" />}
            compact={variant === "mobile"}
            onClick={() => onBatchUpdate(targetIds, { isFreePreview: true })}
          />
          <ImagePresetButton
            disabled={disabled || !paidContentEnabled || nonCoverTargetIds.length === 0}
            label={t("publish.imageManager.paidDirect")}
            icon={<CircleDollarSign className="size-4" />}
            compact={variant === "mobile"}
            onClick={() => onBatchUpdate(nonCoverTargetIds, { isFreePreview: false })}
          />
          {protectionEnabled ? (
            <>
              <ImagePresetButton
                disabled={disabled || targetIds.length === 0}
                label={t("publish.imageManager.allDirect")}
                icon={<LockKeyholeOpen className="size-4" />}
                compact={variant === "mobile"}
                onClick={() => onBatchUpdate(targetIds, { isProtected: false })}
              />
              <ImagePresetButton
                disabled={disabled || nonCoverTargetIds.length === 0}
                label={t("publish.imageManager.allProtected")}
                icon={<LockKeyhole className="size-4" />}
                compact={variant === "mobile"}
                onClick={() => onBatchUpdate(nonCoverTargetIds, { isProtected: true })}
              />
            </>
          ) : null}
          {selectedIds.length > 0 ? (
            <button
              type="button"
              disabled={disabled}
              onClick={() => onRemove(selectedIds)}
              className={cn(
                "flex items-center gap-1 rounded-xl bg-[#fff0f1] font-bold text-[#d33b4f] disabled:opacity-50",
                variant === "mobile" ? "h-8 px-2 text-[11px]" : "h-9 px-3 text-xs",
              )}
            >
              <Trash2 className="size-3.5" />
              {t("publish.imageManager.deleteSelected")}
            </button>
          ) : null}
        </div>
      ) : null}

      {protectionNoticeEnabled ? (
        <div className={cn(
          "mb-3 flex items-start gap-2 rounded-xl border px-3 py-2 text-[11px] leading-5",
          protectionEnabled
            ? "border-amber-200/80 bg-amber-50 text-amber-900"
            : "border-slate-200 bg-slate-50 text-slate-600",
        )}>
          <ShieldAlert className="mt-0.5 size-4 shrink-0" />
          <span>{t(protectionEnabled
            ? "publish.imageManager.protectionNotice"
            : "publish.imageManager.protectionDisabledNotice")}</span>
        </div>
      ) : null}

      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
        <SortableContext items={items.map((item) => item.id)} strategy={rectSortingStrategy}>
          <div
            className={cn(
              "grid",
              variant === "mobile"
                ? "grid-cols-3 gap-2 min-[390px]:gap-2.5"
                : "grid-cols-[repeat(auto-fill,minmax(min(100%,148px),1fr))] gap-3",
            )}
          >
            {items.map((item, index) => (
              <SortableImageCard
                key={item.id}
                disabled={disabled}
                index={index}
                item={item}
                onOpenItem={onOpenItem ? () => onOpenItem(item.id) : undefined}
                onRemove={() => onRemove([item.id])}
                onRetry={onRetry}
                onToggleFree={() => onBatchUpdate([item.id], {
                  isFreePreview: !item.isFreePreview,
                })}
                onToggleProtection={() => onBatchUpdate([item.id], {
                  isProtected: !item.isProtected,
                })}
                onToggleSelection={() => toggleSelection(item.id)}
                paidContentEnabled={paidContentEnabled}
                protectionEnabled={protectionEnabled}
                selected={selected.has(item.id)}
                variant={variant}
              />
            ))}
          </div>
        </SortableContext>
      </DndContext>
    </div>
  );
}

function SortableImageCard({
  disabled,
  index,
  item,
  onRemove,
  onOpenItem,
  onRetry,
  onToggleFree,
  onToggleProtection,
  onToggleSelection,
  paidContentEnabled,
  protectionEnabled,
  selected,
  variant,
}: {
  disabled: boolean;
  index: number;
  item: ManagedImageItem;
  onRemove: () => void;
  onOpenItem?: () => void;
  onRetry?: (id: string) => void;
  onToggleFree: () => void;
  onToggleProtection: () => void;
  onToggleSelection: () => void;
  paidContentEnabled: boolean;
  protectionEnabled: boolean;
  selected: boolean;
  variant: "desktop" | "mobile";
}) {
  const t = useTranslations();
  const isCover = index === 0;
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: item.id,
    disabled,
  });

  return (
    <article
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      className={cn(
        "group relative min-w-0 overflow-hidden rounded-2xl bg-white shadow-sm ring-1 ring-black/[0.06]",
        selected && "ring-2 ring-primary",
        isDragging && "z-20 scale-[1.02] opacity-80 shadow-xl",
      )}
    >
      <div className={cn("relative bg-[#efeff2]", variant === "mobile" ? "aspect-square" : "aspect-[3/4]")}>
        <button
          type="button"
          aria-label={item.name}
          onClick={onOpenItem ?? onToggleSelection}
          onContextMenu={(event) => event.preventDefault()}
          className="absolute inset-0 size-full touch-manipulation select-none"
          {...attributes}
          {...listeners}
        >
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={item.previewUrl}
            alt={item.name}
            loading="lazy"
            decoding="async"
            draggable={false}
            className="size-full object-cover"
          />
        </button>
        <div className="absolute right-1.5 top-1.5 z-10">
          <button
            type="button"
            disabled={disabled}
            aria-label={t("publish.protection.removeImage")}
            onClick={onRemove}
            className="flex size-8 items-center justify-center rounded-full bg-black/50 text-white hover:bg-[#d33b4f] disabled:opacity-50"
          >
            <Trash2 className="size-4" />
          </button>
        </div>
        <div className="absolute bottom-1.5 left-1.5 right-1.5 z-10 flex min-w-0 items-center gap-1">
          <button
            type="button"
            disabled={disabled || !paidContentEnabled || isCover}
            title={item.isFreePreview ? t("publish.protection.setPaid") : t("publish.protection.setFree")}
            aria-label={item.isFreePreview ? t("publish.protection.setPaid") : t("publish.protection.setFree")}
            onClick={onToggleFree}
            className={cn(
              "flex size-8 shrink-0 items-center justify-center rounded-full text-white shadow-sm backdrop-blur-sm disabled:cursor-not-allowed disabled:opacity-65",
              item.isFreePreview ? "bg-primary/90" : "bg-black/75",
            )}
          >
            {item.isFreePreview ? <Eye className="size-4" /> : <CircleDollarSign className="size-4" />}
          </button>
          <button
            type="button"
            disabled={disabled || !protectionEnabled || isCover}
            title={item.isProtected ? t("publish.protection.setDirect") : t("publish.protection.setProtected")}
            aria-label={item.isProtected ? t("publish.protection.setDirect") : t("publish.protection.setProtected")}
            onClick={onToggleProtection}
            className={cn(
              "flex size-8 shrink-0 items-center justify-center rounded-full text-white shadow-sm backdrop-blur-sm disabled:cursor-not-allowed disabled:opacity-65",
              item.isProtected ? "bg-[#2563eb]/92" : "bg-black/55",
            )}
          >
            {item.isProtected ? <LockKeyhole className="size-4" /> : <LockKeyholeOpen className="size-4" />}
          </button>
        </div>
        {item.status === "uploading" ? (
          <div className="absolute inset-x-2 bottom-9 z-10 rounded-full bg-black/55 p-1">
            <div className="h-1.5 overflow-hidden rounded-full bg-white/20">
              <div className="h-full rounded-full bg-white" style={{ width: `${item.progress ?? 0}%` }} />
            </div>
          </div>
        ) : null}
        {item.status === "failed" ? (
          <div className="absolute inset-0 z-10 flex flex-col items-center justify-center bg-black/65 p-2 text-center text-white">
            <ImageIcon className="size-6" />
            <span className="mt-1 line-clamp-2 text-[10px]">{item.error || t("publish.imageManager.uploadFailed")}</span>
            {onRetry ? (
              <button
                type="button"
                onClick={() => onRetry(item.id)}
                className="mt-2 inline-flex h-8 items-center gap-1 rounded-full bg-white px-3 text-[11px] font-bold text-[#25252b]"
              >
                <RefreshCw className="size-3.5" />
                {t("common.retry")}
              </button>
            ) : null}
          </div>
        ) : null}
      </div>
    </article>
  );
}

function ImagePresetButton({
  compact = false,
  disabled,
  icon,
  label,
  onClick,
}: {
  compact?: boolean;
  disabled: boolean;
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
        "flex items-center justify-center rounded-xl border border-black/[0.08] font-bold text-[#55555d] transition hover:border-primary/35 hover:text-primary disabled:opacity-50",
        compact ? "size-8" : "size-9",
      )}
      title={label}
      aria-label={label}
    >
      {icon}
    </button>
  );
}
