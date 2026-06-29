import { headers } from "next/headers";
import { WalletPage } from "@/components/wallet/wallet-page";
import { apiRequestContextFromHeaders } from "@/lib/api";
import { getWalletInitialData } from "@/lib/server/wallet-page-data";

export const metadata = {
  title: "Wallet",
};

export default async function WalletRoute() {
  const headerStore = await headers();
  const initialData = await getWalletInitialData(
    apiRequestContextFromHeaders(headerStore),
  ).catch(() => null);

  return <WalletPage initialData={initialData} />;
}
