function checkFrontendProfileUserStaticContract(checks) {
  const problems = [];
  const details = {
    routeMatrixPath: path.relative(repoRoot, backendRouteMatrixPath),
    frontendApiPath: path.relative(repoRoot, frontendApiPath),
    frontendProfileRoutePath: path.relative(repoRoot, frontendProfileRoutePath),
    frontendUserRoutePath: path.relative(repoRoot, frontendUserRoutePath),
    frontendProfileDataPagePath: path.relative(repoRoot, frontendProfileDataPagePath),
    frontendProfilePagePath: path.relative(repoRoot, frontendProfilePagePath),
    frontendUsersHelperPath: path.relative(repoRoot, frontendUsersHelperPath),
  };

  const frontendApi = fileText(frontendApiPath);
  const profileRoute = fileText(frontendProfileRoutePath);
  const userRoute = fileText(frontendUserRoutePath);
  const profileDataPage = fileText(frontendProfileDataPagePath);
  const profilePage = fileText(frontendProfilePagePath);
  const usersHelper = fileText(frontendUsersHelperPath);

  try {
    const matrix = JSON.parse(fs.readFileSync(backendRouteMatrixPath, "utf8"));
    const routes = Array.isArray(matrix.routes) ? matrix.routes.filter(isRecord) : [];
    const expectedRoutes = [
      { method: "GET", path: "/api/auth/me", auth: "user" },
      { method: "GET", path: "/api/users/:id", auth: "user" },
      { method: "PUT", path: "/api/users/:id", auth: "user" },
      { method: "GET", path: "/api/users/:id/posts", auth: "user" },
      { method: "GET", path: "/api/users/:id/collections", auth: "user" },
      { method: "GET", path: "/api/users/:id/likes", auth: "user" },
      { method: "GET", path: "/api/users/:id/follow-status", auth: "user" },
      { method: "POST", path: "/api/users/:id/follow", auth: "user" },
      { method: "DELETE", path: "/api/users/:id/follow", auth: "user" },
    ];
    const routeResults = [];

    for (const expected of expectedRoutes) {
      const matches = routes.filter(
        (route) => route.method === expected.method && route.path === expected.path,
      );
      const actual = matches[0] ?? null;
      const result = {
        method: expected.method,
        path: expected.path,
        expectedAuth: expected.auth,
        actualAuth: actual?.auth ?? null,
        actualStatus: actual?.status ?? null,
        ok:
          matches.length === 1 &&
          actual?.auth === expected.auth &&
          actual?.status === "native-gin",
      };
      routeResults.push(result);
      if (!result.ok) {
        problems.push(`${expected.method} ${expected.path} route auth/status is not aligned`);
      }
    }

    details.profileUserRoutes = routeResults;
  } catch (error) {
    problems.push(`backend route matrix cannot be read: ${error.message}`);
  }

  const requiredPatterns = [
    {
      fileText: profileRoute,
      pattern: 'import { ViewerProfileDataPage } from "@/components/profile/profile-data-page"',
      problem: "profile route does not render ViewerProfileDataPage",
    },
    {
      fileText: profileRoute,
      pattern: "return <ViewerProfileDataPage />",
      problem: "profile route does not mount the viewer profile data loader",
    },
    {
      fileText: userRoute,
      pattern: 'import { UserProfileDataPage } from "@/components/profile/profile-data-page"',
      problem: "user route does not render UserProfileDataPage",
    },
    {
      fileText: userRoute,
      pattern: "params: Promise<{ id: string }>",
      problem: "user route does not read the dynamic user id from route params",
    },
    {
      fileText: userRoute,
      pattern: "return <UserProfileDataPage userId={id} />",
      problem: "user route does not pass the dynamic id into the profile data loader",
    },
    {
      fileText: profileDataPage,
      pattern: "export function ViewerProfileDataPage()",
      problem: "viewer profile data component is missing",
    },
    {
      fileText: profileDataPage,
      pattern: "export function UserProfileDataPage({ userId }: { userId: string })",
      problem: "user profile data component is missing",
    },
    {
      fileText: profileDataPage,
      pattern: "variant === \"viewer\"",
      problem: "profile data loader does not branch between viewer and public user profiles",
    },
    {
      fileText: profileDataPage,
      pattern: "? await getViewerProfileData()",
      problem: "viewer profile data loader does not call getViewerProfileData",
    },
    {
      fileText: profileDataPage,
      pattern: ": await getUserProfileData(userId ?? \"\")",
      problem: "user profile data loader does not call getUserProfileData",
    },
    {
      fileText: profileDataPage,
      pattern: "loadError instanceof ApiError",
      problem: "profile data loader does not inspect API errors",
    },
    {
      fileText: profileDataPage,
      pattern: "loadError.status === 404",
      problem: "profile data loader does not render not-found state for missing users",
    },
    {
      fileText: profileDataPage,
      pattern: "<ProfileNotFound userId={userId ?? \"\"} />",
      problem: "profile data loader does not render ProfileNotFound",
    },
    {
      fileText: profileDataPage,
      pattern: "profile={payload.profile}",
      problem: "profile data loader does not pass profile payload into ProfilePage",
    },
    {
      fileText: profileDataPage,
      pattern: "tabs={payload.tabs}",
      problem: "profile data loader does not pass profile tab payloads into ProfilePage",
    },
    {
      fileText: frontendApi,
      pattern: "export function getCurrentUser(",
      problem: "frontend getCurrentUser helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<AuthUser>("/api/auth/me"',
      problem: "getCurrentUser does not call /api/auth/me",
    },
    {
      fileText: frontendApi,
      pattern: "export function mapUserProfile(user: AuthUser, isViewer = false): UserProfile",
      problem: "mapUserProfile helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: "user.user_id ?? String(user.id)",
      problem: "mapUserProfile does not normalize backend user identifiers",
    },
    {
      fileText: frontendApi,
      pattern: "async function getUserPostTabs(userId: string | number): Promise<ProfileTabs>",
      problem: "profile tab loader helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: "apiGet<FeedPayload>(`/api/users/${userId}/posts`, { page: 1, limit: 24 })",
      problem: "profile notes tab does not call /api/users/:id/posts",
    },
    {
      fileText: frontendApi,
      pattern: "`/api/users/${userId}/collections`",
      problem: "profile collections tab does not call /api/users/:id/collections",
    },
    {
      fileText: frontendApi,
      pattern: "`/api/users/${userId}/likes`",
      problem: "profile likes tab does not call /api/users/:id/likes",
    },
    {
      fileText: frontendApi,
      pattern: "export async function getViewerProfileData(context: ApiRequestContext = {}): Promise<UserProfilePayload>",
      problem: "viewer profile data helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: "const user = await getCurrentUser(context);",
      problem: "viewer profile data helper does not load the current user",
    },
    {
      fileText: frontendApi,
      pattern: "const tabs = await getUserPostTabs(user.user_id ?? user.id);",
      problem: "viewer profile data helper does not load tabs for the current user id",
    },
    {
      fileText: frontendApi,
      pattern: "export async function getUserProfileData(userId: string): Promise<UserProfilePayload>",
      problem: "user profile data helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: "apiGet<AuthUser>(`/api/users/${encodeURIComponent(userId)}`)",
      problem: "user profile data helper does not load /api/users/:id",
    },
    {
      fileText: frontendApi,
      pattern: "getFollowStatus(userId)",
      problem: "user profile data helper does not load follow status",
    },
    {
      fileText: frontendApi,
      pattern: "isFollowing: followStatus.isFollowing ?? followStatus.followed",
      problem: "user profile data helper does not map follow-state payloads",
    },
    {
      fileText: frontendApi,
      pattern: "export async function updateUserProfile(userId: string | number, input: UpdateUserProfileInput)",
      problem: "updateUserProfile helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: 'apiPut<AuthUser>(`/api/users/${encodeURIComponent(String(userId))}`',
      problem: "updateUserProfile does not PUT /api/users/:id",
    },
    {
      fileText: frontendApi,
      pattern: "return mapUserProfile(user, true)",
      problem: "updateUserProfile does not map the updated backend user payload",
    },
    {
      fileText: frontendApi,
      pattern: "export function getFollowStatus(userId: string | number)",
      problem: "getFollowStatus helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: "apiGet<FollowStatusPayload>(`/api/users/${userId}/follow-status`)",
      problem: "getFollowStatus does not call /api/users/:id/follow-status",
    },
    {
      fileText: frontendApi,
      pattern: "export function followUser(userId: string | number)",
      problem: "followUser helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: "export function unfollowUser(userId: string | number)",
      problem: "unfollowUser helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: "`/api/users/${userId}/follow`",
      problem: "follow/unfollow helpers do not call /api/users/:id/follow",
    },
    {
      fileText: profilePage,
      pattern: "type ProfileTabKey = \"notes\" | \"collections\" | \"likes\"",
      problem: "profile page tab keys are missing notes/collections/likes",
    },
    {
      fileText: profilePage,
      pattern: "const visiblePosts = postsByTab[activeTab]",
      problem: "profile page does not render active tab backend data",
    },
    {
      fileText: profilePage,
      pattern: "getNotificationUnreadCount()",
      problem: "viewer profile page does not load notification unread count",
    },
    {
      fileText: profilePage,
      pattern: "function openEditProfile()",
      problem: "viewer profile page does not expose edit-profile dialog state",
    },
    {
      fileText: profilePage,
      pattern: "const updated = await updateUserProfile(profileState.userId, editDraft)",
      problem: "profile edit dialog does not submit to updateUserProfile",
    },
    {
      fileText: profilePage,
      pattern: "setProfileState((current) => ({",
      problem: "profile edit submit does not update local profile state",
    },
    {
      fileText: profilePage,
      pattern: "function ProfileEditDialog(",
      problem: "profile edit dialog component is missing",
    },
    {
      fileText: profilePage,
      pattern: "t(\"profile.edit.nicknameRequired\")",
      problem: "profile edit dialog does not validate nickname before update",
    },
    {
      fileText: profilePage,
      pattern: "const result = await toggleLike(post.id)",
      problem: "profile page does not wire tab likes through toggleLike",
    },
    {
      fileText: profilePage,
      pattern: "const result = await toggleCollect(post.id)",
      problem: "profile page does not wire tab collections through toggleCollect",
    },
    {
      fileText: profilePage,
      pattern: "? await followUser(targetUserId)",
      problem: "profile page does not call followUser when following a user",
    },
    {
      fileText: profilePage,
      pattern: ": await unfollowUser(targetUserId)",
      problem: "profile page does not call unfollowUser when unfollowing a user",
    },
    {
      fileText: profilePage,
      pattern: "const conversation = await createImConversation([profileState.id])",
      problem: "profile page does not start IM conversations from user profiles",
    },
    {
      fileText: profilePage,
      pattern: "router.push(`/messages?conversation=${conversation.id}`)",
      problem: "profile page does not deep-link to created IM conversations",
    },
    {
      fileText: profilePage,
      pattern: 'Link href="/wallet"',
      problem: "viewer profile page does not expose the wallet entry",
    },
    {
      fileText: profilePage,
      pattern: 'Link href="/notifications"',
      problem: "profile page does not expose the notifications entry",
    },
    {
      fileText: profilePage,
      pattern: "<PostDetailDrawer",
      problem: "profile page does not open tab posts in PostDetailDrawer",
    },
    {
      fileText: usersHelper,
      pattern: "export function getPostAuthorUserId(post: FeedPost)",
      problem: "post author user-id helper is missing",
    },
    {
      fileText: usersHelper,
      pattern: "post.author_account",
      problem: "post author helper does not prefer backend author_account",
    },
    {
      fileText: usersHelper,
      pattern: "post.user_id !== undefined",
      problem: "post author helper does not fall back to backend user_id",
    },
    {
      fileText: usersHelper,
      pattern: "export function getUserHrefFromPost(post: FeedPost)",
      problem: "post user link helper is missing",
    },
    {
      fileText: usersHelper,
      pattern: "return `/user/${encodeURIComponent(userId)}`",
      problem: "user link helper does not route to /user/[id]",
    },
  ];

  for (const check of requiredPatterns) {
    if (!check.fileText.includes(check.pattern)) {
      problems.push(check.problem);
    }
  }

  if (/fixture|mock/i.test(usersHelper)) {
    problems.push("src/lib/users.ts still appears to expose fixture/mock user profile data");
  }

  addCheck(
    checks,
    "frontend-profile-user-contract",
    problems.length === 0 ? "pass" : "fail",
    problems.length === 0
      ? "frontend profile and user page contract is aligned with backend user routes"
      : "frontend profile and user page contract is not aligned with backend user routes",
    {
      ...details,
      problems,
    },
  );
}

