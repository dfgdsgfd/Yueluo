import { LogIn } from "lucide-react";
import { getTranslations } from "next-intl/server";
import { LoginForm } from "@/components/auth/login-form";
import { LanguageSwitcher } from "@/components/language-switcher";

export async function generateMetadata() {
  const t = await getTranslations("loginUnlock");
  return {
    title: t("title"),
  };
}

export default async function LoginPage() {
  const t = await getTranslations("loginUnlock");

  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-6">
      <div className="fixed right-4 top-4">
        <LanguageSwitcher tone="light" />
      </div>
      <section className="w-full max-w-sm rounded-2xl border border-border bg-card p-8 text-center shadow-sm">
        <div className="mx-auto mb-6 flex size-12 items-center justify-center rounded-full bg-primary text-base font-bold text-primary-foreground">
          <LogIn className="size-5" aria-hidden="true" />
        </div>
        <h1 className="text-2xl font-semibold text-foreground">{t("title")}</h1>
        <p className="mt-3 text-sm leading-6 text-muted-foreground">
          {t("description")}
        </p>
        <LoginForm />
      </section>
    </main>
  );
}
