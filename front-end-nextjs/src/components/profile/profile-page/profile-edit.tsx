"use client";
import {
  type Dispatch,
  type FormEvent,
  type SetStateAction
} from "react";
import Image from "next/image";
import Link from "next/link";
import type {
  Area
} from "react-easy-crop";
import {
  ArrowLeft,
  ImageIcon,
  X
} from "lucide-react";
import {
  useTranslations
} from "next-intl";
import {
  Avatar,
  AvatarFallback,
  AvatarImage
} from "@/components/ui/avatar";
import {
  Button
} from "@/components/ui/button";
import type {
  UpdateUserProfileInput,
  UserProfile
} from "@/lib/types";
import {
  cn
} from "@/lib/utils";
import {
  Cropper,
  ImageCropState,
  PendingImageCrop,
  ProfileImageKind
} from "./profile-config";

export type ProfileEditDraft = {
  nickname: string;
  bio: string;
  location: string;
  avatar: string;
  background: string;
  birthday: string;
  gender: string;
  interests: string;
  mbti: string;
  zodiacSign: string;
};


export function profileDraftFromProfile(profile: UserProfile): ProfileEditDraft {
  return {
    nickname: profile.displayName,
    bio: profile.bio ?? "",
    location: profile.location ?? "",
    avatar: profile.avatar ?? "",
    background: profile.background ?? "",
    birthday: formatDateInputValue(profile.birthday),
    gender: profile.gender ?? "",
    interests: profile.interests?.join("，") ?? "",
    mbti: profile.mbti ?? "",
    zodiacSign: profile.zodiacSign ?? "",
  };
}


export function formatDateInputValue(value?: string | null) {
  if (!value) {
    return "";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value.slice(0, 10);
  }

  return date.toISOString().slice(0, 10);
}


export function parseInterestsDraft(value: string) {
  return value
    .split(/[,\uFF0C\u3001]/)
    .map((item) => item.trim())
    .filter(Boolean)
    .slice(0, 20);
}


export function profileUpdateInputFromDraft(draft: ProfileEditDraft): UpdateUserProfileInput {
  return {
    nickname: draft.nickname,
    avatar: draft.avatar,
    background: draft.background,
    bio: draft.bio,
    birthday: draft.birthday,
    gender: draft.gender,
    interests: parseInterestsDraft(draft.interests),
    location: draft.location,
    mbti: draft.mbti,
    zodiac_sign: draft.zodiacSign,
  };
}


