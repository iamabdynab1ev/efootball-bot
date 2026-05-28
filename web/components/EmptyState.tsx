import type { LucideIcon } from "lucide-react";

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  text?: string;
  action?: React.ReactNode;
}

export function EmptyState({ icon: Icon, title, text, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-14 px-6 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-zinc-800/80 text-zinc-500">
        <Icon size={20} />
      </div>
      <div className="space-y-1">
        <p className="text-sm font-semibold text-zinc-300">{title}</p>
        {text && <p className="text-xs text-zinc-500 max-w-xs leading-relaxed">{text}</p>}
      </div>
      {action}
    </div>
  );
}
