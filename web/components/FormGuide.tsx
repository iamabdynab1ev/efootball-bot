"use client";

interface Props {
  form: string[];
}

const colors: Record<string, string> = {
  W: "bg-green-500",
  D: "bg-yellow-500",
  L: "bg-red-500",
};

export function FormGuide({ form }: Props) {
  if (!form || form.length === 0) return null;
  return (
    <div className="flex items-center gap-0.5">
      {form.map((r, i) => (
        <span
          key={i}
          className={`inline-block h-2.5 w-2.5 rounded-full ${colors[r] ?? "bg-zinc-600"}`}
          title={r}
        />
      ))}
    </div>
  );
}
