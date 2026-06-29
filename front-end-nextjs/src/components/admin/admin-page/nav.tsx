"use client";
import {
  Activity,
  AlertCircle,
  Bell,
  CircleDollarSign,
  ClipboardList,
  Database,
  FileSearch,
  FileText,
  Folder,
  Gauge,
  Heart,
  Image as ImageIcon,
  KeyRound,
  LayoutDashboard,
  Megaphone,
  MessageSquare,
  Package,
  Radio,
  ShieldAlert,
  ShieldCheck,
  ShieldX,
  SlidersHorizontal,
  Star,
  Settings,
  Sparkles,
  Tags,
  Trash2,
  UserCog,
  Users,
  Wallet,
  Wrench,
  Zap
} from "lucide-react";
import type {
  AdminListResource
} from "@/lib/types";
import type {
  AdminSection,
  NavItem
} from "./types";

export function createNavSections(
  labels: {
    item: (id: string) => string;
    resource: (id: AdminListResource) => string;
    section: (id: string) => string;
  },
  backendApiEntryPath: string,
): AdminSection[] {
  function navResource(resource: AdminListResource): NavItem {
    return {
      id: resource,
      label: labels.resource(resource),
      icon: resourceIcons[resource] ?? Database,
      view: { kind: "resource", resource },
    };
  }

  return [
  {
    id: "overview",
    label: labels.section("overview"),
    items: [
      { id: "dashboard", label: labels.item("dashboard"), icon: LayoutDashboard, view: { kind: "dashboard" } },
      { id: "queues", label: labels.item("queues"), icon: Gauge, view: { kind: "queues" } },
      { id: "logs", label: labels.item("logs"), icon: ClipboardList, view: { kind: "logs" } },
      { id: "observability", label: labels.item("observability"), icon: Activity, view: { kind: "observability" } },
      { id: "database", label: labels.item("database"), icon: Database, view: { kind: "database" } },
      { id: "maintenance", label: labels.item("maintenance"), icon: ShieldAlert, view: { kind: "maintenance" } },
      { id: "operations", label: labels.item("operations"), icon: Activity, view: { kind: "operations" } },
    ],
  },
  {
    id: "users",
    label: labels.section("users"),
    items: [
      navResource("users"),
      navResource("admins"),
      navResource("sessions"),
      navResource("audit"),
    ],
  },
  {
    id: "content",
    label: labels.section("content"),
    items: [
      navResource("posts"),
      navResource("comments"),
      navResource("categories"),
      navResource("tags"),
      navResource("media-library"),
      navResource("file-recycle-bin"),
      navResource("posts-quality"),
      navResource("collections"),
      navResource("likes"),
      navResource("follows"),
    ],
  },
  {
    id: "risk",
    label: labels.section("risk"),
    items: [
      navResource("content-review"),
      navResource("ai-moderation-logs"),
      navResource("reports"),
      { id: "access-block", label: labels.item("accessBlock"), icon: ShieldX, view: { kind: "access-block" } },
      navResource("banned-word-categories"),
      navResource("banned-words"),
    ],
  },
  {
    id: "creator",
    label: labels.section("creator"),
    items: [
      { id: "withdraw", label: labels.item("withdraw"), icon: Wallet, view: { kind: "withdraw" } },
      navResource("quality-reward-settings"),
    ],
  },
  {
    id: "growth",
    label: labels.section("growth"),
    items: [
      navResource("post-configs"),
      navResource("user-configs"),
      navResource("user-toolbar"),
      { id: "invite", label: labels.item("invite"), icon: Megaphone, view: { kind: "invite" } },
      { id: "coupon", label: labels.item("coupon"), icon: Tags, view: { kind: "coupon" } },
      { id: "points", label: labels.item("points"), icon: CircleDollarSign, view: { kind: "points" } },
      { id: "ai-agent", label: labels.item("aiAgent"), icon: Sparkles, view: { kind: "ai-agent" } },
      { id: "onboarding-settings", label: labels.item("onboardingSettings"), icon: Sparkles, view: { kind: "onboarding-settings" } },
    ],
  },
  {
    id: "notice",
    label: labels.section("notice"),
    items: [
      navResource("announcements"),
      navResource("system-notifications"),
      navResource("notification-templates"),
      navResource("feedback"),
    ],
  },
  {
    id: "system",
    label: labels.section("system"),
    items: [
      navResource("app-versions"),
      { id: "backend-api-docs", label: labels.item("backendApi"), icon: FileText, href: backendApiEntryPath },
      navResource("open-apis"),
      navResource("licenses"),
      { id: "system-update", label: labels.item("systemUpdate"), icon: Package, view: { kind: "system-update" } },
      { id: "component-check", label: labels.item("componentCheck"), icon: Wrench, view: { kind: "component-check" } },
      { id: "hidden-watermark-access", label: labels.item("hiddenWatermarkAccess"), icon: FileSearch, view: { kind: "hidden-watermark-access" } },
      { id: "settings", label: labels.item("settings"), icon: Settings, view: { kind: "settings" } },
    ],
  },
  ];
}

const resourceIcons: Partial<Record<AdminListResource, typeof Database>> = {
  users: Users,
  posts: FileText,
  "ai-moderation-logs": ShieldCheck,
  "content-review": ShieldCheck,
  reports: AlertCircle,
  audit: Star,
  comments: MessageSquare,
  categories: Folder,
  tags: Tags,
  "media-library": ImageIcon,
  "file-recycle-bin": Trash2,
  announcements: Megaphone,
  "system-notifications": Bell,
  "notification-templates": ClipboardList,
  feedback: MessageSquare,
  "open-apis": KeyRound,
  admins: UserCog,
  sessions: Radio,
  "banned-word-categories": ShieldCheck,
  "banned-words": ShieldCheck,
  collections: Folder,
  follows: Users,
  likes: Heart,
  licenses: KeyRound,
  "app-versions": Package,
  "post-configs": Zap,
  "user-configs": SlidersHorizontal,
  "posts-quality": Star,
  "quality-reward-settings": Star,
  "user-toolbar": Settings,
};
