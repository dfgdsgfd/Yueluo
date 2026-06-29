"use client";


import { PublishWorkbenchView } from "./publish-workbench-view";
import { usePublishWorkbenchController } from "./use-publish-workbench-controller";

export { prefetchPublishWorkbenchData } from "./publish-workbench-bootstrap";

export function PublishWorkbench() {
  const controller = usePublishWorkbenchController();
  return <PublishWorkbenchView controller={controller} />;
}
