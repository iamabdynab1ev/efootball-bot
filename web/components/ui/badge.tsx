import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center gap-1.5 rounded-md px-2 py-0.5 text-[11px] font-700 uppercase tracking-wide transition-colors select-none border",
  {
    variants: {
      variant: {
        default:  "bg-zinc-800 text-zinc-300 border-zinc-700",
        yellow:   "bg-yellow-500/15 text-yellow-400 border-yellow-500/25",
        green:    "bg-green-500/15 text-green-400 border-green-500/25",
        blue:     "bg-blue-500/15 text-blue-400 border-blue-500/25",
        red:      "bg-red-500/15 text-red-400 border-red-500/25",
        amber:    "bg-amber-500/15 text-amber-400 border-amber-500/25",
        purple:   "bg-purple-500/15 text-purple-400 border-purple-500/25",
        cyan:     "bg-cyan-500/15 text-cyan-400 border-cyan-500/25",
      },
    },
    defaultVariants: { variant: "default" },
  }
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

export function Badge({ className, variant, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ variant }), className)} {...props} />;
}
