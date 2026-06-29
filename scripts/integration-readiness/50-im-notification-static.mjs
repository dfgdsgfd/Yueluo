function checkImWebSocketStaticContract(checks, frontendEnv) {
  const problems = [];
  const details = {
    routeMatrixPath: path.relative(repoRoot, backendRouteMatrixPath),
    frontendApiPath: path.relative(repoRoot, frontendApiPath),
    frontendMessagesPagePath: path.relative(repoRoot, frontendMessagesPagePath),
  };

  try {
    const matrix = JSON.parse(fs.readFileSync(backendRouteMatrixPath, "utf8"));
    const webSockets = Array.isArray(matrix.webSockets) ? matrix.webSockets : [];
    const imWebSocket = webSockets.find((entry) => isRecord(entry) && entry.path === "/api/im/ws");
    if (!imWebSocket) {
      problems.push("backend route matrix does not expose /api/im/ws");
    } else {
      details.backendWebSocket = {
        path: imWebSocket.path,
        auth: imWebSocket.auth ?? null,
        status: imWebSocket.status ?? null,
      };
      if (imWebSocket.auth !== "query-token-and-redis-session") {
        problems.push("backend /api/im/ws auth mode is not query-token-and-redis-session");
      }
    }
  } catch (error) {
    problems.push(`backend route matrix cannot be read: ${error.message}`);
  }

  const frontendChecks = [
    {
      filePath: frontendApiPath,
      pattern: "export function getImWebSocketUrl(token: string)",
      problem: "frontend API helper getImWebSocketUrl is missing",
    },
    {
      filePath: frontendApiPath,
      pattern: 'new URL("/api/im/ws", base)',
      problem: "frontend WebSocket helper does not target /api/im/ws",
    },
    {
      filePath: frontendApiPath,
      pattern: 'url.protocol = url.protocol === "https:" ? "wss:" : "ws:"',
      problem: "frontend WebSocket helper does not convert http/https to ws/wss",
    },
    {
      filePath: frontendApiPath,
      pattern: 'url.searchParams.set("token", token)',
      problem: "frontend WebSocket helper does not pass token as query parameter",
    },
    {
      filePath: frontendMessagesPagePath,
      pattern: "new WebSocket(getImWebSocketUrl(socketToken))",
      problem: "messages page does not use getImWebSocketUrl to open the socket",
    },
  ];

  for (const check of frontendChecks) {
    if (!fileIncludes(check.filePath, check.pattern)) {
      problems.push(check.problem);
    }
  }

  const sampleBase =
    frontendEnv.NEXT_PUBLIC_API_BASE_URL ||
    frontendEnv.NEXT_PUBLIC_BACKEND_ORIGIN ||
    "http://localhost:3001";
  try {
    const sampleUrl = new URL("/api/im/ws", sampleBase);
    sampleUrl.protocol = sampleUrl.protocol === "https:" ? "wss:" : "ws:";
    sampleUrl.searchParams.set("token", "<redacted>");
    details.sampleWebSocketUrl = sampleUrl.toString();
  } catch {
    problems.push("frontend WebSocket base URL cannot be resolved from env/fallback");
  }

  addCheck(
    checks,
    "frontend-im-websocket-contract",
    problems.length === 0 ? "pass" : "fail",
    problems.length === 0
      ? "IM WebSocket frontend/backend contract is statically aligned"
      : "IM WebSocket frontend/backend contract is not aligned",
    {
      ...details,
      problems,
    },
  );
}

