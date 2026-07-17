import * as React from "react";
import { cn } from "@/lib/utils";

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {}

const Input = React.forwardRef<HTMLInputElement, InputProps>(({ className, ...props }, ref) => (
  <input
    ref={ref}
    className={cn(
      "flex h-9 w-full rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-1 text-sm text-zinc-100 placeholder:text-zinc-500",
      // Фокус: вольтовая рамка + мягкое свечение — «живой» инпут без тяжёлых эффектов.
      "transition-[border-color,box-shadow] duration-150 focus:outline-none focus:border-yellow-400/70",
      "focus:shadow-[0_0_0_3px_var(--volt-glow-soft),0_0_14px_-4px_var(--volt-glow-soft)]",
      "disabled:cursor-not-allowed disabled:opacity-50",
      className
    )}
    {...props}
  />
));
Input.displayName = "Input";

export { Input };
