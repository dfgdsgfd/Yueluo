"use client";
import {
  useEffect,
  useRef,
  useState
} from "react";
import {
  FEED_LOADING_MIN_VISIBLE_MS,
  FEED_SENTINEL_DEBOUNCE_MS
} from "./explore-config";

export function useGracefulLoading(active: boolean) {
  const [visible, setVisible] = useState(false);
  const shownAtRef = useRef(0);

  useEffect(() => {
    let timerId: number | null = null;

    if (active) {
      if (!visible) {
        shownAtRef.current = Date.now();
        timerId = window.setTimeout(() => {
          setVisible(true);
        }, 0);
      }
    } else if (visible) {
      const remaining = Math.max(
        0,
        FEED_LOADING_MIN_VISIBLE_MS - (Date.now() - shownAtRef.current),
      );
      timerId = window.setTimeout(() => setVisible(false), remaining);
    }

    return () => {
      if (timerId !== null) {
        window.clearTimeout(timerId);
      }
    };
  }, [active, visible]);

  return active || visible;
}


export function useFeedSentinel({
  enabled,
  onEnter,
  resetKey,
  rootMargin,
}: {
  enabled: boolean;
  onEnter: () => void;
  resetKey: string;
  rootMargin: string;
}) {
  const elementRef = useRef<HTMLDivElement>(null);
  const onEnterRef = useRef(onEnter);

  useEffect(() => {
    onEnterRef.current = onEnter;
  }, [onEnter]);

  useEffect(() => {
    const element = elementRef.current;
    if (!enabled || !element) {
      return;
    }

    let timerId: number | null = null;
    let triggeredForCurrentEntry = false;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (!entry.isIntersecting) {
          triggeredForCurrentEntry = false;
          if (timerId !== null) {
            window.clearTimeout(timerId);
            timerId = null;
          }
          return;
        }

        if (triggeredForCurrentEntry || timerId !== null) {
          return;
        }

        timerId = window.setTimeout(() => {
          timerId = null;
          triggeredForCurrentEntry = true;
          onEnterRef.current();
        }, FEED_SENTINEL_DEBOUNCE_MS);
      },
      { rootMargin },
    );

    observer.observe(element);
    return () => {
      observer.disconnect();
      if (timerId !== null) {
        window.clearTimeout(timerId);
      }
    };
  }, [enabled, resetKey, rootMargin]);

  return elementRef;
}


export type FeedAnchor = {
  id: string;
  top: number;
};


export function getVisibleFeedAnchor(): FeedAnchor | null {
  const elements = Array.from(
    document.querySelectorAll<HTMLElement>("[data-feed-post-id]"),
  );
  const element = elements.find((candidate) => candidate.getBoundingClientRect().bottom > 0);

  return element
    ? {
        id: element.dataset.feedPostId ?? "",
        top: element.getBoundingClientRect().top,
      }
    : null;
}


export function restoreFeedAnchor(anchor: FeedAnchor | null) {
  if (!anchor?.id) {
    return;
  }

  window.requestAnimationFrame(() => {
    window.requestAnimationFrame(() => {
      const element = Array.from(
        document.querySelectorAll<HTMLElement>("[data-feed-post-id]"),
      ).find((candidate) => candidate.dataset.feedPostId === anchor.id);
      if (!element) {
        return;
      }

      window.scrollBy({ top: element.getBoundingClientRect().top - anchor.top });
    });
  });
}
