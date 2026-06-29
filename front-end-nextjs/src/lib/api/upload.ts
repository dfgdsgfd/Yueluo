import type {
  UploadAsset
} from "../types";
import {
  ApiUploadOptions,
  apiUpload,
  emitUploadProgress,
  shouldUseChunkUpload,
  uploadFileInChunks,
  uploadProgressWithStage
} from "./core";

export type ApkUploadPayload = {
  originalname?: string;
  size?: number;
  url: string;
};


export async function uploadAdminApk(file: File, options: ApiUploadOptions = {}) {
  const uploadOptions: ApiUploadOptions = { ...options, auth: options.auth ?? true };
  if (await shouldUseChunkUpload(file, "apk", uploadOptions)) {
    return uploadFileInChunks(file, "/api/upload/chunk/merge/apk", uploadOptions);
  }

  const form = new FormData();
  form.append("file", file);
  return apiUpload<ApkUploadPayload>("/api/upload/apk", form, uploadOptions);
}


export async function uploadImage(file: File, options?: ApiUploadOptions) {
  if (options?.purpose === "avatar" || options?.purpose === "background") {
    const formData = new FormData();
    formData.set("file", file);
    return apiUpload<UploadAsset>(
      options.purpose === "avatar" ? "/api/users/me/avatar" : "/api/users/me/banner",
      formData,
      options,
    );
  }
  if (await shouldUseChunkUpload(file, "image", options)) {
    return uploadFileInChunks(file, "/api/upload/chunk/merge/image", options);
  }

  const formData = new FormData();
  formData.set("file", file);
  if (options?.purpose) {
    formData.set("purpose", options.purpose);
  }
  return apiUpload<UploadAsset>("/api/upload/single", formData, options);
}


export async function uploadImages(files: File[], options?: ApiUploadOptions) {
  const result = await uploadImageBatch(files, options);
  return result.uploaded;
}

export type UploadImageBatchFailure = {
  batchIndex: number;
  error: string;
  file: string;
};

export type UploadImageBatchResult = {
  uploaded: UploadAsset[];
  errors: UploadImageBatchFailure[];
};

export async function uploadImageBatch(files: File[], options?: ApiUploadOptions): Promise<UploadImageBatchResult> {
  if (files.length <= 1) {
    const asset = files[0] ? await uploadImage(files[0], options) : null;
    return { uploaded: asset ? [{ ...asset, batchIndex: 0 }] : [], errors: [] };
  }

  const uploadableFiles = files;
  const uploaded: UploadAsset[] = [];
  const errors: UploadImageBatchFailure[] = [];
  for (let index = 0; index < uploadableFiles.length; index += 1) {
    const file = uploadableFiles[index];
    try {
      const asset = await uploadImage(file, {
        ...options,
        onProgress: (progress) => {
          const percent = typeof progress.percent === "number"
            ? Math.round(((index + progress.percent / 100) / uploadableFiles.length) * 100)
            : undefined;
          emitUploadProgress(options, { ...progress, fileName: progress.fileName ?? file.name, percent });
        },
      });
      uploaded.push({ ...asset, batchIndex: index });
    } catch (error) {
      errors.push({
        batchIndex: index,
        error: error instanceof Error ? error.message : "Upload failed",
        file: file.name,
      });
    }
  }
  return { uploaded, errors };
}


export async function uploadVideo(file: File, thumbnail?: File, options?: ApiUploadOptions) {
  if (await shouldUseChunkUpload(file, "video", options)) {
    const asset = await uploadFileInChunks(file, "/api/upload/chunk/merge", thumbnail
      ? {
          ...options,
          onProgress: (progress) => {
            emitUploadProgress(options, {
              ...progress,
              percent: typeof progress.percent === "number"
                ? Math.min(96, Math.round(progress.percent * 0.96))
                : progress.percent,
            });
          },
        }
      : options);
    if (!thumbnail) {
      return asset;
    }

    emitUploadProgress(options, {
      fileName: thumbnail.name,
      loaded: 0,
      message: "Uploading cover",
      percent: 96,
      stage: "thumbnail",
      total: thumbnail.size,
    });
    const cover = await uploadImage(thumbnail, {
      ...options,
      onProgress: (progress) => {
        const coverPercent = typeof progress.percent === "number"
          ? 96 + Math.round(progress.percent * 0.04)
          : undefined;
        emitUploadProgress(options, {
          ...uploadProgressWithStage(progress, "thumbnail", "Uploading cover"),
          percent: coverPercent,
        });
      },
    });

    emitUploadProgress(options, {
      fileName: file.name,
      loaded: file.size,
      message: "Upload complete",
      percent: 100,
      stage: "complete",
      total: file.size,
    });

    return {
      ...asset,
      coverUrl: cover.url,
      coverSignedUrl: cover.signedUrl ?? cover.url,
    };
  }

  const formData = new FormData();
  formData.set("file", file);
  if (thumbnail) {
    formData.set("thumbnail", thumbnail);
  }
  return apiUpload<UploadAsset>("/api/upload/video", formData, options);
}


export async function uploadAttachment(file: File, options?: ApiUploadOptions) {
  const formData = new FormData();
  formData.set("file", file);
  return apiUpload<UploadAsset>("/api/upload/attachment", formData, options);
}


export async function uploadAdminMedia(file: File, options: ApiUploadOptions = {}) {
  const formData = new FormData();
  formData.set("file", file);
  return apiUpload<UploadAsset>("/api/admin/media-library", formData, {
    ...options,
    auth: options.auth ?? true,
  });
}
