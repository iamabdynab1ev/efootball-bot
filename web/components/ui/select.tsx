"use client";

import { useEffect, useRef, useState } from "react";
import { AnimatePresence, m } from "framer-motion";
import { Check, ChevronDown } from "lucide-react";
import { cn } from "@/lib/utils";

export interface SelectOption {
  value: string;
  label: string;
}

interface SelectProps {
  value: string;
  onChange: (value: string) => void;
  options: SelectOption[];
  id?: string;
  ariaLabel?: string;
  /** Стили кнопки-триггера (высота/шрифт задаёт вызывающий). */
  className?: string;
  /** Стили контейнера (ширина). По умолчанию — на всю доступную. */
  containerClassName?: string;
  /** Цвет выбранного пункта: volt (жёлтый) или amber (double-elim). */
  accent?: "volt" | "amber";
}

// Select — выпадающий список в стиле приложения. Нативный <select> на
// телефонах открывает огромное системное меню, чужое тёмной теме, а длинные
// опции вылезают за рамки карточек — поэтому свой: меню раскрывается прямо
// в интерфейсе, длинные подписи аккуратно усечаются.
export function Select({ value, onChange, options, id, ariaLabel, className, containerClassName, accent = "volt" }: SelectProps) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const selected = options.find((o) => o.value === value);

  // Закрытие по тапу мимо и по Escape.
  useEffect(() => {
    if (!open) return;
    const onDown = (e: PointerEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => { if (e.key === "Escape") setOpen(false); };
    document.addEventListener("pointerdown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("pointerdown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const accentText = accent === "amber" ? "text-amber-400" : "text-yellow-400";

  return (
    <div ref={rootRef} className={cn("relative min-w-0", containerClassName ?? "w-full")}>
      <button
        type="button"
        id={id}
        aria-label={ariaLabel}
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
        className={cn(
          "flex w-full min-w-0 items-center justify-between gap-1.5 rounded-lg border border-zinc-700 bg-zinc-800 px-2.5 text-left text-xs text-zinc-200 transition-colors",
          open ? "border-yellow-400/60" : "hover:border-zinc-600",
          className,
        )}
      >
        <span className="truncate">{selected?.label ?? "—"}</span>
        <ChevronDown size={14} className={cn("flex-shrink-0 text-zinc-500 transition-transform duration-150", open && "rotate-180")} />
      </button>

      <AnimatePresence>
      {open && (
        <m.div
          role="listbox"
          initial={{ opacity: 0, y: -6, scale: 0.97 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          exit={{ opacity: 0, y: -4, scale: 0.98 }}
          transition={{ duration: 0.14, ease: "easeOut" }}
          className="absolute left-0 right-0 z-50 mt-1 max-h-64 origin-top overflow-y-auto overscroll-contain rounded-xl border border-zinc-700 bg-zinc-900 py-1 shadow-2xl shadow-black/60"
        >
          {options.map((o) => {
            const isSel = o.value === value;
            return (
              <button
                key={o.value}
                type="button"
                role="option"
                aria-selected={isSel}
                onClick={() => { onChange(o.value); setOpen(false); }}
                className={cn(
                  "flex w-full items-center justify-between gap-2 px-3 py-2.5 text-left text-xs transition-colors",
                  isSel ? cn("font-semibold", accentText) : "text-zinc-300 hover:bg-zinc-800 active:bg-zinc-800",
                )}
              >
                <span className="truncate">{o.label}</span>
                {isSel && <Check size={14} className="flex-shrink-0" />}
              </button>
            );
          })}
        </m.div>
      )}
      </AnimatePresence>
    </div>
  );
}
