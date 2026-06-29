import { ProfileHistoryDataPage } from "@/components/profile/profile-history-page";
import { requirePageAccessToken } from "@/lib/server/auth-page-guard";

export const metadata = {
  title: "History",
};

export default async function ProfileHistoryPage() {
  await requirePageAccessToken("/profile/history");

  return <ProfileHistoryDataPage />;
}
