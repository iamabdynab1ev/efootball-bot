import { cn } from "@/lib/utils";

export function Skeleton({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("skeleton rounded-md", className)}
      {...props}
    />
  );
}

export function SkeletonCard() {
  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-900 p-5 space-y-3">
      <Skeleton className="h-4 w-1/3" />
      <Skeleton className="h-8 w-1/2" />
      <Skeleton className="h-3 w-2/3" />
    </div>
  );
}

export function SkeletonRow() {
  return (
    <div className="flex items-center gap-3 px-4 py-3">
      <Skeleton className="h-8 w-8 rounded-full flex-shrink-0" />
      <div className="flex-1 space-y-2">
        <Skeleton className="h-3.5 w-2/5" />
        <Skeleton className="h-3 w-1/4" />
      </div>
      <Skeleton className="h-6 w-12" />
    </div>
  );
}

export function SkeletonTable({ rows = 5 }: { rows?: number }) {
  return (
    <div className="space-y-0">
      {Array.from({ length: rows }).map((_, i) => (
        <SkeletonRow key={i} />
      ))}
    </div>
  );
}

export function SkeletonProfile() {
  return (
    <div className="space-y-4">
      {/* Profile card */}
      <div className="rounded-xl border border-zinc-800 overflow-hidden">
        <div className="h-24 bg-zinc-800 animate-pulse" />
        <div className="grid grid-cols-3 divide-x divide-zinc-800 border-t border-zinc-800">
          {[0,1,2].map(i => (
            <div key={i} className="py-3 flex flex-col items-center gap-1">
              <Skeleton className="h-5 w-8" />
              <Skeleton className="h-3 w-12" />
            </div>
          ))}
        </div>
      </div>
      {/* Form */}
      <div className="rounded-xl border border-zinc-800 bg-zinc-900 p-6 space-y-4">
        <Skeleton className="h-4 w-40" />
        <Skeleton className="h-9 w-full" />
        <Skeleton className="h-9 w-full" />
        <Skeleton className="h-9 w-full" />
        <Skeleton className="h-9 w-32" />
      </div>
    </div>
  );
}

export function SkeletonMatchCard() {
  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
      <div className="flex items-center justify-between px-4 py-2.5 border-b border-zinc-800">
        <Skeleton className="h-3.5 w-16" />
        <Skeleton className="h-5 w-20" />
      </div>
      <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-4 px-5 py-5">
        <div className="flex flex-col items-center gap-2">
          <Skeleton className="h-11 w-11 rounded-full" />
          <Skeleton className="h-3.5 w-20" />
        </div>
        <div className="flex items-center gap-2">
          <Skeleton className="h-8 w-8" />
          <Skeleton className="h-5 w-4" />
          <Skeleton className="h-8 w-8" />
        </div>
        <div className="flex flex-col items-center gap-2">
          <Skeleton className="h-11 w-11 rounded-full" />
          <Skeleton className="h-3.5 w-20" />
        </div>
      </div>
    </div>
  );
}

export function SkeletonBracket() {
  return (
    <div className="flex gap-8 overflow-hidden p-2" aria-hidden="true">
      {[4, 2, 1].map((n, col) => (
        <div key={col} className="flex flex-col justify-around gap-3" style={{ minHeight: 280 }}>
          {Array.from({ length: n }).map((_, i) => (
            <Skeleton key={i} className="h-[60px] w-[160px] rounded-xl" />
          ))}
        </div>
      ))}
    </div>
  );
}
