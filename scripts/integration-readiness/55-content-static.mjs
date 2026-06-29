function checkFrontendContentInteractionStaticContract(checks) {
  const problems = [];
  const details = {
    routeMatrixPath: path.relative(repoRoot, backendRouteMatrixPath),
    frontendApiPath: path.relative(repoRoot, frontendApiPath),
    frontendExploreFeedPath: path.relative(repoRoot, frontendExploreFeedPath),
    frontendPostDetailDrawerPath: path.relative(repoRoot, frontendPostDetailDrawerPath),
  };

  const frontendApi = fileText(frontendApiPath);
  const exploreFeed = fileText(frontendExploreFeedPath);
  const postDetailDrawer = fileText(frontendPostDetailDrawerPath);

  try {
    const matrix = JSON.parse(fs.readFileSync(backendRouteMatrixPath, "utf8"));
    const routes = Array.isArray(matrix.routes) ? matrix.routes.filter(isRecord) : [];
    const expectedRoutes = [
      { method: "GET", path: "/api/posts/:id", auth: "optional-note-guest-restricted" },
      { method: "GET", path: "/api/posts/:id/comments", auth: "optional-note-guest-restricted" },
      { method: "POST", path: "/api/comments", auth: "user" },
      { method: "POST", path: "/api/likes", auth: "user" },
      { method: "POST", path: "/api/posts/:id/collect", auth: "user" },
      { method: "GET", path: "/api/users/:id/follow-status", auth: "user" },
      { method: "POST", path: "/api/users/:id/follow", auth: "user" },
      { method: "DELETE", path: "/api/users/:id/follow", auth: "user" },
      { method: "GET", path: "/api/dislikes", auth: "user" },
      { method: "POST", path: "/api/dislikes", auth: "user" },
      { method: "GET", path: "/api/reports/check", auth: "user" },
      { method: "POST", path: "/api/reports", auth: "user" },
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

    details.contentInteractionRoutes = routeResults;
  } catch (error) {
    problems.push(`backend route matrix cannot be read: ${error.message}`);
  }

  const requiredPatterns = [
    {
      fileText: frontendApi,
      pattern: "export function getPostDetail(postId: string | number)",
      problem: "frontend getPostDetail helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<FeedPost>(`/api/posts/${postId}`, undefined, { auth: true })',
      problem: "getPostDetail does not send the stored ordinary-user token",
    },
    {
      fileText: frontendApi,
      pattern: "export function getPostComments(postId: string | number, page = 1, limit = 20)",
      problem: "frontend getPostComments helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: "`/api/posts/${postId}/comments`",
      problem: "getPostComments does not call /api/posts/:id/comments",
    },
    {
      fileText: frontendApi,
      pattern: "{ auth: true }",
      problem: "post detail/comment helpers do not send the stored ordinary-user token",
    },
    {
      fileText: frontendApi,
      pattern: "{ page, limit }",
      problem: "getPostComments does not pass pagination query parameters",
    },
    {
      fileText: frontendApi,
      pattern: "export function createComment(postId: string | number, content: string)",
      problem: "frontend createComment helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: 'apiPost<BackendComment>("/api/comments",',
      problem: "createComment does not call POST /api/comments",
    },
    {
      fileText: frontendApi,
      pattern: "post_id: postId",
      problem: "createComment does not send post_id",
    },
    {
      fileText: frontendApi,
      pattern: "content,",
      problem: "createComment does not send content",
    },
    {
      fileText: frontendApi,
      pattern: "export function toggleLike(postId: string | number)",
      problem: "frontend toggleLike helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: 'apiPost<{ liked: boolean }>("/api/likes",',
      problem: "toggleLike does not call POST /api/likes",
    },
    {
      fileText: frontendApi,
      pattern: "target_type: 1",
      problem: "toggleLike does not send the post target type",
    },
    {
      fileText: frontendApi,
      pattern: "target_id: postId",
      problem: "toggleLike does not send target_id",
    },
    {
      fileText: frontendApi,
      pattern: "export function toggleCollect(postId: string | number)",
      problem: "frontend toggleCollect helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: 'apiPost<{ collected: boolean }>(`/api/posts/${postId}/collect`)',
      problem: "toggleCollect does not call POST /api/posts/:id/collect",
    },
    {
      fileText: frontendApi,
      pattern: "export async function getOptionalFollowStatus(userId: string | number)",
      problem: "optional follow-status helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: "{ auth: false, context: { token } }",
      problem: "optional protected status helpers do not pass an explicit token without forcing redirects",
    },
    {
      fileText: frontendApi,
      pattern: "if (error instanceof ApiUnauthorizedError)",
      problem: "optional protected status helpers do not swallow unauthorized states",
    },
    {
      fileText: frontendApi,
      pattern: "return null;",
      problem: "optional protected status helpers do not return null without a token",
    },
    {
      fileText: frontendApi,
      pattern: "export function followUser(userId: string | number)",
      problem: "frontend followUser helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: "export function unfollowUser(userId: string | number)",
      problem: "frontend unfollowUser helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: "`/api/users/${userId}/follow`",
      problem: "follow/unfollow helpers do not target /api/users/:id/follow",
    },
    {
      fileText: frontendApi,
      pattern: "export function toggleDislike(postId: string | number)",
      problem: "frontend toggleDislike helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: 'apiPost<DislikeStatusPayload>("/api/dislikes", { post_id: postId })',
      problem: "toggleDislike does not call POST /api/dislikes with post_id",
    },
    {
      fileText: frontendApi,
      pattern: "export async function getOptionalDislikeStatus(postId: string | number)",
      problem: "optional dislike-status helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: "export function createReport(input:",
      problem: "frontend createReport helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: 'apiPost<CreateReportPayload>("/api/reports",',
      problem: "createReport does not call POST /api/reports",
    },
    {
      fileText: frontendApi,
      pattern: "target_type: input.targetType",
      problem: "createReport does not send target_type",
    },
    {
      fileText: frontendApi,
      pattern: "target_id: input.targetId",
      problem: "createReport does not send target_id",
    },
    {
      fileText: frontendApi,
      pattern: "description: input.description?.trim() || undefined",
      problem: "createReport does not trim optional report descriptions",
    },
    {
      fileText: frontendApi,
      pattern: "export async function getOptionalReportStatus(input:",
      problem: "optional report-status helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: '"/api/reports/check"',
      problem: "report status helpers do not target /api/reports/check",
    },
    {
      fileText: exploreFeed,
      pattern: "const detail = await getPostDetail(post.id)",
      problem: "explore feed does not hydrate opened posts from getPostDetail",
    },
    {
      fileText: exploreFeed,
      pattern: "const result = await toggleLike(post.id)",
      problem: "explore feed does not call toggleLike for like actions",
    },
    {
      fileText: exploreFeed,
      pattern: "const result = await toggleCollect(post.id)",
      problem: "explore feed does not call toggleCollect for collect actions",
    },
    {
      fileText: exploreFeed,
      pattern: "<PostDetailDrawer",
      problem: "explore feed does not render PostDetailDrawer",
    },
    {
      fileText: exploreFeed,
      pattern: "onCollect={handleCollect}",
      problem: "explore feed does not wire collect actions into PostDetailDrawer",
    },
    {
      fileText: exploreFeed,
      pattern: "onLike={handleLike}",
      problem: "explore feed does not wire like actions into PostDetailDrawer",
    },
    {
      fileText: postDetailDrawer,
      pattern: "getPostComments(postId)",
      problem: "post detail drawer does not load comments",
    },
    {
      fileText: postDetailDrawer,
      pattern: "const comment = await createComment(post.id, content)",
      problem: "post detail drawer does not submit comments through createComment",
    },
    {
      fileText: postDetailDrawer,
      pattern: "getOptionalFollowStatus(authorUserId)",
      problem: "post detail drawer does not load optional follow status",
    },
    {
      fileText: postDetailDrawer,
      pattern: "await followUser(authorUserId)",
      problem: "post detail drawer does not follow via followUser",
    },
    {
      fileText: postDetailDrawer,
      pattern: "await unfollowUser(authorUserId)",
      problem: "post detail drawer does not unfollow via unfollowUser",
    },
    {
      fileText: postDetailDrawer,
      pattern: "getOptionalDislikeStatus(postId)",
      problem: "post detail drawer does not load optional dislike status",
    },
    {
      fileText: postDetailDrawer,
      pattern: 'getOptionalReportStatus({ targetType: "post", targetId: postId })',
      problem: "post detail drawer does not load optional report status",
    },
    {
      fileText: postDetailDrawer,
      pattern: "const result = await toggleDislike(post.id)",
      problem: "post detail drawer does not toggle dislikes through toggleDislike",
    },
    {
      fileText: postDetailDrawer,
      pattern: 'await createReport({',
      problem: "post detail drawer does not submit reports through createReport",
    },
    {
      fileText: postDetailDrawer,
      pattern: 'targetType: "post"',
      problem: "post detail drawer does not report post targets",
    },
    {
      fileText: postDetailDrawer,
      pattern: "description: reportDescription",
      problem: "post detail drawer does not pass report descriptions",
    },
    {
      fileText: postDetailDrawer,
      pattern: "maxLength={300}",
      problem: "post detail drawer does not bound report description length",
    },
    {
      fileText: postDetailDrawer,
      pattern: "onClick={() => onLike?.(post)}",
      problem: "post detail drawer footer like action is not wired",
    },
    {
      fileText: postDetailDrawer,
      pattern: "onClick={() => onCollect?.(post)}",
      problem: "post detail drawer footer collect action is not wired",
    },
  ];

  for (const check of requiredPatterns) {
    if (!check.fileText.includes(check.pattern)) {
      problems.push(check.problem);
    }
  }

  addCheck(
    checks,
    "frontend-content-interaction-contract",
    problems.length === 0 ? "pass" : "fail",
    problems.length === 0
      ? "frontend content detail and interaction contract is aligned with backend routes"
      : "frontend content detail and interaction contract is not aligned with backend routes",
    {
      ...details,
      problems,
    },
  );
}

