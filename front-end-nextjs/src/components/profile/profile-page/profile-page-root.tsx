"use client";
import {
  useMemo,
  useRef,
  useState,
  useTransition,
  type ChangeEvent,
  type FormEvent
} from "react";
import Image from "next/image";
import Link from "next/link";
import {
  useRouter
} from "next/navigation";
import {
  Masonry
} from "react-plock";
import {
  ArrowLeft,
  Bookmark,
  Heart,
  Home,
  Lock,
  MessageCircle,
  Pencil,
  Settings,
  UserPlus,
  Wallet
} from "lucide-react";
import {
  useTranslations
} from "next-intl";
import {
  toast
} from "sonner";
import {
  Avatar,
  AvatarFallback,
  AvatarImage
} from "@/components/ui/avatar";
import {
  Button
} from "@/components/ui/button";
import { MarkdownContent } from "@/components/markdown-content";
import {
  TooltipProvider
} from "@/components/ui/tooltip";
import type {
  FeedPost,
  ProfileTabs,
  UserProfile
} from "@/lib/types";
import {
  cn
} from "@/lib/utils";
import {
  PostCard
} from "@/components/feed/post-card";
import {
  createImConversation,
  followUser,
  toggleLike,
  unfollowUser,
  updateUserProfile,
  uploadImage
} from "@/lib/api";
import {
  ImageCropState,
  PendingImageCrop,
  ProfileImageKind,
  ProfileTabKey,
  createInitialImageCrop,
  mobileQuickActions,
  publicProfileTabKeys,
  tabKeys
} from "./profile-config";
import {
  normalizeProfileTabs
} from "./profile-tabs";
import {
  ProfileEditDialog,
  ProfileImageCropDialog,
  ProfileTopBar,
  cropProfileImage,
  profileDraftFromProfile,
  profileUpdateInputFromDraft
} from "./profile-edit";
import {
  formatCompactCount
} from "./profile-shell";

