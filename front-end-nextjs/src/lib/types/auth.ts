import type { SiteProfile } from "./site";

export type ApiEnvelope<T> = {
  code: number;
  message?: string;
  data: T;
  success?: boolean;
};

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
  expiresIn?: number;
};

export type AuthUser = {
  id: number | string;
  user_id?: string;
  xise_id?: string | null;
  nickname?: string | null;
  email?: string | null;
  avatar?: string | null;
  background?: string | null;
  bio?: string | null;
  location?: string | null;
  follow_count?: number;
  fans_count?: number;
  like_count?: number;
  collect_count?: number;
  post_count?: number;
  created_at?: string;
  verified?: number | boolean | null;
  verified_name?: string | null;
  birthday?: string | null;
  custom_fields?: unknown;
  education?: string | null;
  gender?: string | null;
  interests?: unknown;
  major?: string | null;
  mbti?: string | null;
  zodiac_sign?: string | null;
  profile_completed?: boolean;
  points_awards?: unknown;
  isFollowing?: boolean;
  buttonType?: string;
};

export type UserSearchResult = Pick<
  AuthUser,
  "id" | "user_id" | "xise_id" | "nickname" | "avatar" | "bio" | "verified" | "fans_count"
>;

export type AuthConfigPayload = {
  emailEnabled: boolean;
  oauth2Enabled: boolean;
  oauth2OnlyLogin: boolean;
  oauth2LoginUrl?: string;
  oauth2StartUrl?: string;
  geetestEnabled: boolean;
  geetestCaptchaId?: string;
  verificationCollectSensitiveInfo?: boolean;
  videoCenterEnabled?: boolean;
  videoCenterAccountCutoff?: string | null;
  videoCenterGuestRestricted?: boolean;
  videoCenterRequestCountryCode?: string | null;
  siteProfile?: SiteProfile | null;
};

export type OAuthAppTokenPayload = {
  access_token: string;
  refresh_token: string;
  is_new_user: boolean;
  user: AuthUser;
};

export type VerificationType = 1 | 2;

export type VerificationApplication = {
  id: number | string;
  user_id?: number | string;
  type: VerificationType | number;
  content?: string | null;
  status?: number | null;
  reason?: string | null;
  audit_result?: {
    verifiedName?: string | null;
    verified_name?: string | null;
    imageUrls?: string[] | null;
    image_urls?: string[] | null;
    images?: string[] | null;
    [key: string]: unknown;
  } | null;
  created_at?: string | null;
  audit_time?: string | null;
};

export type CreateVerificationInput = {
  type: VerificationType;
  content: string;
  verifiedName?: string;
  imageUrls?: string[];
};

export type CreateVerificationPayload = {
  id: number | string;
};

export type UserApiKey = {
  id: number | string;
  name: string;
  api_key_prefix?: string | null;
  api_key?: string;
  is_active?: boolean;
  last_used_at?: string | null;
  created_at?: string | null;
};

export type BackendApiOperation = {
  summary?: string;
  description?: string;
  operationId?: string;
  tags?: string[];
  parameters?: unknown[];
  requestBody?: unknown;
  responses?: Record<string, unknown>;
  security?: unknown[];
};

export type BackendApiSpec = {
  openapi?: string;
  info?: {
    title?: string;
    version?: string;
    description?: string;
  };
  paths?: Record<string, Record<string, BackendApiOperation>>;
  "x-route-count"?: number;
};

export type CaptchaPayload = {
  captchaId: string;
  captchaSvg: string;
};
