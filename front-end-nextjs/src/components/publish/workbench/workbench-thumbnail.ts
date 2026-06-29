import type { Dispatch, SetStateAction } from "react";
import type { PendingUploadFile, UploadMode } from "./workbench-config";

export function replaceWorkbenchThumbnail(
  setPendingFiles: Dispatch<SetStateAction<Record<UploadMode, PendingUploadFile[]>>>,
  dataUrl: string,
) {
  setPendingFiles((prev) => {
    if (prev.video.length === 0) {
      return prev;
    }
    const [first, ...rest] = prev.video;
    return { ...prev, video: [{ ...first, thumbnailDataUrl: dataUrl }, ...rest] };
  });
}