function checkFrontendNotificationsImStaticContract(checks) {
  const problems = [];
  const details = {
    routeMatrixPath: path.relative(repoRoot, backendRouteMatrixPath),
    frontendApiPath: path.relative(repoRoot, frontendApiPath),
    frontendMessagesPagePath: path.relative(repoRoot, frontendMessagesPagePath),
    frontendMessagesRoutePath: path.relative(repoRoot, frontendMessagesRoutePath),
    frontendNotificationsPagePath: path.relative(repoRoot, frontendNotificationsPagePath),
    frontendNotificationsRoutePath: path.relative(repoRoot, frontendNotificationsRoutePath),
    frontendSystemNotificationPopupPath: path.relative(repoRoot, frontendSystemNotificationPopupPath),
    frontendRootLayoutPath: path.relative(repoRoot, frontendRootLayoutPath),
    frontendProfilePagePath: path.relative(repoRoot, frontendProfilePagePath),
  };

  const frontendApi = fileText(frontendApiPath);
  const rootLayout = fileText(frontendRootLayoutPath);
  const messagesPage = fileText(frontendMessagesPagePath);
  const messagesRoute = fileText(frontendMessagesRoutePath);
  const notificationsPage = fileText(frontendNotificationsPagePath);
  const notificationsRoute = fileText(frontendNotificationsRoutePath);
  const systemNotificationPopup = fileText(frontendSystemNotificationPopupPath);
  const profilePage = fileText(frontendProfilePagePath);

  try {
    const matrix = JSON.parse(fs.readFileSync(backendRouteMatrixPath, "utf8"));
    const routes = Array.isArray(matrix.routes) ? matrix.routes.filter(isRecord) : [];
    const expectedRoutes = [
      { method: "GET", path: "/api/notifications", auth: "user" },
      { method: "GET", path: "/api/notifications/unread-count", auth: "user" },
      { method: "PUT", path: "/api/notifications/read-all", auth: "user" },
      { method: "PUT", path: "/api/notifications/:id/read", auth: "user" },
      { method: "DELETE", path: "/api/notifications/:id", auth: "user" },
      { method: "GET", path: "/api/notifications/system", auth: "user" },
      { method: "GET", path: "/api/notifications/system/popup", auth: "user" },
      { method: "POST", path: "/api/notifications/system/:id/confirm", auth: "user" },
      { method: "DELETE", path: "/api/notifications/system/:id/dismiss", auth: "user" },
      { method: "GET", path: "/api/im/conversations", auth: "user" },
      { method: "POST", path: "/api/im/conversations", auth: "user" },
      { method: "GET", path: "/api/im/conversations/:id/messages", auth: "user" },
      { method: "POST", path: "/api/im/conversations/:id/messages", auth: "user" },
      { method: "POST", path: "/api/im/messages/:id/read", auth: "user" },
      { method: "GET", path: "/api/im/sync", auth: "user" },
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

    details.notificationsImRoutes = routeResults;
  } catch (error) {
    problems.push(`backend route matrix cannot be read: ${error.message}`);
  }

  const requiredPatterns = [
    {
      fileText: frontendApi,
      pattern: "export function getNotificationUnreadCount()",
      problem: "frontend getNotificationUnreadCount helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<NotificationUnreadCountPayload>("/api/notifications/unread-count")',
      problem: "getNotificationUnreadCount does not call /api/notifications/unread-count",
    },
    {
      fileText: frontendApi,
      pattern: "export function getNotifications(query:",
      problem: "frontend getNotifications helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<NotificationListPayload>("/api/notifications"',
      problem: "getNotifications does not call /api/notifications",
    },
    {
      fileText: frontendApi,
      pattern: "is_read: query.is_read",
      problem: "getNotifications does not pass notification read-state filters",
    },
    {
      fileText: frontendApi,
      pattern: "export function getSystemNotifications(query:",
      problem: "frontend getSystemNotifications helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<SystemNotificationListPayload>("/api/notifications/system"',
      problem: "getSystemNotifications does not call /api/notifications/system",
    },
    {
      fileText: frontendApi,
      pattern: "export function getPopupSystemNotifications()",
      problem: "frontend getPopupSystemNotifications helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<SystemNotificationItem[]>("/api/notifications/system/popup")',
      problem: "getPopupSystemNotifications does not call /api/notifications/system/popup",
    },
    {
      fileText: frontendApi,
      pattern: "export async function getOptionalPopupSystemNotifications()",
      problem: "frontend optional popup system notifications helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: '{ auth: false, context: { token } }',
      problem: "optional popup system notifications helper does not avoid global login redirects",
    },
    {
      fileText: rootLayout,
      pattern: 'import { SystemNotificationPopup } from "@/components/notifications/system-notification-popup"',
      problem: "root layout does not import SystemNotificationPopup",
    },
    {
      fileText: rootLayout,
      pattern: "<SystemNotificationPopup />",
      problem: "root layout does not render SystemNotificationPopup",
    },
    {
      fileText: systemNotificationPopup,
      pattern: "getOptionalPopupSystemNotifications()",
      problem: "SystemNotificationPopup does not load popup notifications",
    },
    {
      fileText: systemNotificationPopup,
      pattern: "await confirmSystemNotification(notification.id)",
      problem: "SystemNotificationPopup does not confirm popup notifications",
    },
    {
      fileText: systemNotificationPopup,
      pattern: "await dismissSystemNotification(notification.id)",
      problem: "SystemNotificationPopup does not dismiss popup notifications",
    },
    {
      fileText: frontendApi,
      pattern: "export function markNotificationRead(notificationId:",
      problem: "frontend markNotificationRead helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: "apiPut(`/api/notifications/${notificationId}/read`)",
      problem: "markNotificationRead does not call PUT /api/notifications/:id/read",
    },
    {
      fileText: frontendApi,
      pattern: 'apiPut("/api/notifications/read-all")',
      problem: "markAllNotificationsRead does not call PUT /api/notifications/read-all",
    },
    {
      fileText: frontendApi,
      pattern: "apiDelete(`/api/notifications/${notificationId}`)",
      problem: "deleteNotification does not call DELETE /api/notifications/:id",
    },
    {
      fileText: frontendApi,
      pattern: "apiPost(`/api/notifications/system/${notificationId}/confirm`)",
      problem: "confirmSystemNotification does not call POST /api/notifications/system/:id/confirm",
    },
    {
      fileText: frontendApi,
      pattern: "apiDelete(`/api/notifications/system/${notificationId}/dismiss`)",
      problem: "dismissSystemNotification does not call DELETE /api/notifications/system/:id/dismiss",
    },
    {
      fileText: notificationsRoute,
      pattern: 'import { NotificationsPage } from "@/components/notifications/notifications-page"',
      problem: "notifications route does not render NotificationsPage",
    },
    {
      fileText: notificationsPage,
      pattern: "Promise.all([",
      problem: "notifications page does not load notification payloads together",
    },
    {
      fileText: notificationsPage,
      pattern: "getNotifications({ limit: 30 })",
      problem: "notifications page does not load user notifications",
    },
    {
      fileText: notificationsPage,
      pattern: "getSystemNotifications({ limit: 30 })",
      problem: "notifications page does not load system notifications",
    },
    {
      fileText: notificationsPage,
      pattern: "getNotificationUnreadCount()",
      problem: "notifications page does not load unread counts",
    },
    {
      fileText: notificationsPage,
      pattern: "await markAllNotificationsRead()",
      problem: "notifications page does not wire mark-all-read action",
    },
    {
      fileText: notificationsPage,
      pattern: "await markNotificationRead(notification.id)",
      problem: "notifications page does not wire mark-read action",
    },
    {
      fileText: notificationsPage,
      pattern: "await deleteNotification(notification.id)",
      problem: "notifications page does not wire delete action",
    },
    {
      fileText: notificationsPage,
      pattern: "await confirmSystemNotification(notification.id)",
      problem: "notifications page does not wire system confirmation action",
    },
    {
      fileText: notificationsPage,
      pattern: "await dismissSystemNotification(notification.id)",
      problem: "notifications page does not wire system dismiss action",
    },
    {
      fileText: notificationsPage,
      pattern: 'Link href="/messages"',
      problem: "notifications page does not link to messages",
    },
    {
      fileText: frontendApi,
      pattern: "export function getImConversations()",
      problem: "frontend getImConversations helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<ImConversation[]>("/api/im/conversations")',
      problem: "getImConversations does not call GET /api/im/conversations",
    },
    {
      fileText: frontendApi,
      pattern: "export function createImConversation(memberIds:",
      problem: "frontend createImConversation helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: 'apiPost<ImConversation>("/api/im/conversations"',
      problem: "createImConversation does not call POST /api/im/conversations",
    },
    {
      fileText: frontendApi,
      pattern: "member_ids: memberIds",
      problem: "createImConversation does not send member_ids",
    },
    {
      fileText: frontendApi,
      pattern: "export async function getImMessages(",
      problem: "frontend getImMessages helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: "`/api/im/conversations/${conversationId}/messages`",
      problem: "getImMessages does not call /api/im/conversations/:id/messages",
    },
    {
      fileText: frontendApi,
      pattern: "before: query.before",
      problem: "getImMessages does not pass the before cursor",
    },
    {
      fileText: frontendApi,
      pattern: "isRecord(envelope.pagination)",
      problem: "getImMessages does not read top-level backend pagination",
    },
    {
      fileText: frontendApi,
      pattern: "export function sendImMessage(conversationId:",
      problem: "frontend sendImMessage helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: "apiPost<ImMessage>(`/api/im/conversations/${conversationId}/messages`",
      problem: "sendImMessage does not call POST /api/im/conversations/:id/messages",
    },
    {
      fileText: frontendApi,
      pattern: "client_msg_id:",
      problem: "sendImMessage does not send a client message id",
    },
    {
      fileText: frontendApi,
      pattern: "export function markImMessageRead(messageId:",
      problem: "frontend markImMessageRead helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: "apiPost(`/api/im/messages/${messageId}/read`)",
      problem: "markImMessageRead does not call POST /api/im/messages/:id/read",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<ImSyncPayload>("/api/im/sync"',
      problem: "syncImMessages does not call GET /api/im/sync",
    },
    {
      fileText: messagesRoute,
      pattern: 'import { MessagesPage } from "@/components/messages/messages-page"',
      problem: "messages route does not render MessagesPage",
    },
    {
      fileText: messagesPage,
      pattern: "const token = getStoredAccessToken()",
      problem: "messages page does not gate the view with the stored user token",
    },
    {
      fileText: messagesPage,
      pattern: "const nextConversations = await getImConversations()",
      problem: "messages page does not load conversations",
    },
    {
      fileText: messagesPage,
      pattern: "getRequestedConversationId()",
      problem: "messages page does not honor the requested conversation query parameter",
    },
    {
      fileText: messagesPage,
      pattern: "await getImMessages(conversationId, { limit: messagePageSize })",
      problem: "messages page does not load the selected conversation messages",
    },
    {
      fileText: messagesPage,
      pattern: "void markImMessageRead(latestIncoming.id)",
      problem: "messages page does not mark incoming messages read",
    },
    {
      fileText: messagesPage,
      pattern: "new WebSocket(getImWebSocketUrl(socketToken))",
      problem: "messages page does not open the IM WebSocket",
    },
    {
      fileText: messagesPage,
      pattern: 'payload?.type !== "new_message"',
      problem: "messages page does not filter WebSocket new_message events",
    },
    {
      fileText: messagesPage,
      pattern: "void loadConversations({ silent: true })",
      problem: "messages page does not refresh conversations on socket messages",
    },
    {
      fileText: messagesPage,
      pattern: "before: nextBefore",
      problem: "messages page does not load older messages with the before cursor",
    },
    {
      fileText: messagesPage,
      pattern: "const sentMessage = await sendImMessage(selectedConversationId, content)",
      problem: "messages page does not send messages through sendImMessage",
    },
    {
      fileText: messagesPage,
      pattern: 'actionHref="/login"',
      problem: "messages page does not send signed-out users to login",
    },
    {
      fileText: messagesPage,
      pattern: 'Link href="/notifications"',
      problem: "messages page does not link to notifications",
    },
    {
      fileText: profilePage,
      pattern: "const conversation = await createImConversation([profileState.id])",
      problem: "profile page does not create IM conversations from the user profile",
    },
    {
      fileText: profilePage,
      pattern: "router.push(`/messages?conversation=${conversation.id}`)",
      problem: "profile page does not deep-link into the created conversation",
    },
  ];

  for (const check of requiredPatterns) {
    if (!check.fileText.includes(check.pattern)) {
      problems.push(check.problem);
    }
  }

  addCheck(
    checks,
    "frontend-notifications-im-contract",
    problems.length === 0 ? "pass" : "fail",
    problems.length === 0
      ? "frontend notifications and IM page contract is aligned with backend routes"
      : "frontend notifications and IM page contract is not aligned with backend routes",
    {
      ...details,
      problems,
    },
  );
}

