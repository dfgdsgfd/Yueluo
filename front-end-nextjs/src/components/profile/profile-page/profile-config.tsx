"use client";
import dynamic from "next/dynamic";
import type {
  Area,
  Point
} from "react-easy-crop";
import type {
  ProfileCropperProps
} from "@/components/profile/profile-cropper";
import {
  UserRound
} from "lucide-react";
import {
  CropperSkeleton
} from "./profile-edit";

export const Cropper = dynamic<ProfileCropperProps>(
  () => import("@/components/profile/profile-cropper").then((mod) => mod.ProfileCropper),
  {
    loading: () => <CropperSkeleton />,
    ssr: false,
  },
);


export type ProfileTabKey = "notes" | "private" | "collections" | "likes";

export type ProfileImageKind = "avatar" | "background";


export type PendingImageCrop = {
  kind: ProfileImageKind;
  name: string;
  src: string;
};


export type ImageCropState = {
  areaPixels: Area | null;
  position: Point;
  zoom: number;
};


export const tabKeys = ["notes", "private", "collections", "likes"] as const;

export const publicProfileTabKeys = ["notes", "collections", "likes"] as const;

export const mobileQuickActions = [
  { key: "creatorCenter", icon: UserRound, href: "/creator-center" },
] as const;


export function createInitialImageCrop(): ImageCropState {
  return {
    areaPixels: null,
    position: { x: 0, y: 0 },
    zoom: 1,
  };
}
