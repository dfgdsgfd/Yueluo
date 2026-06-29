import type { Dispatch, SetStateAction } from "react";
import type { FeedPost, UploadAsset } from "@/lib/types";
import { revokePendingObjectUrls } from "./post-builders";
import { workbenchPostEditState } from "./post-edit-state";
import type {
  ImagePaymentSettings,
  PendingUploadFile,
  PublishMode,
  UploadFailure,
  UploadMode,
  UploadProgressDetailState,
  UploadProgressState,
  Visibility,
  WorkspaceSection,
} from "./workbench-config";

type WorkbenchRestoreOptions = {
  currentDraftId: string | number | null;
  pendingFiles: Record<UploadMode, PendingUploadFile[]>;
  post: FeedPost;
  setActiveSection: Dispatch<SetStateAction<WorkspaceSection>>;
  setArticleComposerOpen?: Dispatch<SetStateAction<boolean>>;
  setBody: Dispatch<SetStateAction<string>>;
  setCurrentDraftId: Dispatch<SetStateAction<string | number | null>>;
  setDraftsOpen: Dispatch<SetStateAction<boolean>>;
  setEditingPostId: Dispatch<SetStateAction<string | number | null>>;
  setImagePaymentSettings: Dispatch<SetStateAction<ImagePaymentSettings>>;
  setMode: Dispatch<SetStateAction<PublishMode>>;
  setPendingFiles: Dispatch<SetStateAction<Record<UploadMode, PendingUploadFile[]>>>;
  setSelectedCategoryId: Dispatch<SetStateAction<number | null>>;
  setTags: Dispatch<SetStateAction<string>>;
  setTitle: Dispatch<SetStateAction<string>>;
  setUploadedAssets: Dispatch<SetStateAction<Record<UploadMode, UploadAsset[]>>>;
  setUploadFailures: Dispatch<SetStateAction<Record<UploadMode, UploadFailure[]>>>;
  setUploadingMode: Dispatch<SetStateAction<UploadMode | null>>;
  setUploadProgress: Dispatch<SetStateAction<UploadProgressState>>;
  setUploadProgressDetails: Dispatch<SetStateAction<UploadProgressDetailState>>;
  setVisibility: Dispatch<SetStateAction<Visibility>>;
};

export function restoreWorkbenchDraftState(options: WorkbenchRestoreOptions) {
  applyWorkbenchPostState(options);
  options.setCurrentDraftId(options.currentDraftId);
  options.setEditingPostId(null);
}

export function restoreWorkbenchEditingState(options: WorkbenchRestoreOptions) {
  applyWorkbenchPostState(options);
  options.setCurrentDraftId(null);
  options.setEditingPostId(options.post.id);
  options.setArticleComposerOpen?.(false);
}

function applyWorkbenchPostState({
  pendingFiles,
  post,
  setActiveSection,
  setBody,
  setDraftsOpen,
  setImagePaymentSettings,
  setMode,
  setPendingFiles,
  setSelectedCategoryId,
  setTags,
  setTitle,
  setUploadedAssets,
  setUploadFailures,
  setUploadingMode,
  setUploadProgress,
  setUploadProgressDetails,
  setVisibility,
}: WorkbenchRestoreOptions) {
  const restored = workbenchPostEditState(post);

  revokePendingObjectUrls(pendingFiles);
  setMode(restored.mode);
  setTitle(restored.title);
  setBody(restored.body);
  setTags(restored.tags);
  setVisibility(restored.visibility);
  setSelectedCategoryId(restored.categoryId);
  setUploadedAssets(restored.assets);
  setImagePaymentSettings(restored.paymentSettings);
  setPendingFiles({ image: [], video: [], podcast: [] });
  setUploadFailures({ image: [], video: [], podcast: [] });
  setUploadProgress({ image: null, video: null, podcast: null });
  setUploadProgressDetails({ image: null, video: null, podcast: null });
  setUploadingMode(null);
  setActiveSection("publish");
  setDraftsOpen(false);
}
