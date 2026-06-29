"use client";

import Cropper from "react-easy-crop";
import type { Area, Point } from "react-easy-crop";

export type ProfileCropperProps = {
  aspect: number;
  classes: { cropAreaClassName: string };
  crop: Point;
  cropShape: "rect";
  image: string;
  maxZoom: number;
  minZoom: number;
  objectFit: "contain";
  onCropChange: (position: Point) => void;
  onCropComplete: (croppedArea: Area, croppedAreaPixels: Area) => void;
  onZoomChange: (zoom: number) => void;
  roundCropAreaPixels: boolean;
  showGrid: boolean;
  zoom: number;
};

export function ProfileCropper(props: ProfileCropperProps) {
  return <Cropper {...props} />;
}
