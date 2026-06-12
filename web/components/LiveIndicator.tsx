"use client";

import { cn } from "@/lib/utils";

/**
 * Пульсирующий LIVE-индикатор: показывает, что страница получает
 * обновления результатов в реальном времени по SSE.
 */
export function LiveIndicator({ live, className }: { live: boolean; className?: string }) {
  if (!live) return null;
  return (
    <span
      role="status"
      aria-live="polite"
      aria-label="LIVE"
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border border-cyan-400/30 bg-cyan-400/10 px-2 py-0.5",
        className,
      )}
    >
      <span className="relative flex h-1.5 w-1.5">
        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-cyan-400 opacity-60 motion-reduce:animate-none" />
        <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-cyan-400" />
      </span>
      <span className="text-[10px] font-bold tracking-widest text-cyan-300">LIVE</span>
    </span>
  );
}
