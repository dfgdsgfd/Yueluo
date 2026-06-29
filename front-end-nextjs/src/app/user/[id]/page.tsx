import { UserProfileDataPage } from "@/components/profile/profile-data-page";
import { getUserProfileData } from "@/lib/api";
import { requirePageAccessToken } from "@/lib/server/auth-page-guard";

export const metadata = {
  title: "Creator Profile",
};

export default async function UserProfilePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const context = await requirePageAccessToken(`/user/${encodeURIComponent(id)}`);
  const initialPayload = await getUserProfileData(
    id,
    context,
  ).catch(() => null);

  return <UserProfileDataPage userId={id} initialPayload={initialPayload} />;
}
