import { MobilePublishPage } from "@/components/publish/mobile-publish-page";
import { requirePageAccessToken } from "@/lib/server/auth-page-guard";

export const metadata = {
  title: "Create",
};

export default async function PublishMobilePage() {
  await requirePageAccessToken("/publish/mobile");

  return <MobilePublishPage />;
}
