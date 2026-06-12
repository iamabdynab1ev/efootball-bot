"use client";

import type { LucideIcon } from "lucide-react";
import { m, useReducedMotion } from "framer-motion";

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  text?: string;
  action?: React.ReactNode;
}

export function EmptyState({ icon: Icon, title, text, action }: EmptyStateProps) {
  const reduced = useReducedMotion();
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-14 px-6 text-center">
      <m.div
        initial={reduced ? false : { opacity: 0, y: 8, scale: 0.9 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        transition={{ type: "spring", stiffness: 240, damping: 18 }}
        className="flex h-14 w-14 items-center justify-center rounded-full text-zinc-400"
        style={{
          background:
            "radial-gradient(circle at 35% 30%, hsl(240 6% 18%), hsl(240 8% 11%))",
          boxShadow: "inset 0 0 0 1px hsl(240 6% 22%), 0 0 24px rgb(200 241 53 / 0.06)",
        }}
      >
        <Icon size={22} />
      </m.div>
      <div className="space-y-1">
        <p className="text-sm font-semibold text-zinc-300">{title}</p>
        {text && <p className="text-xs text-zinc-500 max-w-xs leading-relaxed">{text}</p>}
      </div>
      {action}
    </div>
  );
}
