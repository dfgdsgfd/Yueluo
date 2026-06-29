import { ViewerProfileDataPage } from "@/components/profile/profile-data-page";
import { getViewerProfileData } from "@/lib/api";
import { requirePageAccessToken } from "@/lib/server/auth-page-guard";

export const metadata = {
  title: "Profile",
};

export default async function ViewerProfilePage() {
  const context = await requirePageAccessToken("/profile");
  const initialPayload = await getViewerProfileData(
    context,
  ).catch(() => null);

  return <ViewerProfileDataPage initialPayload={initialPayload} />;
}
