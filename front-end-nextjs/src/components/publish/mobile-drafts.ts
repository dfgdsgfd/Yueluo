"use client";

import type { Category } from "@/lib/types";

export type MobilePublishVisibility = "public" | "followers" | "private";

export type MobilePublishDraftMedia = {
  file: File;
  id: string;
  isFreePreview?: boolean;
  isProtected?: boolean;
  kind: "image" | "video";
  name: string;
  previewDataUrl?: string;
};

export type MobilePublishDraftAttachment = {
  file: File;
  name: string;
  size: number;
  type: string;
};

export type MobilePublishDraft = {
  attachment: MobilePublishDraftAttachment | null;
  board: string;
  body: string;
  id: string;
  imagePaymentMethod?: "balance" | "points";
  imagePrice?: string;
  mediaAssets: MobilePublishDraftMedia[];
  savedAt: string;
  selectedCategoryId: number | null;
  title: string;
  topic: string;
  visibility: MobilePublishVisibility;
};

export const localDraftsKey = "yuem_mobile_publish_drafts";
export const draftRestoreKey = "yuem_mobile_publish_restore_draft";

const draftDbName = "yuem_mobile_publish_drafts_db";
const draftDbVersion = 1;
const draftStoreName = "drafts";
const legacyDraftsMigratedKey = "yuem_mobile_publish_drafts_migrated";

export async function readLocalDrafts(): Promise<MobilePublishDraft[]> {
  if (!canUseIndexedDb()) {
    return readLegacyDrafts();
  }

  const db = await openDraftDb();
  try {
    await migrateLegacyDrafts(db);
    const drafts = await getAllDrafts(db);
    return drafts.sort(compareDraftsBySavedAt);
  } finally {
    db.close();
  }
}

export async function saveLocalDraft(draft: MobilePublishDraft) {
  if (!canUseIndexedDb()) {
    const nextDrafts = [draft, ...readLegacyDrafts()].slice(0, 20);
    writeLegacyDrafts(nextDrafts);
    return nextDrafts;
  }

  const db = await openDraftDb();
  try {
    await putDraft(db, draft);
  } finally {
    db.close();
  }

  const drafts = await readLocalDrafts();
  const nextDrafts = drafts.slice(0, 20);
  if (drafts.length > nextDrafts.length) {
    await writeLocalDrafts(nextDrafts);
  }

  return nextDrafts;
}

export async function writeLocalDrafts(drafts: MobilePublishDraft[]) {
  if (!canUseIndexedDb()) {
    writeLegacyDrafts(drafts);
    return;
  }

  const db = await openDraftDb();
  const tx = db.transaction(draftStoreName, "readwrite");
  try {
    const store = tx.objectStore(draftStoreName);
    store.clear();
    for (const draft of drafts) {
      store.put(draft);
    }
    await waitForTransaction(tx);
  } finally {
    db.close();
  }
}

export async function deleteLocalDraft(id: string) {
  if (!canUseIndexedDb()) {
    writeLegacyDrafts(readLegacyDrafts().filter((draft) => draft.id !== id));
    return;
  }

  const db = await openDraftDb();
  const tx = db.transaction(draftStoreName, "readwrite");
  try {
    tx.objectStore(draftStoreName).delete(id);
    await waitForTransaction(tx);
  } finally {
    db.close();
  }
}

export function writePendingRestoreDraft(draft: MobilePublishDraft) {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.setItem(draftRestoreKey, draft.id);
}

export async function consumePendingRestoreDraft(): Promise<MobilePublishDraft | null> {
  if (typeof window === "undefined") {
    return null;
  }

  const draftId = window.localStorage.getItem(draftRestoreKey);
  window.localStorage.removeItem(draftRestoreKey);
  if (!draftId) {
    return null;
  }

  const drafts = await readLocalDrafts();
  return drafts.find((draft) => draft.id === draftId) ?? null;
}

export function getDraftTitle(draft: MobilePublishDraft, fallback: string) {
  return draft.title.trim() || draft.body.trim() || fallback;
}

export function formatDraftTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString();
}

export function getCategoryDisplayName(category: Pick<Category, "category_title" | "display_name" | "name">) {
  return translateCategoryName(category.display_name?.trim() || category.category_title?.trim() || category.name);
}

export function translateCategoryName(name: string) {
  const normalizedName = name.trim().replace(/\s+/g, " ");
  if (!normalizedName) {
    return "";
  }

  return normalizedName;
}

function canUseIndexedDb() {
  return typeof window !== "undefined" && "indexedDB" in window;
}

function openDraftDb() {
  return new Promise<IDBDatabase>((resolve, reject) => {
    const request = window.indexedDB.open(draftDbName, draftDbVersion);

    request.onupgradeneeded = () => {
      const db = request.result;
      if (!db.objectStoreNames.contains(draftStoreName)) {
        db.createObjectStore(draftStoreName, { keyPath: "id" });
      }
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error ?? new Error("draft.storage_open_failed"));
  });
}

async function migrateLegacyDrafts(db: IDBDatabase) {
  if (typeof window === "undefined" || window.localStorage.getItem(legacyDraftsMigratedKey)) {
    return;
  }

  const legacyDrafts = readLegacyDrafts();
  if (legacyDrafts.length > 0) {
    const tx = db.transaction(draftStoreName, "readwrite");
    const store = tx.objectStore(draftStoreName);
    for (const draft of legacyDrafts) {
      store.put(draft);
    }
    await waitForTransaction(tx);
  }

  window.localStorage.setItem(legacyDraftsMigratedKey, "1");
}

