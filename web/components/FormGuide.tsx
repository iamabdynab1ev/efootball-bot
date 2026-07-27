"use client";

interface Props {
  form: string[];
  /** Компакт для узких экранов: точки меньше, максимум max последних матчей. */
  compact?: boolean;
  max?: number;
}

const colors: Record<string, string> = {
  W: "bg-green-500",
  D: "bg-yellow-500",
  L: "bg-red-500",
};

export function FormGuide({ form, compact = false, max }: Props) {
  if (!form || form.length === 0) return null;
  const shown = max && form.length > max ? form.slice(-max) : form;
  return (
    <div className={compact ? "flex items-center justify-center gap-[3px]" : "flex items-center gap-0.5"}>
      {shown.map((r, i) => (
        <span
          key={i}
          className={`inline-block rounded-full ${compact ? "h-1.5 w-1.5" : "h-2.5 w-2.5"} ${colors[r] ?? "bg-zinc-600"}`}
          title={r}
        />
      ))}
    </div>
  );
}
