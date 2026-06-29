import { headers } from "next/headers";
import { MobileCreatorCenterPage } from "@/components/creator-center/mobile-creator-center-page";
import { apiRequestContextFromHeaders } from "@/lib/api";
import { getCreatorCenterInitialData } from "@/lib/server/creator-center-page-data";

export const metadata = {
  title: "Creator Center",
};

export default async function CreatorCenterPage() {
  const headerStore = await headers();
  const initialData = await getCreatorCenterInitialData(
    apiRequestContextFromHeaders(headerStore),
  ).catch(() => null);

  return <MobileCreatorCenterPage initialData={initialData} />;
}
