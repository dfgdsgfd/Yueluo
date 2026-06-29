import type { FeedPost, InitialFeedData } from "./types";
import { normalizeSiteProfile } from "./seo";

const baseFixturePosts: FeedPost[] = [
  {
    id: 101,
    title: "A compact weekly plan that finally stayed realistic",
    content:
      "A quiet review flow for errands, workouts, meals, and focus blocks.",
    category_id: 4,
    category: "Work",
    type: 1,
    like_count: 238,
    collect_count: 42,
    comment_count: 18,
    created_at: "2026-06-01T08:30:00.000Z",
    author: "Mira Studio",
    nickname: "Mira Studio",
    avatar:
      "https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=160&h=160&fit=crop&crop=faces",
    liked: true,
    collected: false,
    images: [
      {
        url: "https://images.unsplash.com/photo-1497215728101-856f4ea42174?w=900&auto=format&fit=crop",
        width: 900,
        height: 1180,
        isFreePreview: true,
      },
    ],
  },
  {
    id: 102,
    title: "Market list for a clean five-day meal rhythm",
    content: "Batch prep notes, fresh produce, and simple repeatable dinners.",
    category_id: 2,
    category: "Food",
    type: 1,
    like_count: 459,
    collect_count: 80,
    comment_count: 27,
    created_at: "2026-06-02T10:10:00.000Z",
    author: "Sunday Table",
    nickname: "Sunday Table",
    avatar:
      "https://images.unsplash.com/photo-1502685104226-ee32379fefbe?w=160&h=160&fit=crop&crop=faces",
    liked: false,
    collected: true,
    images: [
      {
        url: "https://images.unsplash.com/photo-1512621776951-a57141f2eefd?w=900&auto=format&fit=crop",
        width: 900,
        height: 720,
        isFreePreview: true,
      },
    ],
  },
  {
    id: 103,
    title: "Desk reset before the next planning sprint",
    content: "A low-friction layout for notes, inbox review, and deep work.",
    category_id: 4,
    category: "Work",
    type: 2,
    like_count: 312,
    collect_count: 51,
    comment_count: 12,
    created_at: "2026-06-03T09:20:00.000Z",
    author: "Northline",
    nickname: "Northline",
    avatar:
      "https://images.unsplash.com/photo-1517841905240-472988babdf9?w=160&h=160&fit=crop&crop=faces",
    liked: false,
    collected: false,
    images: [
      {
        url: "https://images.unsplash.com/photo-1524758631624-e2822e304c36?w=900&auto=format&fit=crop",
        width: 900,
        height: 1120,
        isFreePreview: true,
      },
    ],
    image:
      "https://images.unsplash.com/photo-1524758631624-e2822e304c36?w=900&auto=format&fit=crop",
    preview_video_url:
      "https://storage.googleapis.com/shaka-demo-assets/angel-one-hls/hls.m3u8",
  },
  {
    id: 104,
    title: "Small apartment zones that make routines visible",
    content: "A visual home map for reading, charging, laundry, and resets.",
    category_id: 5,
    category: "Home",
    type: 1,
    like_count: 188,
    collect_count: 33,
    comment_count: 8,
    created_at: "2026-06-04T11:00:00.000Z",
    author: "Room Notes",
    nickname: "Room Notes",
    avatar:
      "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=160&h=160&fit=crop&crop=faces",
    liked: true,
    collected: true,
    images: [
      {
        url: "https://images.unsplash.com/photo-1500530855697-b586d89ba3ee?w=900&auto=format&fit=crop",
        width: 900,
        height: 920,
        isFreePreview: true,
      },
    ],
  },
  {
    id: 105,
    title: "A city walk plan for one open afternoon",
    content: "Bookmarks, light packing, and a gentle route between stops.",
    category_id: 6,
    category: "Travel",
    type: 1,
    like_count: 96,
    collect_count: 19,
    comment_count: 4,
    created_at: "2026-06-05T12:45:00.000Z",
    author: "Field Map",
    nickname: "Field Map",
    avatar:
      "https://images.unsplash.com/photo-1524504388940-b1c1722653e1?w=160&h=160&fit=crop&crop=faces",
    liked: false,
    collected: false,
    images: [
      {
        url: "https://images.unsplash.com/photo-1518005020951-eccb494ad742?w=900&auto=format&fit=crop",
        width: 900,
        height: 1260,
        isFreePreview: true,
      },
    ],
  },
  {
    id: 106,
    title: "The tiny checklist that makes publishing less noisy",
    content: "Capture, trim, title, tag, preview, and ship without drama.",
    category_id: 1,
    category: "Outfit",
    type: 1,
    like_count: 533,
    collect_count: 105,
    comment_count: 31,
    created_at: "2026-06-06T14:05:00.000Z",
    author: "Daily Kit",
    nickname: "Daily Kit",
    avatar:
      "https://images.unsplash.com/photo-1508214751196-bcfd4ca60f91?w=160&h=160&fit=crop&crop=faces",
    liked: false,
    collected: true,
    images: [
      {
        url: "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=900&auto=format&fit=crop",
        width: 900,
        height: 1250,
        isFreePreview: true,
      },
    ],
  },
  {
    id: 107,
    title: "Soft morning makeup with a low-effort finish",
    content: "A short routine with compact tools and soft daylight checks.",
    category_id: 3,
    category: "Beauty",
    type: 1,
    like_count: 782,
    collect_count: 133,
    comment_count: 41,
    created_at: "2026-06-07T08:15:00.000Z",
    author: "Glow Notes",
    nickname: "Glow Notes",
    avatar:
      "https://images.unsplash.com/photo-1529626455594-4ff0802cfb7e?w=160&h=160&fit=crop&crop=faces",
    liked: false,
    collected: false,
    images: [
      {
        url: "https://images.unsplash.com/photo-1522335789203-aabd1fc54bc9?w=900&auto=format&fit=crop",
        width: 900,
        height: 980,
        isFreePreview: true,
      },
    ],
  },
  {
    id: 108,
    title: "Evening stretch notes after a long commute",
    content: "A simple cooldown sequence and a few reminders for recovery.",
    category_id: 7,
    category: "Fitness",
    type: 1,
    like_count: 149,
    collect_count: 28,
    comment_count: 9,
    created_at: "2026-06-07T11:35:00.000Z",
    author: "Motion Log",
    nickname: "Motion Log",
    avatar:
      "https://images.unsplash.com/photo-1508214751196-bcfd4ca60f91?w=160&h=160&fit=crop&crop=faces",
    liked: true,
    collected: false,
    images: [
      {
        url: "https://images.unsplash.com/photo-1518611012118-696072aa579a?w=900&auto=format&fit=crop",
        width: 900,
        height: 760,
        isFreePreview: true,
      },
    ],
  },
];

export const fixturePosts: FeedPost[] = Array.from({ length: 3 }, (_, group) =>
  baseFixturePosts.map((post, index) => ({
    ...post,
    id: `${post.id}-${group}`,
    like_count: post.like_count + group * 73 + index * 9,
    liked: group === 0 ? post.liked : (group + index) % 4 === 0,
    collected: group === 0 ? post.collected : (group + index) % 5 === 0,
  })),
).flat();

export const fixtureInitialFeedData: InitialFeedData = {
  posts: fixturePosts,
  categories: [],
  pagination: {
    page: 1,
    limit: fixturePosts.length,
    total: fixturePosts.length,
    pages: 1,
    hasNextPage: false,
    cursor: null,
  },
  source: "fixture",
  toolbarItems: [],
  siteProfile: normalizeSiteProfile(),
  backendNotice:
    "The backend API was not available, so typed fixture data is rendering this feed.",
};
