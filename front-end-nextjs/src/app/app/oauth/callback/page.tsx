import { getTranslations } from "next-intl/server";
import { NativeOAuthFallback } from "@/components/auth/native-oauth-fallback";
import { buildNativeOAuthDeepLinks } from "@/lib/native-oauth-links";

type CallbackSearchParams = Promise<
  Record<string, string | string[] | undefined>
>;

export const metadata = {
  title: "Yuem",
  robots: { index: false, follow: false },
};

export default async function NativeOAuthCallbackPage({
  searchParams,
}: {
  searchParams: CallbackSearchParams;
}) {
  const t = await getTranslations("login");
  const rawParams = await searchParams;
  const code = firstValue(rawParams.code);
  const appState = firstValue(rawParams.app_state);
  const error = firstValue(rawParams.error);
  const valid = Boolean(appState && (code || error));
  const { schemeUrl, intentUrl } = buildNativeOAuthDeepLinks({
    appState,
    code,
    error,
  });

  return (
    <main className="flex min-h-dvh items-center justify-center bg-background px-5 py-[max(1.25rem,env(safe-area-inset-top))] text-foreground">
      <section className="w-full max-w-sm rounded-3xl border border-border bg-card p-7 text-center shadow-xl">
        <div className="mx-auto grid size-16 place-items-center rounded-2xl bg-primary text-2xl font-bold text-primary-foreground">
          Y
        </div>
        <h1 className="mt-6 text-2xl font-semibold">{t("nativeCallbackTitle")}</h1>
        <p className="mt-3 text-sm leading-6 text-muted-foreground">
          {valid ? t("nativeCallbackDescription") : t("nativeCallbackInvalid")}
        </p>
        {valid ? (
          <NativeOAuthFallback
            deepLink={schemeUrl}
            intentLink={intentUrl}
            openLabel={t("nativeCallbackOpen")}
          />
        ) : null}
      </section>
    </main>
  );
}

function firstValue(value: string | string[] | undefined) {
  return Array.isArray(value) ? value[0] ?? "" : value ?? "";
}
