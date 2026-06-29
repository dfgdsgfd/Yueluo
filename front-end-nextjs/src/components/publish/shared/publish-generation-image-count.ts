export function countPublishGenerationSelectableImages(imageCount: number, maxImages: number | null | undefined) {
  const normalizedCount = Math.max(0, Math.floor(Number.isFinite(imageCount) ? imageCount : 0));
  const normalizedMax = Math.floor(Number(maxImages ?? 3));
  if (!Number.isFinite(normalizedMax) || normalizedMax <= 0) {
    return 0;
  }
  return Math.min(normalizedCount, normalizedMax);
}
