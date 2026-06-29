"use client";
import {
ChevronLeft,
} from "lucide-react";
import {
Tone
} from "./types";

export function toneSoftClass(tone: Tone) {
  const classes: Record<Tone, string> = {
    red: "bg-[#fef2f2] text-[#dc2626]",
    green: "bg-[#eafff3] text-[#18a058]",
    blue: "bg-[#eef6ff] text-[#2f7df6]",
    purple: "bg-[#f3efff] text-[#7357d6]",
    amber: "bg-[#fff7e8] text-[#c97900]",
    slate: "bg-[#f0f2f6] text-[#5f6674]",
  };
  return classes[tone];
}


export function tonePillClass(tone: Tone) {
  const classes: Record<Tone, string> = {
    red: "bg-[#fff0f2] text-[#d71935]",
    green: "bg-[#eafff3] text-[#16824a]",
    blue: "bg-[#eef6ff] text-[#1e62ca]",
    purple: "bg-[#f3efff] text-[#6446c5]",
    amber: "bg-[#fff7e8] text-[#a96300]",
    slate: "bg-[#eef0f4] text-[#626977]",
  };
  return classes[tone];
}


export function toneTextClass(tone: Tone) {
  const classes: Record<Tone, string> = {
    red: "text-[#dc2626]",
    green: "text-[#18a058]",
    blue: "text-[#2f7df6]",
    purple: "text-[#7357d6]",
    amber: "text-[#c97900]",
    slate: "text-[#414856]",
  };
  return classes[tone];
}


export function toneDotClass(tone: Tone) {
  const classes: Record<Tone, string> = {
    red: "bg-[#dc2626]",
    green: "bg-[#18a058]",
    blue: "bg-[#2f7df6]",
    purple: "bg-[#7357d6]",
    amber: "bg-[#f59e0b]",
    slate: "bg-[#747b87]",
  };
  return classes[tone];
}


export function ArrowLeftIcon() {
  return (
    <span className="flex size-8 items-center justify-center rounded-lg bg-white/70">
      <ChevronLeft className="size-4" />
    </span>
  );
}
