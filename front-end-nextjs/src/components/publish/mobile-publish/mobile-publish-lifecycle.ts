import {
  getPostDetail,
  getPostProtectionConfig,
  getStoredAccessToken,
  listMentionUsers,
  searchUsers,
} from "@/lib/api";
import type { AuthUser } from "@/lib/types";
import type { useTranslations } from "next-intl";
import type { MutableRefObject } from "react";
import { useEffect } from "react";
import { toast } from "sonner";
import { consumePendingRestoreDraft, translateCategoryName } from "../mobile-drafts";
import { enforceImageCoverPolicy } from "../shared/image-access";
import type { MobilePublishControllerState } from "./mobile-publish-controller-state";
import { loadMobilePublishData } from "./mobile-publish-bootstrap";
import { defaultImageLimit, defaultMobilePaymentMaxPrices } from "./mobile-publish-config";
import { mobilePostEditState } from "./mobile-publish-edit";
import { draftMediaToAsset } from "./mobile-publish-utils";

type SearchParamsLike = {
  get(name: string): string | null;
};

type MobilePublishLifecycleOptions = {
  searchParams: SearchParamsLike;
  state: MobilePublishControllerState;
  t: ReturnType<typeof useTranslations>;
};

function abortCurrentUpload(uploadAbortControllerRef: MutableRefObject<AbortController | null>) {
  uploadAbortControllerRef.current?.abort();
}

