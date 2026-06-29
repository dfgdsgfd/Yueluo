"use client";
import type {
  ProfileTabs,
  VideoCenterVisibilityConfig
} from "@/lib/types";
import {
  shouldShowVideoCenterForUser
} from "@/lib/video-center";

export function normalizeProfileTabs(tabs: ProfileTabs) {
  return {
    notes: tabs.notes ?? [],
    private: tabs.private ?? [],
    collections: tabs.collections ?? [],
    likes: tabs.likes ?? [],
  };
}


export function shouldShowVideoCenter(
  createdAt?: string | null,
  config?: VideoCenterVisibilityConfig | null,
) {
  return shouldShowVideoCenterForUser(createdAt, config);
}
