import { MobileDraftsPage } from "@/components/publish/mobile-drafts-page";
import { requirePageAccessToken } from "@/lib/server/auth-page-guard";

export const metadata = {
  title: "Drafts",
};

export default async function PublishMobileDraftsPage() {
  await requirePageAccessToken("/publish/mobile/drafts");

  return <MobileDraftsPage />;
}
