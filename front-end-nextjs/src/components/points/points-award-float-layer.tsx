"use client";

import { motion } from "framer-motion";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  consumePendingPointsAwards,
  POINTS_AWARD_EVENT,
  normalizePointsAward,
  type PointsAwardEventPayload,
} from "@/lib/points-award-events";

const FLOAT_DURATION_SECONDS = 2.2;
const FLOAT_DURATION_MS = 2400;

type FloatingAward = PointsAwardEventPayload & {
  color: string;
  drift: number;
  id: number;
  x: number;
  y: number;
};

function createFloatingAward(award: PointsAwardEventPayload, id: number): FloatingAward {
  const width = window.innerWidth || 390;
  const height = window.innerHeight || 720;
  const mobile = width < 768;
  const minX = width * (mobile ? 0.18 : 0.34);
  const rangeX = width * (mobile ? 0.64 : 0.32);
  const x = minX + Math.random() * rangeX;
  const y = height * (mobile ? 0.62 : 0.64);
  const drift = (Math.random() - 0.5) * (mobile ? 34 : 64);

  return {
    ...award,
    color: award.amount >= 50 ? "#eab308" : "#22c55e",
    drift,
    id,
    x,
    y,
  };
}

function formatAwardAmount(amount: number) {
  return new Intl.NumberFormat(undefined, {
    maximumFractionDigits: Number.isInteger(amount) ? 0 : 2,
  }).format(amount);
}

export function PointsAwardFloatLayer() {
  const [awards, setAwards] = useState<FloatingAward[]>([]);
  const nextIdRef = useRef(0);
  const timersRef = useRef(new Map<number, ReturnType<typeof setTimeout>>());

  const removeAward = useCallback((id: number) => {
    const timer = timersRef.current.get(id);
    if (timer) {
      clearTimeout(timer);
      timersRef.current.delete(id);
    }
    setAwards((current) => current.filter((item) => item.id !== id));
  }, []);

  const pushAward = useCallback(
    (award: PointsAwardEventPayload) => {
      const id = nextIdRef.current + 1;
      nextIdRef.current = id;
      const item = createFloatingAward(award, id);

      setAwards((current) => [...current.slice(-5), item]);
      const timer = setTimeout(() => removeAward(id), FLOAT_DURATION_MS);
      timersRef.current.set(id, timer);
    },
    [removeAward],
  );

  useEffect(() => {
    const timers = timersRef.current;
    for (const award of consumePendingPointsAwards()) {
      pushAward(award);
    }

    function handlePointsAward(event: Event) {
      consumePendingPointsAwards();
      const detail = "detail" in event ? event.detail : null;
      const award = normalizePointsAward(detail);
      if (award) {
        pushAward(award);
      }
    }

    window.addEventListener(POINTS_AWARD_EVENT, handlePointsAward);
    return () => {
      window.removeEventListener(POINTS_AWARD_EVENT, handlePointsAward);
      for (const timer of timers.values()) {
        clearTimeout(timer);
      }
      timers.clear();
    };
  }, [pushAward]);

  if (!awards.length) {
    return null;
  }

  return (
    <div aria-live="polite" className="pointer-events-none fixed inset-0 z-[9999] overflow-hidden">
      {awards.map((award) => (
        <div
          key={award.id}
          className="pointer-events-none absolute"
          style={{ left: `${award.x}px`, top: `${award.y}px`, transform: "translate(-50%, -50%)" }}
        >
          <motion.div
            animate={{
              opacity: [0, 1, 1, 0],
              scale: [0.72, 1.08, 1, 0.98],
              x: [0, award.drift],
              y: [0, -110],
            }}
            className="flex flex-col items-center text-center drop-shadow-[0_18px_30px_rgba(0,0,0,0.45)] will-change-transform"
            initial={{ opacity: 0, scale: 0.72, x: 0, y: 0 }}
            transition={{
              duration: FLOAT_DURATION_SECONDS,
              ease: [0.23, 1, 0.32, 1],
              times: [0, 0.16, 0.72, 1],
            }}
            style={{ color: award.color }}
          >
            <span className="text-[clamp(1.875rem,8.5vw,2.75rem)] font-black leading-none md:text-[clamp(2.5rem,5vw,3.5rem)]">
              +{formatAwardAmount(award.amount)}
            </span>
            {award.reason ? (
              <span className="mt-1.5 max-w-[min(72vw,240px)] break-words rounded-full bg-[#111217]/85 px-2.5 py-1 text-[clamp(0.6875rem,2.8vw,0.8125rem)] font-bold leading-snug text-white shadow-[0_10px_24px_rgba(0,0,0,0.28)] ring-1 ring-white/15 backdrop-blur-md md:mt-2 md:max-w-[280px] md:px-3 md:py-1.5 md:text-[0.875rem]">
                {award.reason}
              </span>
            ) : null}
          </motion.div>
        </div>
      ))}
    </div>
  );
}
