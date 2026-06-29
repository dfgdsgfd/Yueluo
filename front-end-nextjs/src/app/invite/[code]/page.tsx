import { MobileInviteLandingPage } from "@/components/invite/mobile-invite-landing-page";

export const metadata = {
  title: "Invitation",
};

export default async function InviteLandingRoute({
  params,
}: {
  params: Promise<{ code: string }>;
}) {
  const { code } = await params;
  return <MobileInviteLandingPage code={code} />;
}
