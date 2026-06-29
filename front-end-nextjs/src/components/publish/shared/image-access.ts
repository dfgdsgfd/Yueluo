export type ImageAccessFlags = {
  isFreePreview: boolean;
  isProtected: boolean;
};

export type ImageAccessPatch = Partial<ImageAccessFlags>;

export function enforceImageCoverPolicy<
  T extends { isFreePreview?: boolean; isProtected?: boolean },
>(items: T[]): Array<T & ImageAccessFlags> {
  return items.map((item, index) => ({
    ...item,
    isFreePreview: index === 0 ? true : (item.isFreePreview ?? true),
    isProtected: index === 0 ? false : (item.isProtected ?? false),
  }));
}

export function applyImageAccessPatch<T extends { id: string; isFreePreview?: boolean; isProtected?: boolean }>(
  items: T[],
  ids: Iterable<string>,
  patch: ImageAccessPatch,
) {
  const selected = new Set(ids);
  return enforceImageCoverPolicy(items.map((item) => (
    selected.has(item.id) ? { ...item, ...patch } : item
  )));
}
