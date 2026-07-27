"use client";

import { useEffect, useRef, useState } from "react";
import { AnimatePresence, m } from "framer-motion";
import { Check, ChevronDown, Globe } from "lucide-react";
import { useLang, type Lang } from "@/lib/i18n";
import { cn } from "@/lib/utils";

// Переключатель языка в шапке: текущий язык одной кнопкой, тап — падает
// премиум-меню с полными названиями. Виден всегда (и гостям) — язык
// выбирается в одно касание из любого места приложения.

const LANGS: { code: Lang; short: string; full: string; flag: string }[] = [
  { code: "ru", short: "RU", full: "Русский",   flag: "🇷🇺" },
  { code: "uz", short: "UZ", full: "O'zbekcha", flag: "🇺🇿" },
  { code: "tg", short: "TJ", full: "Тоҷикӣ",    flag: "🇹🇯" },
];

export function LangSwitcher({ align = "right" }: { align?: "left" | "right" }) {
  const { lang, setLang } = useLang();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  // Закрытие по клику вне и по Escape.
  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent | TouchEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    document.addEventListener("mousedown", onDown);
    document.addEventListener("touchstart", onDown, { passive: true });
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("touchstart", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const current = LANGS.find((l) => l.code === lang) ?? LANGS[0];

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen((v) => !v)}
        aria-label={current.full}
        aria-expanded={open}
        className={cn(
          "flex items-center gap-1 rounded-md p-2 text-xs font-bold transition-colors",
          open ? "bg-zinc-800 text-yellow-400" : "text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100",
        )}
      >
        <Globe size={15} />
        <span className="tracking-wide">{current.short}</span>
        <ChevronDown size={11} className={cn("transition-transform duration-150", open && "rotate-180")} />
      </button>

      <AnimatePresence>
        {open && (
          <m.div
            initial={{ opacity: 0, y: -6, scale: 0.96 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -4, scale: 0.98 }}
            transition={{ duration: 0.15, ease: "easeOut" }}
            className={cn(
              "absolute top-full z-50 mt-1.5 w-44 origin-top overflow-hidden rounded-xl card-premium shadow-2xl shadow-black/50",
              align === "right" ? "right-0" : "left-0",
            )}
            role="listbox"
          >
            {LANGS.map((l) => (
              <button
                key={l.code}
                role="option"
                aria-selected={lang === l.code}
                onClick={() => { setLang(l.code); setOpen(false); }}
                className={cn(
                  "flex w-full items-center gap-2.5 px-3.5 py-2.5 text-left text-sm transition-colors",
                  lang === l.code
                    ? "bg-yellow-400/10 font-bold text-yellow-400"
                    : "font-semibold text-zinc-300 hover:bg-zinc-800/60 hover:text-zinc-100",
                )}
              >
                <span className="text-base leading-none" aria-hidden>{l.flag}</span>
                <span className="flex-1">{l.full}</span>
                {lang === l.code && <Check size={14} />}
              </button>
            ))}
          </m.div>
        )}
      </AnimatePresence>
    </div>
  );
}
