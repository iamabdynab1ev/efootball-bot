"use client";

import { useCountUp } from "@/lib/useCountUp";

/**
 * Число с плавным count-up при изменении значения.
 * Использовать вместе с `tabular-nums`, чтобы ширина не прыгала.
 */
export function CountUp({ value, className }: { value: number; className?: string }) {
  const display = useCountUp(value);
  return <span className={className}>{display}</span>;
}
