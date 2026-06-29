import { clsx, type ClassValue } from "clsx";
import type { FFmpeg as FFmpegType } from "@ffmpeg/ffmpeg";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

// Lazily-loaded singleton FFmpeg instance.  A single instance is reused
// across all thumbnail requests so the ~30 MB WASM core is only downloaded
// and compiled once per page load.
let _ffmpeg: FFmpegType | null = null;
let _loadPromise: Promise<FFmpegType | null> | null = null;

async function loadFFmpeg(): Promise<FFmpegType | null> {
  if (_ffmpeg) return _ffmpeg;
  if (_loadPromise) return _loadPromise;

  _loadPromise = (async () => {
    try {
      const { FFmpeg } = await import("@ffmpeg/ffmpeg");
      const { toBlobURL } = await import("@ffmpeg/util");

      const ffmpeg = new FFmpeg();
      await ffmpeg.load({
        coreURL: await toBlobURL(
          "https://unpkg.com/@ffmpeg/core@0.12.6/dist/umd/ffmpeg-core.js",
          "text/javascript",
        ),
        wasmURL: await toBlobURL(
          "https://unpkg.com/@ffmpeg/core@0.12.6/dist/umd/ffmpeg-core.wasm",
          "application/wasm",
        ),
      });

      _ffmpeg = ffmpeg;
      return ffmpeg;
    } catch {
      // Allow retry on next call.
      _loadPromise = null;
      return null;
    }
  })();

  return _loadPromise;
}

/**
 * Generate a thumbnail from a video file using FFmpeg.wasm.
 *
 * FFmpeg extracts the frame at 00:00:01 (falling back to the last frame for
 * very short clips), scales it to 480 px wide, and returns it as a JPEG
 * data-URL so callers can pass it directly to `dataURLtoFile` for upload.
 *
 * The FFmpeg WASM core (~30 MB) is loaded from the unpkg CDN on the first
 * call and cached as a singleton for all subsequent calls.  A 30-second
 * timeout ensures the Promise always resolves – returning `null` on failure
 * so callers degrade gracefully.
 */
export function generateVideoThumbnail(file: File): Promise<string | null> {
  return new Promise((resolve) => {
    let settled = false;

    const finish = (result: string | null) => {
      if (settled) return;
      settled = true;
      clearTimeout(timer);
      resolve(result);
    };

    // 30 s covers the first-ever load of the WASM binary (~20–30 s on a slow
    // connection) as well as the actual extraction step.
    const timer = setTimeout(() => finish(null), 30_000);

    void (async () => {
      try {
        const ffmpeg = await loadFFmpeg();
        if (!ffmpeg) {
          finish(null);
          return;
        }

        const { fetchFile } = await import("@ffmpeg/util");

        // Use a unique suffix so concurrent calls don't clobber each other.
        const uid = crypto.randomUUID().slice(0, 8);
        const inputName = `input_${uid}.mp4`;
        const outputName = `thumb_${uid}.jpg`;

        await ffmpeg.writeFile(inputName, await fetchFile(file));
        await ffmpeg.exec([
          "-i",
          inputName,
          "-ss",
          "00:00:01",
          "-frames:v",
          "1",
          "-vf",
          "scale=480:-1",
          outputName,
        ]);

        const data = await ffmpeg.readFile(outputName);

        // Clean up the virtual filesystem entries.
        await ffmpeg.deleteFile(inputName).catch(() => {});
        await ffmpeg.deleteFile(outputName).catch(() => {});

        const blob = new Blob([new Uint8Array(data as Uint8Array)], {
          type: "image/jpeg",
        });

        // Convert to a data-URL so dataURLtoFile() can consume it.
        const reader = new FileReader();
        reader.onload = () => finish(reader.result as string);
        reader.onerror = () => finish(null);
        reader.readAsDataURL(blob);
      } catch {
        finish(null);
      }
    })();
  });
}

/**
 * Determine whether a File is a video.  On iOS Safari, `file.type` can be an
 * empty string for videos selected from the photo library, so we also check
 * the file extension as a fallback.
 */
export function isVideoFile(file: File): boolean {
  if (file.type.startsWith("video/")) return true;
  if (file.type && !file.type.startsWith("video/")) return false;
  // Empty MIME type: fall back to extension check.
  const ext = file.name.split(".").pop()?.toLowerCase() ?? "";
  return ["mp4", "mov", "m4v", "avi", "mkv", "webm", "3gp", "hevc"].includes(ext);
}

/**
 * Convert a data URL (e.g. from canvas.toDataURL) to a File object so it can
 * be appended to a FormData and uploaded to the server.
 */
export function dataURLtoFile(dataURL: string, filename: string): File {
  const [header, base64] = dataURL.split(",");
  const mime = header.match(/:(.*?);/)?.[1] ?? "image/jpeg";
  const binaryStr = atob(base64);
  const bytes = new Uint8Array(binaryStr.length);
  for (let i = 0; i < binaryStr.length; i++) {
    bytes[i] = binaryStr.charCodeAt(i);
  }
  return new File([bytes], filename, { type: mime });
}
