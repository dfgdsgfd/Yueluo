"use client";

import "shaka-player/dist/controls.css";
import { useEffect, useRef, useState } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

type ShakaPlayerErrorEvent = Event & {
  detail?: unknown;
};

type ShakaErrorLike = {
  category?: unknown;
  code?: unknown;
  data?: unknown;
  message?: unknown;
  severity?: unknown;
};

type ShakaPlayerConstructor = {
  new (mediaElement?: HTMLMediaElement | null): {
    attach: (mediaElement: HTMLMediaElement) => Promise<unknown>;
    configure: (config: string | object, value?: unknown) => boolean;
    destroy: () => Promise<unknown>;
    load: (assetUri: string) => Promise<unknown>;
    addEventListener: (
      type: "error",
      listener: (event: ShakaPlayerErrorEvent) => void,
    ) => void;
    removeEventListener: (
      type: "error",
      listener: (event: ShakaPlayerErrorEvent) => void,
    ) => void;
  };
  isBrowserSupported?: () => boolean;
};

type ShakaOverlayConstructor = {
  new (
    player: unknown,
    videoContainer: HTMLElement,
    video: HTMLMediaElement,
  ): {
    configure: (config: object) => void;
    destroy: (forceDisconnect?: boolean) => Promise<unknown>;
  };
};

type ShakaModule = {
  default: {
    Player: ShakaPlayerConstructor;
    polyfill?: {
      installAll?: () => void;
    };
    ui: {
      Overlay: ShakaOverlayConstructor;
    };
  };
};

export function ShakaVideoPlayer({
  src,
  poster,
  className,
}: {
  src?: string | null;
  poster?: string | null;
  className?: string;
}) {
  const t = useTranslations("video");
  const containerRef = useRef<HTMLDivElement | null>(null);
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const [isLoading, setIsLoading] = useState(Boolean(src));
  const [hasError, setHasError] = useState(false);

  useEffect(() => {
    const container = containerRef.current;
    const video = videoRef.current;

    if (!src || !container || !video) {
      setIsLoading(false);
      return;
    }

    const assetUri = src;
    const playerContainer = container;
    const mediaElement = video;
    let isMounted = true;
    let hasReportedError = false;
    let player: InstanceType<ShakaPlayerConstructor> | null = null;
    let overlay: InstanceType<ShakaOverlayConstructor> | null = null;

    function handlePlayerError(error?: unknown) {
      if (!isMounted || hasReportedError) {
        return;
      }

      hasReportedError = true;
      console.debug("Shaka player failed to load", getShakaErrorSummary(error));
      setIsLoading(false);
      setHasError(true);
      toast.error(t("error"));
    }

    async function loadPlayer() {
      try {
        setHasError(false);
        setIsLoading(true);

        const shaka = (
          (await import("shaka-player/dist/shaka-player.ui")) as unknown as ShakaModule
        ).default;
        shaka.polyfill?.installAll?.();

        if (shaka.Player.isBrowserSupported?.() === false) {
          throw new Error("Shaka Player is not supported in this browser.");
        }

        player = new shaka.Player();
        player.addEventListener("error", handlePlayerError);
        player.configure({
          abr: {
            enabled: true,
          },
          streaming: {
            retryParameters: {
              maxAttempts: 3,
            },
          },
        });

        await player.attach(mediaElement);

        overlay = new shaka.ui.Overlay(player, playerContainer, mediaElement);
        overlay.configure({
          addSeekBar: true,
          clearBufferOnQualityChange: true,
          controlPanelElements: [
            "play_pause",
            "mute",
            "volume",
            "time_and_duration",
            "spacer",
            "overflow_menu",
            "fullscreen",
          ],
          enableTooltips: true,
          overflowMenuButtons: [
            "quality",
            "language",
            "captions",
            "playback_rate",
            "picture_in_picture",
          ],
          playbackRates: [0.5, 0.75, 1, 1.25, 1.5, 1.75, 2],
          qualityMarks: {
            720: "HD",
            1080: "FHD",
            1440: "2K",
            2160: "4K",
            4320: "8K",
          },
          seekBarColors: {
            adBreaks: "rgb(255, 204, 0)",
            base: "rgba(255, 255, 255, 0.25)",
            buffered: "rgba(255, 255, 255, 0.45)",
            chapters: "rgba(255, 36, 66, 0.8)",
            played: "rgb(255, 36, 66)",
          },
          showAudioCodec: true,
          showVideoCodec: true,
          volumeBarColors: {
            base: "rgba(255, 255, 255, 0.45)",
            level: "rgb(255, 36, 66)",
          },
        });

        await player.load(assetUri);
        mediaElement.muted = false;
        mediaElement.volume = 1;
        void mediaElement.play().catch(() => {
          // Browsers may block autoplay with sound; controls remain available.
        });

        if (isMounted) {
          setIsLoading(false);
        }
      } catch (error) {
        handlePlayerError(error);
      }
    }

    void loadPlayer();

    return () => {
      isMounted = false;
      player?.removeEventListener("error", handlePlayerError);
      void overlay?.destroy();
      void player?.destroy();
    };
  }, [src, t]);

  return (
    <div
      aria-busy={isLoading}
      data-state={hasError ? "error" : isLoading ? "loading" : "ready"}
      className={cn(
        "shaka-player-shell relative overflow-hidden rounded-2xl bg-black",
        "aspect-[16/10]",
        className,
      )}
    >
      {src ? (
        <div ref={containerRef} className="shaka-video-container size-full">
          <video
            ref={videoRef}
            className="size-full object-contain"
            autoPlay
            playsInline
            preload="metadata"
            poster={poster ?? undefined}
          />
        </div>
      ) : (
        <div className="flex size-full items-center justify-center px-6 text-center text-sm text-white/70">
          {t("unavailable")}
        </div>
      )}
      {(isLoading || hasError) && (
        <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center gap-3 bg-black/55 px-6 text-center text-sm font-medium text-white backdrop-blur-[2px] transition-opacity duration-300">
          {hasError ? (
            <span className="inline-flex max-w-full items-center rounded-full border border-white/10 bg-white/[0.08] px-4 py-2 text-white/86">
              {t("error")}
            </span>
          ) : (
            <>
              <span className="size-9 rounded-full border-2 border-white/20 border-t-primary animate-spin" />
              <span className="text-white/78">{t("loading")}</span>
            </>
          )}
        </div>
      )}
    </div>
  );
}

function getShakaErrorSummary(error: unknown) {
  const detail =
    isRecord(error) && "detail" in error
      ? error.detail
      : error;

  if (isRecord(detail)) {
    const shakaError = detail as ShakaErrorLike;

    return {
      category: shakaError.category,
      code: shakaError.code,
      data: shakaError.data,
      message: shakaError.message,
      severity: shakaError.severity,
    };
  }

  if (detail instanceof Error) {
    return {
      message: detail.message,
      name: detail.name,
    };
  }

  return { message: String(detail ?? "Unknown Shaka error") };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
