"use client";

import { useUIStore } from "@/lib/store";
import { cn } from "@/lib/utils";

export function SidebarOffset({ children }: { children: React.ReactNode }) {
  const { sidebarCollapsed } = useUIStore();

  return (
    <div
      className={cn(
        "transition-[margin] duration-200 ease-in-out",
        "mt-14 lg:mt-0 pb-20 lg:pb-0",
        sidebarCollapsed ? "lg:ml-16" : "lg:ml-[220px]"
      )}
    >
      {children}
    </div>
  );
}