export function useMobilePublishLifecycle({ searchParams, state, t }: MobilePublishLifecycleOptions) {
  const {
    mentionKeyword,
    setAttachmentFile,
    setAttachmentTouched,
    setBoard,
    setBody,
    setCategories,
    setCurrentDraftId,
    setCurrentUser,
    setEditingPostId,
    setEditingPostType,
    setExistingAttachment,
    setImageLimit,
    setImagePaymentMethod,
    setImagePrice,
    setImageProtectionEnabled,
    setImageProtectionNoticeEnabled,
    setImageSelectAllEnabled,
    setIsSearchingMentionUsers,
    setMediaAssets,
    setMentionUsers,
    setPaidContentPaymentMethods,
    setPaymentMaxPrices,
    setPostContentLimit,
    setSelectedCategoryId,
    setTagsList,
    setTitle,
    setTopic,
    setVisibility,
    showMentionSheet,
    uploadAbortControllerRef,
  } = state;

  useEffect(() => {
    return () => {
      abortCurrentUpload(uploadAbortControllerRef);
    };
  }, [uploadAbortControllerRef]);

  useEffect(() => {
    let cancelled = false;
    void getPostProtectionConfig()
      .then((config) => {
        if (!cancelled) {
          setImageProtectionEnabled(Boolean(config.enabled));
          setImageLimit(Math.max(1, Number(config.maxImages) || defaultImageLimit));
          setPostContentLimit(Math.max(1, Number(config.maxContentLength) || 100000));
          setImageProtectionNoticeEnabled(config.noticeEnabled !== false);
          setImageSelectAllEnabled(config.selectAllEnabled !== false);
          const methods = {
            balance: config.paymentMethods?.balance !== false,
            points: config.paymentMethods?.points !== false,
          };
          setPaidContentPaymentMethods(methods);
          setPaymentMaxPrices({
            balance: Number(config.paymentMaxPrices?.balance) > 0
              ? Number(config.paymentMaxPrices?.balance)
              : defaultMobilePaymentMaxPrices.balance,
            points: Number(config.paymentMaxPrices?.points) > 0
              ? Number(config.paymentMaxPrices?.points)
              : defaultMobilePaymentMaxPrices.points,
          });
          setImagePaymentMethod((current) => (methods[current] ? current : methods.balance ? "balance" : "points"));
        }
      })
      .catch(() => {
        if (!cancelled) {
          setImageProtectionEnabled(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [
    setImageLimit,
    setImagePaymentMethod,
    setImageProtectionEnabled,
    setImageProtectionNoticeEnabled,
    setImageSelectAllEnabled,
    setPaidContentPaymentMethods,
    setPaymentMaxPrices,
    setPostContentLimit,
  ]);

  useEffect(() => {
    let cancelled = false;
    const accessToken = getStoredAccessToken();
    if (!accessToken) {
      return () => {
        cancelled = true;
      };
    }
    fetch("/api/auth/me", {
      cache: "no-store",
      credentials: "include",
      headers: {
        authorization: `Bearer ${accessToken}`,
      },
    })
      .then(async (response) => {
        if (!response.ok) {
          throw new Error(t("publish.mobile.currentUserLoadFailed"));
        }
        const payload = await response.json() as { data?: AuthUser } | AuthUser;
        return "data" in payload && payload.data ? payload.data : payload as AuthUser;
      })
      .then((user) => {
        if (cancelled) {
          return;
        }
        setCurrentUser(user);
        window.localStorage.setItem("yuem_user", JSON.stringify(user));
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [setCurrentUser, t]);

  useEffect(() => {
    let cancelled = false;
    void loadMobilePublishData({ refresh: true })
      .then((data) => {
        if (!cancelled) {
          setCategories(data.categories);
          setTagsList(data.tags);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setCategories([]);
          setTagsList([]);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [setCategories, setTagsList]);

  useEffect(() => {
    let cancelled = false;
    const editId = searchParams.get("edit");
    if (editId) {
      void getPostDetail(editId)
        .then((post) => {
          if (cancelled) {
            return;
          }
          const restored = mobilePostEditState(post);
          setEditingPostId(String(post.id));
          setEditingPostType(restored.type);
          setTitle(restored.title);
          setBody(restored.body);
          setTopic(restored.topicInput);
          setBoard(restored.categoryName ? translateCategoryName(restored.categoryName) : "");
          setSelectedCategoryId(restored.categoryId);
          setVisibility(restored.visibility);
          setCurrentDraftId(null);
          setAttachmentFile(null);
          setExistingAttachment(restored.attachment);
          setAttachmentTouched(false);
          setMediaAssets(restored.media);
          setImagePaymentMethod(restored.paymentMethod);
          setImagePrice(restored.price);
          toast.success(t("publish.mobile.editMode"));
        })
        .catch((error) => {
          if (!cancelled) {
            toast.error(error instanceof Error ? error.message : t("publish.mobile.loadPostFailed"));
          }
        });
      return () => {
        cancelled = true;
      };
    }
    void consumePendingRestoreDraft()
      .then((draft) => {
        if (!draft || cancelled) {
          return;
        }
        setTitle(draft.title);
        setBody(draft.body);
        setTopic(draft.topic);
        setBoard(translateCategoryName(draft.board));
        setSelectedCategoryId(draft.selectedCategoryId);
        setVisibility(draft.visibility);
        setCurrentDraftId(draft.id);
        setImagePaymentMethod(draft.imagePaymentMethod ?? "balance");
        setImagePrice(draft.imagePrice ?? "1");
        setAttachmentFile(draft.attachment?.file ?? null);
        setExistingAttachment(null);
        setAttachmentTouched(Boolean(draft.attachment));
        setMediaAssets(enforceImageCoverPolicy(draft.mediaAssets.map(draftMediaToAsset)));
        toast.success(t("publish.mobile.draftRestored"));
      })
      .catch((error) => {
        if (!cancelled) {
          toast.error(error instanceof Error ? error.message : t("publish.mobile.draftRestoreFailed"));
        }
      });
    return () => {
      cancelled = true;
    };
  }, [
    searchParams,
    setAttachmentFile,
    setAttachmentTouched,
    setBoard,
    setBody,
    setCurrentDraftId,
    setEditingPostId,
    setEditingPostType,
    setExistingAttachment,
    setImagePaymentMethod,
    setImagePrice,
    setMediaAssets,
    setSelectedCategoryId,
    setTitle,
    setTopic,
    setVisibility,
    t,
  ]);

  useEffect(() => {
    if (!showMentionSheet) {
      return;
    }
    const keyword = mentionKeyword.trim();
    let cancelled = false;
    const timer = window.setTimeout(() => {
      const request = keyword
        ? searchUsers({ keyword, limit: 12 })
        : listMentionUsers({ limit: 12 });
      request
        .then((users) => {
          if (!cancelled) {
            setMentionUsers(users);
          }
        })
        .catch((error) => {
          if (!cancelled) {
            setMentionUsers([]);
            toast.error(error instanceof Error ? error.message : t("publish.mobile.userSearchFailed"));
          }
        })
        .finally(() => {
          if (!cancelled) {
            setIsSearchingMentionUsers(false);
          }
        });
    }, keyword ? 260 : 0);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [
    mentionKeyword,
    setIsSearchingMentionUsers,
    setMentionUsers,
    showMentionSheet,
    t,
  ]);
}