function getAllDrafts(db: IDBDatabase) {
  return new Promise<MobilePublishDraft[]>((resolve, reject) => {
    const request = db.transaction(draftStoreName, "readonly").objectStore(draftStoreName).getAll();
    request.onsuccess = () => {
      const drafts = request.result
        .map((value) => normalizeDraft(value))
        .filter((draft): draft is MobilePublishDraft => Boolean(draft));
      resolve(drafts);
    };
    request.onerror = () => reject(request.error ?? new Error("draft.read_failed"));
  });
}

function putDraft(db: IDBDatabase, draft: MobilePublishDraft) {
  const tx = db.transaction(draftStoreName, "readwrite");
  tx.objectStore(draftStoreName).put(draft);
  return waitForTransaction(tx);
}

function waitForTransaction(tx: IDBTransaction) {
  return new Promise<void>((resolve, reject) => {
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error ?? new Error("draft.storage_failed"));
    tx.onabort = () => reject(tx.error ?? new Error("draft.storage_aborted"));
  });
}

function readLegacyDrafts() {
  if (typeof window === "undefined") {
    return [];
  }

  try {
    const raw = window.localStorage.getItem(localDraftsKey);
    const parsed = raw ? JSON.parse(raw) : [];
    if (!Array.isArray(parsed)) {
      return [];
    }

    return parsed
      .map((value) => normalizeDraft(value))
      .filter((draft): draft is MobilePublishDraft => Boolean(draft))
      .sort(compareDraftsBySavedAt);
  } catch {
    return [];
  }
}

function writeLegacyDrafts(drafts: MobilePublishDraft[]) {
  if (typeof window === "undefined") {
    return;
  }

  const metadataOnlyDrafts = drafts.map((draft) => ({
    ...draft,
    attachment: null,
    mediaAssets: [],
  }));
  window.localStorage.setItem(localDraftsKey, JSON.stringify(metadataOnlyDrafts));
}

function normalizeDraft(value: unknown): MobilePublishDraft | null {
  if (!value || typeof value !== "object") {
    return null;
  }

  const draft = value as Partial<MobilePublishDraft>;
  if (
    typeof draft.id !== "string" ||
    typeof draft.savedAt !== "string" ||
    typeof draft.title !== "string" ||
    typeof draft.body !== "string" ||
    typeof draft.topic !== "string" ||
    typeof draft.board !== "string" ||
    !isVisibility(draft.visibility)
  ) {
    return null;
  }

  return {
    attachment: normalizeAttachment(draft.attachment),
    board: draft.board,
    body: draft.body,
    id: draft.id,
    imagePaymentMethod:
      draft.imagePaymentMethod === "points" || draft.imagePaymentMethod === "balance"
        ? draft.imagePaymentMethod
        : undefined,
    imagePrice: typeof draft.imagePrice === "string" ? draft.imagePrice : undefined,
    mediaAssets: Array.isArray(draft.mediaAssets) ? draft.mediaAssets.map(normalizeMedia).filter((media): media is MobilePublishDraftMedia => Boolean(media)) : [],
    savedAt: draft.savedAt,
    selectedCategoryId: typeof draft.selectedCategoryId === "number" ? draft.selectedCategoryId : null,
    title: draft.title,
    topic: draft.topic,
    visibility: draft.visibility,
  };
}

function normalizeMedia(value: unknown): MobilePublishDraftMedia | null {
  if (!value || typeof value !== "object") {
    return null;
  }

  const media = value as Partial<MobilePublishDraftMedia>;
  if (
    typeof media.id !== "string" ||
    typeof media.name !== "string" ||
    (media.kind !== "image" && media.kind !== "video") ||
    !isStoredFile(media.file)
  ) {
    return null;
  }

  return {
    file: media.file,
    id: media.id,
    isFreePreview: typeof media.isFreePreview === "boolean" ? media.isFreePreview : undefined,
    isProtected: typeof media.isProtected === "boolean" ? media.isProtected : undefined,
    kind: media.kind,
    name: media.name,
    previewDataUrl: typeof media.previewDataUrl === "string" ? media.previewDataUrl : undefined,
  };
}

function normalizeAttachment(value: unknown): MobilePublishDraftAttachment | null {
  if (!value || typeof value !== "object") {
    return null;
  }

  const attachment = value as Partial<MobilePublishDraftAttachment>;
  if (
    typeof attachment.name !== "string" ||
    typeof attachment.size !== "number" ||
    typeof attachment.type !== "string" ||
    !isStoredFile(attachment.file)
  ) {
    return null;
  }

  return {
    file: attachment.file,
    name: attachment.name,
    size: attachment.size,
    type: attachment.type,
  };
}

function isStoredFile(value: unknown): value is File {
  return typeof File !== "undefined" && value instanceof File;
}

function isVisibility(value: unknown): value is MobilePublishVisibility {
  return value === "public" || value === "followers" || value === "private";
}

function compareDraftsBySavedAt(a: MobilePublishDraft, b: MobilePublishDraft) {
  return new Date(b.savedAt).getTime() - new Date(a.savedAt).getTime();
}
