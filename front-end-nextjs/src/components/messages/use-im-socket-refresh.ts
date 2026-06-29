"use client";

import { useEffect, useRef, type Dispatch, type SetStateAction } from "react";
import { getImWebSocketUrl } from "@/lib/api/im";
import { IM_BACKGROUND_REQUEST_OPTIONS } from "@/lib/im-background-request";
import {
  getImSocketConversationId,
  parseImSocketMessage,
} from "@/lib/im-unread";
import type { SocketState } from "./messages-page-view";

const socketRefreshDebounceMs = 350;
const socketInitialReconnectMs = 2500;
const socketMaxReconnectMs = 30000;
const socketSessionRetryMs = 5000;

type SilentRefreshOptions = {
  silent?: boolean;
};

export function useImSocketRefresh({
  authToken,
  loadConversations,
  loadMessages,
  selectedConversationIdRef,
  setSocketState,
}: {
  authToken: string | null | undefined;
  loadConversations: (options?: SilentRefreshOptions) => Promise<void>;
  loadMessages: (
    conversationId: string | number,
    options?: SilentRefreshOptions,
  ) => Promise<void>;
  selectedConversationIdRef: { current: string | number | null };
  setSocketState: Dispatch<SetStateAction<SocketState>>;
}) {
  const pendingSocketConversationIdsRef = useRef(new Set<string>());
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const socketRefreshTimerRef = useRef<ReturnType<typeof setTimeout> | null>(
    null,
  );
  const socketReconnectDelayRef = useRef(socketInitialReconnectMs);
  const socketRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!authToken) {
      setSocketState("idle");
      return;
    }

    const socketToken = authToken;
    let closedByEffect = false;
    let socketActive = document.visibilityState !== "hidden";
    let connectInFlight = false;
    const pendingSocketConversationIds =
      pendingSocketConversationIdsRef.current;

    function shouldConnect() {
      return !closedByEffect && socketActive;
    }

    function isSocketConnectingOrOpen(socket: WebSocket | null) {
      return (
        socket?.readyState === WebSocket.CONNECTING ||
        socket?.readyState === WebSocket.OPEN
      );
    }

    function clearReconnectTimer() {
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
        reconnectTimerRef.current = null;
      }
    }

    function clearSocketRefreshTimer() {
      if (socketRefreshTimerRef.current) {
        clearTimeout(socketRefreshTimerRef.current);
        socketRefreshTimerRef.current = null;
      }
    }

    function scheduleReconnect(connect: () => Promise<void>, delayMs: number) {
      if (!shouldConnect()) {
        return;
      }
      clearReconnectTimer();
      reconnectTimerRef.current = setTimeout(() => {
        reconnectTimerRef.current = null;
        if (shouldConnect()) {
          void connect();
        }
      }, delayMs);
      socketReconnectDelayRef.current = Math.min(
        delayMs * 2,
        socketMaxReconnectMs,
      );
    }

    function scheduleSocketRefresh(
      conversationId: string | number | undefined,
    ) {
      if (conversationId !== undefined) {
        pendingSocketConversationIds.add(String(conversationId));
      }
      if (socketRefreshTimerRef.current) {
        return;
      }
      socketRefreshTimerRef.current = setTimeout(() => {
        socketRefreshTimerRef.current = null;
        if (!shouldConnect()) {
          return;
        }
        const conversationIds = [...pendingSocketConversationIds];
        pendingSocketConversationIds.clear();
        void loadConversations({ silent: true });
        const currentConversationId = selectedConversationIdRef.current;
        if (
          currentConversationId !== null &&
          conversationIds.some((id) => id === String(currentConversationId))
        ) {
          void loadMessages(currentConversationId, { silent: true });
        }
      }, socketRefreshDebounceMs);
    }

    function closeSocket(updateState = true) {
      const socket = socketRef.current;
      socketRef.current = null;
      if (socket) {
        socket.close();
      }
      if (updateState) {
        setSocketState("closed");
      }
    }

    function stopSocketActivity() {
      clearReconnectTimer();
      clearSocketRefreshTimer();
      pendingSocketConversationIds.clear();
      closeSocket();
    }

    function refreshVisibleMessages() {
      void loadConversations({ silent: true });
      const currentConversationId = selectedConversationIdRef.current;
      if (currentConversationId !== null) {
        void loadMessages(currentConversationId, { silent: true });
      }
    }

    async function connect() {
      if (!shouldConnect() || connectInFlight) {
        return;
      }
      const existingSocket = socketRef.current;
      if (isSocketConnectingOrOpen(existingSocket)) {
        return;
      }
      if (existingSocket) {
        socketRef.current = null;
      }
      clearReconnectTimer();
      setSocketState("connecting");
      connectInFlight = true;
      let wsUrl: string;
      try {
        wsUrl = await getImWebSocketUrl(
          socketToken,
          IM_BACKGROUND_REQUEST_OPTIONS,
        );
      } catch {
        if (shouldConnect()) {
          scheduleReconnect(connect, socketSessionRetryMs);
        }
        connectInFlight = false;
        return;
      }
      connectInFlight = false;
      if (!shouldConnect()) {
        return;
      }
      const socket = new WebSocket(wsUrl);
      socketRef.current = socket;

      socket.onopen = () => {
        if (socketRef.current !== socket) {
          return;
        }
        socketReconnectDelayRef.current = socketInitialReconnectMs;
        setSocketState("open");
      };

      socket.onmessage = (event) => {
        if (socketRef.current !== socket || !shouldConnect()) {
          return;
        }
        const payload = parseImSocketMessage(event.data);
        if (payload?.type !== "new_message") {
          return;
        }

        scheduleSocketRefresh(getImSocketConversationId(payload));
      };

      socket.onerror = () => {
        if (socketRef.current === socket) {
          setSocketState("closed");
        }
      };

      socket.onclose = () => {
        const isCurrentSocket = socketRef.current === socket;
        if (isCurrentSocket) {
          socketRef.current = null;
        }
        if (isCurrentSocket || !shouldConnect()) {
          setSocketState("closed");
        }
        if (isCurrentSocket && shouldConnect()) {
          scheduleReconnect(connect, socketReconnectDelayRef.current);
        }
      };
    }

    function handleVisibilityChange() {
      if (document.visibilityState === "hidden") {
        socketActive = false;
        stopSocketActivity();
        return;
      }
      socketActive = true;
      socketReconnectDelayRef.current = socketInitialReconnectMs;
      refreshVisibleMessages();
      void connect();
    }

    function handlePageHide() {
      socketActive = false;
      stopSocketActivity();
    }

    if (socketActive) {
      void connect();
    } else {
      setSocketState("closed");
    }
    document.addEventListener("visibilitychange", handleVisibilityChange);
    window.addEventListener("pagehide", handlePageHide);

    return () => {
      closedByEffect = true;
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      window.removeEventListener("pagehide", handlePageHide);
      clearReconnectTimer();
      clearSocketRefreshTimer();
      pendingSocketConversationIds.clear();
      closeSocket(false);
    };
  }, [
    authToken,
    loadConversations,
    loadMessages,
    selectedConversationIdRef,
    setSocketState,
  ]);
}
