import type { FeedPost,VideoCenterVisibilityConfig } from "./content";

export type ProfileTabs = {
  notes: FeedPost[];
  private: FeedPost[];
  collections: FeedPost[];
  likes: FeedPost[];
};

export type UserProfilePayload = {
  profile: UserProfile;
  tabs: ProfileTabs;
  videoCenter?: VideoCenterVisibilityConfig;
};

export type UserProfile = {
  id: number | string;
  userId: string;
  displayName: string;
  avatar?: string | null;
  background?: string | null;
  bio?: string | null;
  location?: string | null;
  createdAt?: string | null;
  verified?: number | boolean | null;
  verifiedName?: string | null;
  followCount: number;
  fansCount: number;
  likeCount: number;
  collectCount: number;
  postCount: number;
  isViewer?: boolean;
  isFollowing?: boolean;
  birthday?: string | null;
  customFields?: Record<string, string>;
  education?: string | null;
  gender?: string | null;
  interests?: string[];
  major?: string | null;
  mbti?: string | null;
  zodiacSign?: string | null;
};

export type UpdateUserProfileInput = {
  nickname: string;
  avatar?: string | null;
  background?: string | null;
  birthday?: string | null;
  bio?: string | null;
  custom_fields?: Record<string, string>;
  education?: string | null;
  gender?: string | null;
  interests?: string[];
  location?: string | null;
  major?: string | null;
  mbti?: string | null;
  zodiac_sign?: string | null;
};

export type OnboardingProfileTask = {
  task_type: "set_avatar" | "set_background" | "set_name" | "set_signature" | string;
  name?: string | null;
  description?: string | null;
  points: number;
  is_active: boolean;
};

export type OnboardingFieldRule = {
  enabled?: boolean;
  min?: number;
  required?: boolean;
};

export type OnboardingPointsIntro = {
  title?: string | null;
  summary?: string | null;
  detail?: string | null;
  result_title?: string | null;
  saved_text?: string | null;
  wallet_label?: string | null;
  wallet_url?: string | null;
};

export type OnboardingConfigPayload = {
  allow_skip: boolean;
  custom_fields?: unknown[];
  enabled: boolean;
  fields?: Record<string, OnboardingFieldRule>;
  interest_options?: string[];
  points_intro?: OnboardingPointsIntro;
  profile_tasks?: OnboardingProfileTask[];
};

export type OnboardingSubmitInput = Partial<UpdateUserProfileInput> & {
  skipped?: boolean;
};

export type UserPrivacySettingsPayload = {
  ai_auto_comment_enabled: boolean;
  privacy_age: boolean;
  privacy_birthday: boolean;
  privacy_custom_fields: Record<string, boolean>;
  privacy_mbti: boolean;
  privacy_zodiac: boolean;
};
