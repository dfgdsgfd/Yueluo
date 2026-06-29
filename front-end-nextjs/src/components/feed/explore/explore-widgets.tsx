"use client";
import {
  type FormEvent
} from "react";
import {
  type LucideIcon,
  Search
} from "lucide-react";
import {
  Button
} from "@/components/ui/button";
import type {
  Category
} from "@/lib/types";
export function ProfileTabSkeleton() {
  return (
    <div className="theme-adaptive min-h-dvh bg-[#121212] px-4 pb-20 pt-[72px] text-[#e0e0e0]" aria-busy="true">
      <div className="mx-auto max-w-[430px] overflow-hidden rounded-2xl border border-white/[0.08] bg-[#181818]">
        <div className="h-32 animate-pulse bg-white/[0.08]" />
        <div className="px-5 pb-6">
          <div className="-mt-10 size-20 animate-pulse rounded-full border-[3px] border-[#181818] bg-white/[0.12]" />
          <div className="mt-4 h-5 w-28 animate-pulse rounded bg-white/[0.12]" />
          <div className="mt-3 h-4 w-44 animate-pulse rounded bg-white/[0.08]" />
          <div className="mt-5 grid grid-cols-3 gap-2">
            {[0, 1, 2].map((item) => (
              <div key={item} className="h-16 animate-pulse rounded-xl bg-white/[0.06]" />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}


export function SearchForm({
  active,
  clearLabel,
  disabled,
  onChange,
  onClear,
  onSubmit,
  placeholder,
  submitLabel,
  value,
}: {
  active: boolean;
  clearLabel: string;
  disabled?: boolean;
  onChange: (value: string) => void;
  onClear: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  placeholder: string;
  submitLabel: string;
  value: string;
}) {
  return (
    <form
      onSubmit={onSubmit}
      suppressHydrationWarning
      className="flex h-11 items-center gap-2 rounded-full bg-[var(--explore-control)] px-3"
    >
      <Search className="size-4 shrink-0 text-[var(--explore-subtle)]" />
      <input
        type="search"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        aria-label={submitLabel}
        autoComplete="off"
        enterKeyHint="search"
        placeholder={placeholder}
        suppressHydrationWarning
        className="min-w-0 flex-1 bg-transparent text-sm text-[var(--explore-strong)] outline-none placeholder:text-[var(--explore-subtle)]"
      />
      {active ? (
        <button
          type="button"
          onClick={onClear}
          className="h-8 rounded-full px-2 text-xs font-semibold text-[var(--explore-muted)] hover:bg-[var(--explore-control-hover)] hover:text-[var(--explore-strong)]"
        >
          {clearLabel}
        </button>
      ) : null}
      <Button
        type="submit"
        size="icon"
        aria-label={submitLabel}
        className="size-8 shrink-0"
        disabled={disabled}
      >
        <Search className="size-4" />
      </Button>
    </form>
  );
}


export function NavIconWithBadge({
  Icon,
  className,
  count = 0,
  strokeWidth,
}: {
  Icon: LucideIcon;
  className?: string;
  count?: number;
  strokeWidth?: number;
}) {
  return (
    <span className="relative inline-flex items-center justify-center">
      <Icon className={className} strokeWidth={strokeWidth} />
      {count > 0 ? (
        <span className="absolute -right-2 -top-2 flex min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-black leading-4 text-white ring-2 ring-[var(--explore-badge-ring)]">
          {count > 99 ? "99+" : count}
        </span>
      ) : null}
    </span>
  );
}


export type KnownCategoryNameKey =
  | "beauty"
  | "fashion"
  | "food"
  | "gaming"
  | "interest"
  | "knowledge"
  | "lifestyle"
  | "music"
  | "novel"
  | "photography"
  | "pictureSet"
  | "ropeBondage"
  | "shackles"
  | "sports"
  | "technology"
  | "travel"
  | "video";

export function normalizeCategories(
  categories: Category[],
  allLabel: string,
  locale: string,
  knownNames: Record<KnownCategoryNameKey, string>,
) {
  const result: Category[] = [
    { id: -1, name: allLabel },
    ...categories.map((category) => ({
      ...category,
      name: localizedCategoryName(category, locale, knownNames),
    })),
  ];
  const seen = new Set<string>();

  return result.filter((category) => {
    const key =
      category.name.trim().toLowerCase() === allLabel.trim().toLowerCase()
        ? "all"
        : String(category.id);

    if (seen.has(key)) {
      return false;
    }

    seen.add(key);
    return true;
  });
}

function localizedCategoryName(
  category: Category,
  locale: string,
  knownNames: Record<KnownCategoryNameKey, string>,
) {
  const translations = category.translations;
  const localized = translations?.[locale as keyof NonNullable<Category["translations"]>]?.trim();
  const english = translations?.en?.trim();
  const knownKey = knownCategoryNameKey(
    category.name,
    category.category_title,
    english,
    category.display_name,
  );
  const localizedIsFallback = Boolean(
    localized &&
      locale !== "en" &&
      [category.name, category.category_title, english]
        .filter((value): value is string => Boolean(value?.trim()))
        .some((value) => value.trim().toLocaleLowerCase() === localized.toLocaleLowerCase()),
  );

  if (localized && !localizedIsFallback) {
    return localized;
  }
  if (knownKey) {
    return knownNames[knownKey];
  }
  return localized || category.display_name?.trim() || category.category_title?.trim() || category.name;
}

function knownCategoryNameKey(...values: Array<string | null | undefined>): KnownCategoryNameKey | null {
  for (const value of values) {
    const normalized = value?.trim().toLocaleLowerCase().replace(/[\s_-]+/g, "");
    switch (normalized) {
      case "beauty":
      case "beauties":
        return "beauty";
      case "fashion":
        return "fashion";
      case "food":
      case "foods":
        return "food";
      case "game":
      case "games":
      case "gaming":
        return "gaming";
      case "interest":
      case "interests":
      case "hobby":
      case "hobbies":
        return "interest";
      case "knowledge":
        return "knowledge";
      case "life":
      case "lifestyle":
        return "lifestyle";
      case "music":
        return "music";
      case "novel":
      case "novels":
        return "novel";
      case "photo":
      case "photos":
      case "photography":
        return "photography";
      case "album":
      case "photoalbum":
      case "pictureset":
      case "setofpictures":
        return "pictureSet";
      case "ropebondage":
        return "ropeBondage";
      case "shackle":
      case "shackles":
        return "shackles";
      case "sport":
      case "sports":
        return "sports";
      case "tech":
      case "technology":
        return "technology";
      case "travel":
      case "travels":
        return "travel";
      case "video":
      case "videos":
        return "video";
    }
  }
  return null;
}
