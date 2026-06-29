import { ServiceModeEntry } from "@/components/maintenance/service-mode-entry";

export default async function ServiceModePage({
  params,
}: {
  params: Promise<{ code: string }>;
}) {
  const { code } = await params;
  return <ServiceModeEntry code={code} />;
}