export function ProfileEditDialog({
  avatar,
  background,
  displayName,
  draft,
  isSaving,
  onAvatarSelect,
  onBackgroundSelect,
  onChange,
  onClose,
  onSubmit,
}: {
  avatar?: string | null;
  background?: string | null;
  displayName: string;
  draft: ProfileEditDraft;
  isSaving: boolean;
  onAvatarSelect: () => void;
  onBackgroundSelect: () => void;
  onChange: (draft: ProfileEditDraft) => void;
  onClose: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  const t = useTranslations();

  function updateField(key: keyof ProfileEditDraft, value: string) {
    onChange({ ...draft, [key]: value });
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/55 px-4 py-6">
      <form
        onSubmit={onSubmit}
        className="flex max-h-[88dvh] w-full max-w-[430px] flex-col overflow-hidden rounded-[10px] border border-[#d8edf8] bg-[#f7fcff] text-[#1f2937] shadow-2xl"
      >
        <div className="flex h-12 shrink-0 items-center justify-between gap-4 border-b border-[#d9ecf7] px-5">
          <h2 className="text-base font-semibold text-[#253241]">{t("profile.edit.title")}</h2>
          <Button
            type="button"
            variant="ghost"
            onClick={onClose}
            disabled={isSaving}
            size="icon"
            aria-label={t("profile.edit.cancel")}
            className="size-8 text-[#657386] hover:bg-[#e6f4ff] hover:text-[#26384f]"
          >
            <X className="size-4" />
          </Button>
        </div>

        <div className="min-h-0 space-y-4 overflow-y-auto px-5 py-4">
          <ProfileEditImagePicker
            kind="avatar"
            label="头像:"
            image={avatar}
            fallback={displayName}
            disabled={isSaving}
            onSelect={onAvatarSelect}
          />
          <ProfileEditImagePicker
            kind="background"
            label="背景图:"
            image={background}
            disabled={isSaving}
            onSelect={onBackgroundSelect}
          />
          <ProfileEditField
            label={t("profile.edit.nickname")}
            value={draft.nickname}
            onChange={(value) => updateField("nickname", value)}
            maxLength={10}
            required
          />
          <ProfileEditField
            as="textarea"
            label={t("profile.edit.bio")}
            value={draft.bio}
            onChange={(value) => updateField("bio", value)}
            maxLength={160}
          />
          <ProfileEditField
            label="地点"
            value={draft.location}
            onChange={(value) => updateField("location", value)}
            maxLength={40}
          />
          <div className="grid grid-cols-2 gap-3">
            <ProfileEditField
              label="性别"
              value={draft.gender}
              onChange={(value) => updateField("gender", value)}
              maxLength={20}
            />
            <ProfileEditField
              label="生日"
              value={draft.birthday}
              onChange={(value) => updateField("birthday", value)}
              type="date"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <ProfileEditField
              label="星座"
              value={draft.zodiacSign}
              onChange={(value) => updateField("zodiacSign", value)}
              maxLength={20}
            />
            <ProfileEditField
              label="MBTI"
              value={draft.mbti}
              onChange={(value) => updateField("mbti", value)}
              maxLength={12}
            />
          </div>
          <ProfileEditField
            label="兴趣"
            value={draft.interests}
            onChange={(value) => updateField("interests", value)}
            maxLength={120}
            placeholder="用逗号分隔，例如 摄影，旅行，绘画"
          />
        </div>

        <div className="flex shrink-0 justify-end gap-2 border-t border-[#d9ecf7] px-5 py-3">
          <Button
            type="button"
            variant="outline"
            onClick={onClose}
            disabled={isSaving}
            className="h-9 rounded-[6px] border-[#c9dce8] bg-white px-4 text-[#526273] hover:bg-[#eef7fc]"
          >
            {t("profile.edit.cancel")}
          </Button>
          <Button
            type="submit"
            disabled={isSaving}
            className="h-9 rounded-[6px] bg-[#ff4f86] px-5 text-white hover:bg-[#f43b78]"
          >
            {isSaving ? t("profile.edit.saving") : t("profile.edit.save")}
          </Button>
        </div>
      </form>
    </div>
  );
}


export function ProfileEditImagePicker({
  disabled,
  fallback,
  image,
  kind,
  label,
  onSelect,
}: {
  disabled: boolean;
  fallback?: string;
  image?: string | null;
  kind: ProfileImageKind;
  label: string;
  onSelect: () => void;
}) {
  const isAvatar = kind === "avatar";

  return (
    <div className={cn("grid gap-2", isAvatar && "justify-items-center")}>
      <span className={cn("text-sm font-medium text-[#324255]", isAvatar && "w-full text-center")}>
        {label}
      </span>
      <button
        type="button"
        disabled={disabled}
        onClick={onSelect}
        className={cn(
          "group relative overflow-hidden border border-dashed border-[#aecfe4] bg-white text-left outline-none transition hover:border-[#4aa3df] hover:bg-[#f0f9ff] focus-visible:ring-2 focus-visible:ring-[#4aa3df] disabled:cursor-not-allowed disabled:opacity-60",
          isAvatar ? "size-[92px] rounded-full" : "aspect-[16/9] w-full rounded-[8px]",
        )}
      >
        {image ? (
          isAvatar ? (
            <Avatar className="size-full">
              <AvatarImage src={image} />
              <AvatarFallback className="bg-[#eef4f8] text-xl text-[#738295]">
                {(fallback || "U").charAt(0).toUpperCase()}
              </AvatarFallback>
            </Avatar>
          ) : (
            <Image
              src={image}
              alt="背景图"
              fill
              unoptimized
              sizes="(max-width: 520px) 100vw, 430px"
              className="object-cover"
            />
          )
        ) : (
          <span className="flex size-full items-center justify-center bg-[#eef6fb] text-[#7d8da0]">
            <ImageIcon className={isAvatar ? "size-7" : "size-8"} />
          </span>
        )}
        <span className="absolute inset-0 flex items-center justify-center bg-black/0 text-xs font-semibold text-transparent transition group-hover:bg-black/35 group-hover:text-white">
          {isAvatar ? "点击更换头像" : "点击更换背景图"}
        </span>
      </button>
    </div>
  );
}


export function ProfileImageCropDialog({
  crop,
  image,
  isSaving,
  onChange,
  onClose,
  onSave,
}: {
  crop: ImageCropState;
  image: PendingImageCrop;
  isSaving: boolean;
  onChange: Dispatch<SetStateAction<ImageCropState>>;
  onClose: () => void;
  onSave: () => void;
}) {
  const aspect = image.kind === "avatar" ? 1 : 16 / 9;
  const title = image.kind === "avatar" ? "裁剪头像" : "裁剪背景图";
  const dialogWidthClassName = image.kind === "avatar" ? "max-w-[min(92vw,540px)]" : "max-w-[min(92vw,760px)]";
  const stageHeightClassName = image.kind === "avatar"
    ? "h-[clamp(220px,54dvh,520px)] max-h-[calc(100dvh-11rem)]"
    : "h-[clamp(190px,50dvh,460px)] max-h-[calc(100dvh-11rem)]";
  const cropAreaClassName = cn(
    "!border-2 !border-[#2797dc] !shadow-[0_0_0_9999px_rgba(0,0,0,0.55)]",
    "after:absolute after:-bottom-1.5 after:-right-1.5 after:size-3 after:rounded-sm after:bg-[#2797dc] after:content-['']",
  );

  return (
    <div className="fixed inset-0 z-[60] flex items-stretch justify-center overflow-y-auto bg-black/45 px-4 py-4 sm:items-center sm:px-5 sm:py-6">
      <div className={cn("my-auto flex max-h-[calc(100dvh-2rem)] w-full flex-col overflow-hidden rounded-[10px] border border-[#a7d7f2] bg-[#edf8ff] text-[#26384a] shadow-2xl", dialogWidthClassName)}>
        <div className="flex h-12 shrink-0 items-center justify-between gap-3 border-b border-[#cbe8f8] px-4">
          <h2 className="truncate text-base font-semibold">{title}</h2>
          <Button
            type="button"
            variant="ghost"
            onClick={onClose}
            disabled={isSaving}
            size="icon"
            aria-label="取消"
            className="size-8 rounded-full text-[#5b7288] hover:bg-[#d8eefc] hover:text-[#26384a]"
          >
            <X className="size-4" />
          </Button>
        </div>

        <div className="min-h-0 overflow-y-auto px-4 py-4">
          <div
            className={cn(
              "relative mx-auto min-h-[140px] w-full overflow-hidden rounded-[4px] bg-black shadow-inner",
              stageHeightClassName,
            )}
          >
            <Cropper
              image={image.src}
              crop={crop.position}
              zoom={crop.zoom}
              aspect={aspect}
              cropShape="rect"
              maxZoom={3}
              minZoom={1}
              objectFit="contain"
              roundCropAreaPixels
              showGrid={false}
              onCropChange={(position) => {
                onChange((current) => ({ ...current, position }));
              }}
              onCropComplete={(_, areaPixels) => {
                onChange((current) => ({ ...current, areaPixels }));
              }}
              onZoomChange={(zoom) => {
                onChange((current) => ({ ...current, zoom }));
              }}
              classes={{ cropAreaClassName }}
            />
          </div>
        </div>

        <div className="flex shrink-0 justify-end gap-2 border-t border-[#cbe8f8] bg-[#f8fdff] px-4 py-3">
          <Button
            type="button"
            variant="outline"
            onClick={onClose}
            disabled={isSaving}
            className="h-9 rounded-[6px] border-[#c9dce8] bg-white px-4 text-[#526273] hover:bg-[#eef7fc]"
          >
            取消
          </Button>
          <Button
            type="button"
            onClick={onSave}
            disabled={isSaving || !crop.areaPixels}
            className="h-9 rounded-[6px] bg-[#ff4f86] px-5 text-white hover:bg-[#f43b78]"
          >
            {isSaving ? "保存中..." : "确认裁剪"}
          </Button>
        </div>
      </div>
    </div>
  );
}


export function CropperSkeleton() {
  return (
    <div className="flex size-full items-center justify-center bg-black text-sm font-medium text-white/62">
      <span className="inline-flex rounded-full border border-white/10 bg-white/[0.08] px-4 py-2">
        Loading image editor...
      </span>
    </div>
  );
}


export function cropProfileImage(
  source: string,
  areaPixels: Area | null,
  kind: ProfileImageKind,
  fileName: string,
) {
  const output = kind === "avatar" ? { width: 512, height: 512 } : { width: 1280, height: 720 };

  return new Promise<File>((resolve, reject) => {
    if (!areaPixels) {
      reject(new Error("请先等待图片载入完成。"));
      return;
    }

    const sourceImage = new window.Image();
    sourceImage.onload = () => {
      const canvas = document.createElement("canvas");
      canvas.width = output.width;
      canvas.height = output.height;

      const context = canvas.getContext("2d");
      if (!context) {
        reject(new Error("Image crop failed."));
        return;
      }

      const cropX = Math.max(0, areaPixels.x);
      const cropY = Math.max(0, areaPixels.y);
      const cropWidth = Math.min(areaPixels.width, sourceImage.naturalWidth - cropX);
      const cropHeight = Math.min(areaPixels.height, sourceImage.naturalHeight - cropY);

      context.drawImage(
        sourceImage,
        cropX,
        cropY,
        cropWidth,
        cropHeight,
        0,
        0,
        output.width,
        output.height,
      );

      canvas.toBlob(
        (blob) => {
          if (!blob) {
            reject(new Error("Image crop failed."));
            return;
          }

          const extension = kind === "avatar" ? "jpg" : getImageExtension(fileName);
          const type = extension === "png" ? "image/png" : "image/jpeg";
          resolve(new File([blob], `cropped-${kind}.${extension}`, { type }));
        },
        kind === "avatar" ? "image/jpeg" : getImageMimeType(fileName),
        0.92,
      );
    };
    sourceImage.onerror = () => reject(new Error("Image crop failed."));
    sourceImage.src = source;
  });
}


export function getImageExtension(fileName: string) {
  return fileName.toLowerCase().endsWith(".png") ? "png" : "jpg";
}


export function getImageMimeType(fileName: string) {
  return getImageExtension(fileName) === "png" ? "image/png" : "image/jpeg";
}


export function ProfileEditField({
  as = "input",
  inputMode,
  label,
  maxLength,
  onChange,
  placeholder,
  required,
  type = "text",
  value,
}: {
  as?: "input" | "textarea";
  inputMode?: "url";
  label: string;
  maxLength?: number;
  onChange: (value: string) => void;
  placeholder?: string;
  required?: boolean;
  type?: "date" | "text";
  value: string;
}) {
  const controlClassName =
    "mt-2 w-full rounded-[6px] border border-[#c9dce8] bg-white px-3 py-2 text-sm text-[#26384a] outline-none transition placeholder:text-[#9aaabd] focus:border-[#4aa3df] focus:ring-2 focus:ring-[#bde5ff]";

  return (
    <label className="block">
      <span className="text-sm font-medium text-[#324255]">{label}</span>
      {as === "textarea" ? (
        <textarea
          value={value}
          onChange={(event) => onChange(event.target.value)}
          maxLength={maxLength}
          placeholder={placeholder}
          rows={3}
          className={controlClassName}
        />
      ) : (
        <input
          type={type}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          required={required}
          maxLength={maxLength}
          inputMode={inputMode}
          placeholder={placeholder}
          className={controlClassName}
        />
      )}
    </label>
  );
}


export function ProfileTopBar({
  variant,
  displayName,
  onBack,
}: {
  variant: "viewer" | "user";
  displayName: string;
  onBack?: () => void;
}) {
  const t = useTranslations();

  return (
    <header className="fixed inset-x-0 top-0 z-40 h-14 border-b border-white/[0.07] bg-[#121212]/96 backdrop-blur lg:hidden">
      <div className="flex h-full items-center gap-2 px-3">
        {onBack ? (
          <Button
            type="button"
            onClick={onBack}
            variant="ghost"
            size="icon"
            aria-label={t("profile.backHome")}
            className="size-10 text-white hover:bg-white/[0.06]"
          >
            <ArrowLeft className="size-5" />
          </Button>
        ) : (
          <Button
            asChild
            variant="ghost"
            size="icon"
            aria-label={t("profile.backHome")}
            className="size-10 text-white hover:bg-white/[0.06]"
          >
            <Link href="/">
              <ArrowLeft className="size-5" />
            </Link>
          </Button>
        )}
        <div className="min-w-0 flex-1">
          <p className="truncate text-[15px] font-semibold text-white">
            {variant === "viewer" ? t("profile.personalCenter") : displayName}
          </p>
        </div>
      </div>
    </header>
  );
}
