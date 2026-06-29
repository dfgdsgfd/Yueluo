import type { UserSearchResult } from "@/lib/types";
import type { useTranslations } from "next-intl";
import { toast } from "sonner";
import { saveLocalDraft, type MobilePublishDraft } from "../mobile-drafts";
import type { MobilePublishControllerState } from "./mobile-publish-controller-state";
import { appendMobileCustomTopic } from "./mobile-publish-config";
import { nextMobileVisibility } from "./mobile-publish-submit";
import { assetToDraftMedia, fileToDraftAttachment, getDisplayName } from "./mobile-publish-utils";

type MobilePublishDraftActionsOptions = {
  draftableContent: boolean;
  router: { push(href: string): void };
  state: MobilePublishControllerState;
  t: ReturnType<typeof useTranslations>;
};

export function createMobilePublishDraftActions({
  draftableContent,
  router,
  state,
  t,
}: MobilePublishDraftActionsOptions) {
  const {
    attachmentFile,
    board,
    body,
    currentDraftId,
    customTopicInput,
    imagePaymentMethod,
    imagePrice,
    mediaAssets,
    selectedCategoryId,
    setBody,
    setCurrentDraftId,
    setCustomTopicInput,
    setMentionKeyword,
    setShowCloseConfirm,
    setShowMentionSheet,
    setShowTopicSheet,
    setTopic,
    setVisibility,
    title,
    topic,
    visibility,
  } = state;

  function cycleVisibility() {
    setVisibility((current) => nextMobileVisibility(current));
  }

  async function openDraftPage() {
    const saved = await saveCurrentDraft({ silent: true });
    if (saved) {
      toast.success(t("publish.mobile.savedToDrafts"));
    }
    router.push("/publish/mobile/drafts");
  }

  function handleCloseClick() {
    if (draftableContent) {
      setShowCloseConfirm(true);
      return;
    }
    router.push("/");
  }

  async function saveAndClose() {
    const saved = await saveCurrentDraft({ silent: true });
    if (saved) {
      toast.success(t("publish.mobile.draftSaved"));
    }
    router.push("/");
  }

  function closeWithoutSaving() {
    router.push("/");
  }

  async function saveCurrentDraft(options?: { silent?: boolean }) {
    if (!draftableContent) {
      return false;
    }
    const draft: MobilePublishDraft = {
      attachment: attachmentFile ? fileToDraftAttachment(attachmentFile) : null,
      board,
      body,
      id: currentDraftId ?? crypto.randomUUID(),
      imagePaymentMethod,
      imagePrice,
      mediaAssets: mediaAssets.flatMap((asset) => {
        const media = assetToDraftMedia(asset);
        return media ? [media] : [];
      }),
      savedAt: new Date().toISOString(),
      selectedCategoryId,
      title,
      topic,
      visibility,
    };
    await saveLocalDraft(draft);
    setCurrentDraftId(draft.id);
    if (!options?.silent) {
      toast.success(t("publish.mobile.draftSaved"));
    }
    return true;
  }

  function insertMention(user: UserSearchResult) {
    const name = getDisplayName(user);
    const mention = `@${name} `;
    const prefix = body && !/\s$/.test(body) ? " " : "";
    setBody((currentBody) => `${currentBody}${prefix}${mention}`);
    setShowMentionSheet(false);
    setMentionKeyword("");
  }

  function insertEmoji(emoji: string) {
    setBody((currentBody) => `${currentBody}${emoji}`);
  }

  function submitCustomTopic() {
    const nextTopic = appendMobileCustomTopic(topic, customTopicInput);
    if (nextTopic.reason === "required") {
      toast.error(t("publish.mobile.topicRequired"));
      return;
    }
    if (nextTopic.reason === "limit") {
      toast.error(t("publish.mobile.topicLimit", { count: 12 }));
      return;
    }
    setTopic(nextTopic.value);
    setCustomTopicInput("");
    setShowTopicSheet(false);
  }

  return {
    closeWithoutSaving,
    cycleVisibility,
    handleCloseClick,
    insertEmoji,
    insertMention,
    openDraftPage,
    saveAndClose,
    saveCurrentDraft,
    submitCustomTopic,
  };
}
