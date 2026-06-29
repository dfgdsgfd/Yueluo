"use client";
import {
  cn
} from "@/lib/utils";
import {
  SelectOption
} from "./types";
import {
  truthy
} from "./helpers";

export function BooleanSelect({ label, value, onChange, onLabel = "启用", offLabel = "停用" }: { label: string; value: unknown; onChange: (value: boolean) => void; onLabel?: string; offLabel?: string }) {
  return (
    <div className="grid gap-1.5">
      <span className="text-xs font-semibold text-[#666c78]">{label}</span>
      <ToggleSwitch value={truthy(value)} onChange={onChange} onLabel={onLabel} offLabel={offLabel} />
    </div>
  );
}


export function SelectField({ label, value, options, onChange, placeholder = "请选择" }: { label: string; value: unknown; options: SelectOption[]; onChange: (value: string) => void; placeholder?: string }) {
  return (
    <label className="grid gap-1.5">
      <span className="text-xs font-semibold text-[#666c78]">{label}</span>
      <ChoiceSelect value={String(value ?? "")} onChange={onChange} options={options} placeholder={placeholder} />
    </label>
  );
}


export function ToggleSwitch({
  value,
  onChange,
  disabled = false,
  onLabel = "开启",
  offLabel = "关闭",
}: {
  value: boolean;
  onChange: (value: boolean) => void;
  disabled?: boolean;
  onLabel?: string;
  offLabel?: string;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      role="switch"
      aria-checked={value}
      onClick={() => onChange(!value)}
      className={cn(
        "inline-flex h-10 w-full items-center justify-between rounded-lg border px-3 text-sm font-semibold transition focus:outline-none focus:ring-4 focus:ring-[#1d4ed8]/10",
        value ? "border-[#18a058]/25 bg-[#f0fff7] text-[#107a43]" : "border-black/[0.08] bg-[#fafbfe] text-[#687080]",
        disabled && "cursor-not-allowed opacity-60",
      )}
    >
      <span>{value ? onLabel : offLabel}</span>
      <span className={cn("flex h-5 w-9 items-center rounded-full p-0.5 transition", value ? "bg-[#18a058]" : "bg-[#c8ced8]")}>
        <span className={cn("size-4 rounded-full bg-white shadow-sm transition", value && "translate-x-4")} />
      </span>
    </button>
  );
}


export function ChoiceSelect({
  value,
  onChange,
  options,
  placeholder,
  disabled = false,
  required = false,
  className,
}: {
  value: string;
  onChange: (value: string) => void;
  options: SelectOption[];
  placeholder?: string;
  disabled?: boolean;
  required?: boolean;
  className?: string;
}) {
  const visibleOptions = selectOptionsWithCurrent(options, value);
  return (
    <select
      value={value}
      onChange={(event) => onChange(event.target.value)}
      disabled={disabled}
      required={required}
      className={cn("h-10 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8] disabled:bg-[#f1f2f5] disabled:text-[#8a8f9d]", className)}
    >
      {placeholder !== undefined ? <option value="">{placeholder}</option> : null}
      {visibleOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
    </select>
  );
}


export function selectOptionsWithCurrent(options: SelectOption[], value: unknown) {
  const current = String(value ?? "");
  if (!current || options.some((option) => option.value === current)) return options;
  return [{ value: current, label: `当前值：${current}` }, ...options];
}


export function castSelectValue(value: string, type?: "string" | "number" | "boolean") {
  if (type === "number") {
    const numeric = Number(value);
    return Number.isFinite(numeric) ? numeric : value;
  }
  if (type === "boolean") return value === "true";
  return value;
}


export function TextareaField({ label, value, onChange }: { label: string; value: unknown; onChange: (value: string) => void }) {
  return (
    <label className="grid gap-1.5">
      <span className="text-xs font-semibold text-[#666c78]">{label}</span>
      <textarea value={String(value ?? "")} onChange={(event) => onChange(event.target.value)} className="min-h-[110px] rounded-lg border border-black/[0.08] bg-[#fafbfe] p-3 text-sm outline-none focus:border-[#1d4ed8]" />
    </label>
  );
}