export function ProfilePage({
  onBack,
  profile,
  tabs,
  variant,
}: {
  onBack?: () => void;
  profile: UserProfile;
  tabs: ProfileTabs;
  variant: "viewer" | "user";
}) {
  const t = useTranslations();
  const router = useRouter();
  const initialTabs = normalizeProfileTabs(tabs);
  const [activeContent, setActiveContent] = useState<ProfileTabKey>("notes");
  const [profileState, setProfileState] = useState(profile);
  const [postsByTab, setPostsByTab] = useState<ProfileTabs>(initialTabs);
  const [isStartingConversation, setIsStartingConversation] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editDraft, setEditDraft] = useState(() => profileDraftFromProfile(profile));
  const [pendingImageCrop, setPendingImageCrop] = useState<PendingImageCrop | null>(null);
  const [imageCrop, setImageCrop] = useState<ImageCropState>(() => createInitialImageCrop());
  const avatarInputRef = useRef<HTMLInputElement | null>(null);
  const backgroundInputRef = useRef<HTMLInputElement | null>(null);
  const [isSavingProfile, startSavingProfile] = useTransition();

  const visibleTabKeys: readonly ProfileTabKey[] =
    variant === "viewer" ? tabKeys : publicProfileTabKeys;
  const safeActiveContent = visibleTabKeys.includes(activeContent)
    ? activeContent
    : "notes";
  const visiblePosts = postsByTab[safeActiveContent];
  const stats = useMemo(
    () => [
      { key: "following", value: profileState.followCount },
      { key: "followers", value: profileState.fansCount },
      {
        key: "likesAndCollections",
        value: profileState.likeCount + profileState.collectCount,
      },
    ],
    [
      profileState.collectCount,
      profileState.fansCount,
      profileState.followCount,
      profileState.likeCount,
    ],
  );

  async function handleLike(post: FeedPost) {
    const nextLiked = !post.liked;
    const updatePost = (item: FeedPost) =>
      item.id === post.id
        ? {
            ...item,
            liked: nextLiked,
            like_count: Math.max(0, item.like_count + (nextLiked ? 1 : -1)),
          }
        : item;

    setPostsByTab((currentTabs) => {
      return {
        notes: currentTabs.notes.map(updatePost),
        private: currentTabs.private.map(updatePost),
        collections: currentTabs.collections.map(updatePost),
        likes: currentTabs.likes.map(updatePost),
      };
    });

    try {
      const result = await toggleLike(post.id);
      const liked = result.liked;
      const syncPost = (item: FeedPost) =>
        item.id === post.id
          ? {
              ...item,
              liked,
              like_count: Math.max(
                0,
                item.like_count + (liked === nextLiked ? 0 : liked ? 1 : -1),
              ),
            }
          : item;

      setPostsByTab((currentTabs) => {
        return {
          notes: currentTabs.notes.map(syncPost),
          private: currentTabs.private.map(syncPost),
          collections: currentTabs.collections.map(syncPost),
          likes: currentTabs.likes.map(syncPost),
        };
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Like failed");
    }
  }

  async function handleFollowToggle() {
    const targetUserId = profileState.userId;
    const nextFollowing = !profileState.isFollowing;

    setProfileState((current) => ({
      ...current,
      isFollowing: nextFollowing,
      fansCount: Math.max(0, current.fansCount + (nextFollowing ? 1 : -1)),
    }));

    try {
      const result = nextFollowing
        ? await followUser(targetUserId)
        : await unfollowUser(targetUserId);
      const isFollowing = result.isFollowing ?? result.followed ?? nextFollowing;
      setProfileState((current) => ({
        ...current,
        isFollowing,
        fansCount: Math.max(
          0,
          current.fansCount + (isFollowing === nextFollowing ? 0 : isFollowing ? 1 : -1),
        ),
      }));
    } catch (error) {
      setProfileState((current) => ({
        ...current,
        isFollowing: !nextFollowing,
        fansCount: Math.max(0, current.fansCount + (nextFollowing ? -1 : 1)),
      }));
      toast.error(error instanceof Error ? error.message : "Follow failed");
    }
  }

  async function handleStartConversation() {
    if (variant === "viewer" || isStartingConversation) {
      return;
    }

    setIsStartingConversation(true);
    try {
      const conversation = await createImConversation([profileState.id]);
      toast.success(t("profile.messageReady"));
      router.push(`/messages/${conversation.id}`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("profile.messageFailed"));
    } finally {
      setIsStartingConversation(false);
    }
  }

  function openEditProfile() {
    setEditDraft(profileDraftFromProfile(profileState));
    setEditOpen(true);
  }

  function handleProfileSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!editDraft.nickname.trim()) {
      toast.error(t("profile.edit.nicknameRequired"));
      return;
    }

    startSavingProfile(async () => {
      try {
        const updated = await updateUserProfile(profileState.userId, profileUpdateInputFromDraft(editDraft));
        setProfileState((current) => ({
          ...current,
          ...updated,
          isViewer: true,
        }));
        setEditDraft(profileDraftFromProfile(updated));
        setEditOpen(false);
        toast.success(t("profile.edit.success"));
        router.refresh();
      } catch (error) {
        toast.error(error instanceof Error ? error.message : t("profile.edit.failure"));
      }
    });
  }

  function handleProfileImageChange(kind: ProfileImageKind, event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";

    if (!file || variant !== "viewer" || isSavingProfile) {
      return;
    }

    if (!file.type.startsWith("image/")) {
      toast.error("Please select an image file.");
      return;
    }

    const reader = new FileReader();
    reader.onload = () => {
      if (typeof reader.result !== "string") {
        toast.error("Image preview failed.");
        return;
      }

      setImageCrop(createInitialImageCrop());
      setPendingImageCrop({
        kind,
        name: file.name,
        src: reader.result,
      });
    };
    reader.onerror = () => toast.error("Image preview failed.");
    reader.readAsDataURL(file);
  }

  function handleCropDialogClose() {
    if (!isSavingProfile) {
      setPendingImageCrop(null);
    }
  }

  function handleCroppedImageSave() {
    if (!pendingImageCrop) {
      return;
    }

    const target = pendingImageCrop;
    startSavingProfile(async () => {
      try {
        const croppedFile = await cropProfileImage(
          target.src,
          imageCrop.areaPixels,
          target.kind,
          target.name,
        );
        const asset = await uploadImage(croppedFile, { purpose: target.kind });
        const nextDraft = {
          ...profileDraftFromProfile(profileState),
          ...editDraft,
          avatar: profileState.avatar ?? "",
          background: profileState.background ?? "",
          [target.kind]: asset.url,
        };
        const updated = await updateUserProfile(profileState.id, profileUpdateInputFromDraft(nextDraft));
        setProfileState((current) => ({
          ...current,
          ...updated,
          isViewer: true,
        }));
        setEditDraft(profileDraftFromProfile(updated));
        setPendingImageCrop(null);
        toast.success(target.kind === "avatar" ? "头像已更新" : "背景图已更新");
        router.refresh();
      } catch (error) {
        toast.error(error instanceof Error ? error.message : t("profile.edit.failure"));
      }
    });
  }

  return (
    <div className="theme-adaptive min-h-screen bg-[#121212] text-[#e0e0e0]">
      <ProfileTopBar
        variant={variant}
        displayName={profileState.displayName}
        onBack={onBack}
      />

      <main className="mx-auto min-h-screen w-full max-w-[1120px] px-4 pb-24 pt-[72px] sm:px-6 lg:px-8 lg:pb-14 lg:pt-10">
        <div className="mb-5 hidden h-10 items-center justify-between lg:flex">
          {onBack ? (
            <Button
              type="button"
              onClick={onBack}
              variant="ghost"
              className="h-10 px-3 text-white/64 hover:bg-white/[0.06] hover:text-white"
            >
              <ArrowLeft className="size-4" />
              {t("profile.backHome")}
            </Button>
          ) : (
            <Button
              asChild
              variant="ghost"
              className="h-10 px-3 text-white/64 hover:bg-white/[0.06] hover:text-white"
            >
              <Link href="/">
                <ArrowLeft className="size-4" />
                {t("profile.backHome")}
              </Link>
            </Button>
          )}
          <span className="min-w-0 truncate text-sm text-white/45">
            {variant === "viewer" ? t("profile.personalCenter") : profileState.displayName}
          </span>
        </div>

        <section className="relative overflow-hidden rounded-2xl border border-white/[0.08] bg-[#181818] shadow-[0_18px_50px_rgba(0,0,0,0.2)]">
          <input
            ref={backgroundInputRef}
            type="file"
            accept="image/*"
            className="hidden"
            onChange={(event) => handleProfileImageChange("background", event)}
          />
          <input
            ref={avatarInputRef}
            type="file"
            accept="image/*"
            className="hidden"
            onChange={(event) => handleProfileImageChange("avatar", event)}
          />
          <button
            type="button"
            aria-label="Change background"
            disabled={variant !== "viewer" || isSavingProfile}
            onClick={() => backgroundInputRef.current?.click()}
            className="absolute inset-0 bg-[#242428] text-left outline-none transition hover:brightness-105 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary disabled:cursor-default lg:relative lg:inset-auto lg:block lg:h-[220px]"
          >
            {profileState.background ? (
              <Image
                src={profileState.background}
                alt={t("profile.coverAlt", { name: profileState.displayName })}
                fill
                unoptimized
                priority
                sizes="(max-width: 768px) 100vw, 1120px"
                className="object-cover opacity-70 lg:opacity-100"
              />
            ) : (
              <div className="size-full bg-[radial-gradient(circle_at_18%_10%,rgba(255,255,255,0.16),transparent_34%),linear-gradient(160deg,#2a2a2f_0%,#171719_58%,#121212_100%)] lg:bg-[#242428]" />
            )}
            <div className="absolute inset-0 bg-gradient-to-b from-white/[0.16] via-[#181818]/40 to-[#181818] lg:bg-gradient-to-t lg:from-[#181818] lg:via-[#181818]/25 lg:to-transparent" />
          </button>

          <div className="relative z-10 px-5 pb-5 pt-5 sm:px-8 sm:pb-8 lg:px-5 lg:pb-6 lg:pt-0">
            <div className="flex items-center justify-between gap-3 lg:-mt-14 lg:items-end">
              <div className="flex min-w-0 items-center gap-3 lg:items-end lg:gap-4">
                {variant === "viewer" ? (
                  <button
                    type="button"
                    aria-label="Change avatar"
                    disabled={isSavingProfile}
                    onClick={() => avatarInputRef.current?.click()}
                    className="shrink-0 rounded-full outline-none transition hover:opacity-90 focus-visible:ring-2 focus-visible:ring-primary/70 disabled:cursor-default"
                  >
                    <Avatar className="size-16 border-2 border-white/25 bg-[#29292e] shadow-[0_8px_24px_rgba(0,0,0,0.22)] lg:size-28 lg:border-[3px] lg:border-[#181818]">
                      <AvatarImage src={profileState.avatar ?? undefined} />
                      <AvatarFallback className="bg-[#29292e] text-xl text-white/78 lg:text-3xl">
                        {profileState.displayName.charAt(0).toUpperCase()}
                      </AvatarFallback>
                    </Avatar>
                  </button>
                ) : (
                  <Avatar className="size-16 border-2 border-white/25 bg-[#29292e] shadow-[0_8px_24px_rgba(0,0,0,0.22)] lg:size-28 lg:border-[3px] lg:border-[#181818]">
                    <AvatarImage src={profileState.avatar ?? undefined} />
                    <AvatarFallback className="bg-[#29292e] text-xl text-white/78 lg:text-3xl">
                      {profileState.displayName.charAt(0).toUpperCase()}
                    </AvatarFallback>
                  </Avatar>
                )}

                <div className="min-w-0 lg:pb-1">
                  <div className="flex min-w-0 items-center gap-2">
                    <h1 className="truncate text-[19px] font-bold leading-tight text-white drop-shadow-sm lg:text-[30px]">
                      {profileState.displayName}
                    </h1>
                    {profileState.verified ? (
                      <span className="shrink-0 rounded-full bg-primary px-2 py-0.5 text-[11px] font-semibold text-white">
                        {t("profile.verified")}
                      </span>
                    ) : null}
                  </div>
                  <p className="mt-1 truncate text-[13px] text-white/62 lg:text-sm lg:text-white/45">
                    {t("profile.userId", { id: profileState.userId })}
                  </p>
                </div>
              </div>

              {variant === "viewer" ? (
                <div className="shrink-0 lg:hidden">
                  <Button
                    type="button"
                    onClick={openEditProfile}
                    className="h-9 rounded-full bg-white/16 px-4 text-sm font-semibold text-white shadow-sm backdrop-blur hover:bg-white/22"
                  >
                    <Pencil className="size-4" />
                    {t("profile.editProfile")}
                  </Button>
                </div>
              ) : null}

              <div className="hidden h-10 items-center gap-2 lg:flex lg:pb-1">
                {variant === "viewer" ? (
                  <>
                    <Button
                      type="button"
                      onClick={openEditProfile}
                      className="h-10 px-4 text-sm font-semibold"
                    >
                      <Pencil className="size-4" />
                      {t("profile.editProfile")}
                    </Button>
                    <Button
                      asChild
                      variant="outline"
                      className="h-10 border-white/10 bg-white/[0.04] px-4 text-white hover:bg-white/[0.08]"
                    >
                      <Link href="/wallet">
                        <Wallet className="size-4" />
                        钱包
                      </Link>
                    </Button>
                    <Button
                      variant="outline"
                      size="icon"
                      aria-label={t("profile.settings")}
                      className="size-10 border-white/10 bg-white/[0.04] text-white hover:bg-white/[0.08]"
                    >
                      <Settings className="size-5" />
                    </Button>
                  </>
                ) : (
                  <>
                    <Button
                      type="button"
                      onClick={handleFollowToggle}
                      className="h-10 px-5 text-sm font-semibold"
                    >
                      <UserPlus className="size-4" />
                      {profileState.isFollowing
                        ? t("profile.followingAction")
                        : t("profile.follow")}
                    </Button>
                    <Button
                      variant="outline"
                      type="button"
                      onClick={handleStartConversation}
                      disabled={isStartingConversation}
                      className="h-10 border-white/10 bg-white/[0.04] px-4 text-sm text-white hover:bg-white/[0.08]"
                    >
                      <MessageCircle className="size-4" />
                      {t("profile.message")}
                    </Button>
                  </>
                )}
              </div>
            </div>

            {variant === "user" ? (
              <div className="mt-4 grid grid-cols-2 gap-2 lg:hidden">
                <Button
                  type="button"
                  onClick={handleFollowToggle}
                  className="h-10 min-w-0 rounded-lg px-2 text-sm font-semibold"
                >
                  <UserPlus className="size-4 shrink-0" />
                  <span className="truncate">
                    {profileState.isFollowing
                      ? t("profile.followingAction")
                      : t("profile.follow")}
                  </span>
                </Button>
                <Button
                  variant="outline"
                  type="button"
                  onClick={handleStartConversation}
                  disabled={isStartingConversation}
                  className="h-10 min-w-0 rounded-lg border-white/10 bg-white/[0.04] px-2 text-sm text-white hover:bg-white/[0.08]"
                >
                  <MessageCircle className="size-4 shrink-0" />
                  <span className="truncate">{t("profile.message")}</span>
                </Button>
              </div>
            ) : null}

            <MarkdownContent
              className="markdown-content-compact mt-5 max-w-2xl text-[15px] leading-6 text-white/80 lg:mt-4 lg:leading-7 lg:text-white/76"
              content={profileState.bio || t("profile.emptyBio")}
            />

            <div className="mt-3 flex flex-wrap gap-2 text-xs text-white/66 lg:mt-4 lg:text-white/50">
              {profileState.interests?.slice(0, 3).map((interest) => (
                <span key={interest} className="inline-flex h-7 items-center rounded-full bg-white/[0.1] px-3 lg:bg-white/[0.05]">
                  {interest}
                </span>
              ))}
            </div>

            <div className="mt-5 grid grid-cols-3 rounded-xl border border-white/[0.08] bg-black/20 lg:max-w-[520px] lg:rounded-2xl lg:bg-white/[0.035]">
              {stats.map((stat) => (
                <div key={stat.key} className="px-2 py-3 text-center sm:px-3">
                  <p className="text-[17px] font-bold leading-tight text-white lg:text-lg">
                    {formatCompactCount(stat.value)}
                  </p>
                  <p className="mt-1 truncate text-[11px] leading-4 text-white/52 sm:text-xs lg:text-white/45">
                    {t(`profile.stats.${stat.key}`)}
                  </p>
                </div>
              ))}
            </div>

            {variant === "viewer" ? (
              <div className="mt-4 flex gap-2 overflow-x-auto overscroll-x-contain pb-1 [scrollbar-width:none] lg:hidden [&::-webkit-scrollbar]:hidden">
                {mobileQuickActions.map(({ key, icon: Icon, href }) => {
                  const content = (
                    <>
                      <Icon className="size-4 shrink-0" />
                      <span className="min-w-0 whitespace-nowrap text-center leading-tight">
                        {t(`profile.quickActions.${key}`)}
                      </span>
                    </>
                  );
                  const className = cn(
                    "flex h-10 min-w-[112px] shrink-0 items-center justify-center gap-1.5 rounded-lg border border-white/[0.09] bg-white/[0.1] px-3 text-xs font-semibold text-white/86 backdrop-blur transition-colors hover:bg-white/[0.16]",
                  );

                  return href ? (
                    <Link key={key} href={href} className={className}>
                      {content}
                    </Link>
                  ) : (
                    <button
                      key={key}
                      type="button"
                      className={className}
                    >
                      {content}
                    </button>
                  );
                })}
              </div>
            ) : null}
          </div>
        </section>

        <section className="mt-4 lg:mt-6">
          <div className="sticky top-[56px] z-20 -mx-4 bg-[#121212]/96 px-4 py-2 backdrop-blur sm:-mx-6 sm:px-6 lg:top-0 lg:-mx-8 lg:border-b lg:border-white/[0.07] lg:px-8 lg:py-0">
            <div className="mx-auto flex h-11 max-w-[360px] items-center justify-center gap-1 rounded-full bg-white/[0.05] p-1 lg:h-[52px] lg:max-w-[680px] lg:rounded-none lg:bg-transparent lg:p-0">
              {visibleTabKeys.map((tabKey) => {
                const active = safeActiveContent === tabKey;

                return (
                  <button
                    key={tabKey}
                    type="button"
                    onClick={() => setActiveContent(tabKey)}
                    className={cn(
                      "relative flex h-9 min-w-0 flex-1 items-center justify-center gap-2 rounded-full px-2 text-sm font-semibold text-white/52 transition-colors lg:h-[52px] lg:rounded-none",
                      active && "bg-white text-[#121212] lg:bg-transparent lg:text-white",
                    )}
                  >
                    {tabKey === "notes" ? <Home className="hidden size-3.5 lg:block" /> : null}
                    {tabKey === "private" ? <Lock className="hidden size-3.5 lg:block" /> : null}
                    {tabKey === "collections" ? <Bookmark className="hidden size-3.5 lg:block" /> : null}
                    {tabKey === "likes" ? <Heart className="hidden size-3.5 lg:block" /> : null}
                    <span className="truncate">{t(`profile.tabs.${tabKey}`)}</span>
                    {active ? (
                      <span className="absolute bottom-0 hidden h-0.5 w-9 rounded-full bg-primary lg:block" />
                    ) : null}
                  </button>
                );
              })}
            </div>
          </div>

          <div
            key={safeActiveContent}
            className="animate-[profile-content-in_260ms_cubic-bezier(0.2,0.8,0.2,1)] pt-5"
          >
            {visiblePosts.length > 0 ? (
              <TooltipProvider>
                <Masonry
                  items={visiblePosts}
                  config={{
                    columns: [2, 3, 4, 5],
                    gap: [10, 18, 22, 26],
                    media: [640, 920, 1200, 1500],
                    useBalancedLayout: true,
                  }}
                  render={(post, index) => (
                    <PostCard
                      key={`${post.id}-${index}`}
                      post={post}
                      index={index}
                      transitionScope="profile"
                      onLike={handleLike}
                    />
                  )}
                />
              </TooltipProvider>
            ) : (
              <div className="flex min-h-[32vh] flex-col items-center justify-center rounded-2xl border border-dashed border-white/10 px-6 text-center">
                <p className="text-base font-semibold text-white">
                  {t("profile.emptyTabTitle")}
                </p>
                <p className="mt-2 text-sm text-white/45">
                  {t("profile.emptyTabDescription")}
                </p>
              </div>
            )}
          </div>
        </section>
      </main>

      {variant === "viewer" && editOpen ? (
        <ProfileEditDialog
          draft={editDraft}
          isSaving={isSavingProfile}
          onChange={setEditDraft}
            onClose={() => {
              if (!isSavingProfile) {
                setEditOpen(false);
              }
            }}
          avatar={editDraft.avatar || profileState.avatar}
          background={editDraft.background || profileState.background}
          displayName={editDraft.nickname || profileState.displayName}
          onAvatarSelect={() => avatarInputRef.current?.click()}
          onBackgroundSelect={() => backgroundInputRef.current?.click()}
          onSubmit={handleProfileSubmit}
        />
      ) : null}

      {variant === "viewer" && pendingImageCrop ? (
        <ProfileImageCropDialog
          crop={imageCrop}
          image={pendingImageCrop}
          isSaving={isSavingProfile}
          onChange={setImageCrop}
          onClose={handleCropDialogClose}
          onSave={handleCroppedImageSave}
        />
      ) : null}
    </div>
  );
}
