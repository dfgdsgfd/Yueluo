import { PublishWorkbench } from "@/components/publish/publish-workbench";
import { requirePageAccessToken } from "@/lib/server/auth-page-guard";

export const metadata = {
  title: "Publish",
};

export default async function PublishPage() {
  await requirePageAccessToken("/publish");

  return <PublishWorkbench />;
}
