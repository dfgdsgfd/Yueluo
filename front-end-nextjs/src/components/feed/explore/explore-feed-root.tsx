"use client";

import type { InitialFeedData } from "@/lib/types/content";
import { ExploreFeedView } from "./explore-feed-view";
import { useExploreFeedController } from "./use-explore-feed-controller";

export function ExploreFeed({ initialData }: { initialData: InitialFeedData }) {
  const controller = useExploreFeedController({ initialData });
  return <ExploreFeedView controller={controller} />;
}
