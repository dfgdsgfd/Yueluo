"use client";

import type { ResourceConfig } from "./types";
import { engagementResourceConfigs } from "./resource-config-engagement";
import { notificationResourceConfigs } from "./resource-config-notifications";
import { primaryResourceConfigs } from "./resource-config-primary";
import { recommendationResourceConfigs } from "./resource-config-recommendation";
import { securityResourceConfigs } from "./resource-config-security";
import { taxonomyMediaResourceConfigs } from "./resource-config-taxonomy-media";

export const resourceConfigs: ResourceConfig[] = [
  ...primaryResourceConfigs,
  ...taxonomyMediaResourceConfigs,
  ...notificationResourceConfigs,
  ...securityResourceConfigs,
  ...engagementResourceConfigs,
  ...recommendationResourceConfigs,
];

export const resourceConfigMap = new Map(resourceConfigs.map((config) => [config.resource, config]));
