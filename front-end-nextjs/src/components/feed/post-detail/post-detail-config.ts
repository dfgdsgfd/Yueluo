export const defaultPostSlideshowMaxImages = 25;

export function getPostSlideshowMaxImages(value = process.env.NEXT_PUBLIC_POST_SLIDESHOW_MAX_IMAGES) {
  const parsed = Number(value);
  if (!Number.isInteger(parsed) || parsed < 1) {
    return defaultPostSlideshowMaxImages;
  }
  return Math.min(parsed, 500);
}
