"use client";

import { useEffect, useState } from "react";
import { AnimatePresence, m, useMotionValue, useSpring, useTransform } from "framer-motion";
import { X } from "lucide-react";
import { useLang } from "@/lib/i18n";

// Просмотр логотипа — как аватарка в соцсетях, но премиум: тап по знаку в
// шапке → щит крупно в центре, вращающиеся золотые лучи, скользящий блик и
// живой 3D-наклон за пальцем/курсором. Открытие: window-событие
// "logo:showcase" (кидает кнопка-логотип в Navbar).

export function openLogoShowcase() {
  window.dispatchEvent(new Event("logo:showcase"));
}

export function LogoShowcase() {
  const { t } = useLang();
  const [open, setOpen] = useState(false);

  useEffect(() => {
    const on = () => setOpen(true);
    window.addEventListener("logo:showcase", on);
    return () => window.removeEventListener("logo:showcase", on);
  }, []);

  // Пока показ открыт — блокируем прокрутку страницы под ним (html — реальный
  // скроллер приложения), чтобы жест по логотипу не листал фон.
  useEffect(() => {
    if (!open) return;
    const html = document.documentElement;
    const prev = html.style.overflow;
    html.style.overflow = "hidden";
    return () => { html.style.overflow = prev; };
  }, [open]);

  // 3D-наклон: позиция пальца/курсора → углы поворота (с пружиной).
  const px = useMotionValue(0.5);
  const py = useMotionValue(0.5);
  const rotateY = useSpring(useTransform(px, [0, 1], [-20, 20]), { stiffness: 160, damping: 18 });
  const rotateX = useSpring(useTransform(py, [0, 1], [16, -16]), { stiffness: 160, damping: 18 });

  const onMove = (e: React.PointerEvent<HTMLDivElement>) => {
    const r = e.currentTarget.getBoundingClientRect();
    px.set((e.clientX - r.left) / r.width);
    py.set((e.clientY - r.top) / r.height);
  };
  const onLeave = () => { px.set(0.5); py.set(0.5); };

  return (
    <AnimatePresence>
      {open && (
        <m.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.22 }}
          onClick={() => setOpen(false)}
          onPointerDown={onMove}
          onPointerMove={onMove}
          onPointerLeave={onLeave}
          className="fixed inset-0 z-[80] flex touch-none select-none items-center justify-center overflow-hidden bg-black/88 backdrop-blur-md"
          role="dialog"
          aria-label="Логотип eFootLeague"
        >
          {/* Вращающиеся золотые лучи */}
          <m.div
            animate={{ rotate: 360 }}
            transition={{ duration: 26, repeat: Infinity, ease: "linear" }}
            className="pointer-events-none absolute h-[170vmax] w-[170vmax] opacity-[0.13] [background:repeating-conic-gradient(from_0deg,rgba(250,204,21,0.9)_0deg_7deg,transparent_7deg_22deg)] [mask-image:radial-gradient(circle,black_0%,transparent_55%)]"
          />
          {/* Тёплое дыхание за щитом */}
          <m.div
            animate={{ scale: [1, 1.12, 1], opacity: [0.35, 0.55, 0.35] }}
            transition={{ duration: 3.6, repeat: Infinity, ease: "easeInOut" }}
            className="pointer-events-none absolute h-96 w-96 rounded-full bg-[radial-gradient(circle,rgba(250,204,21,0.35),transparent_65%)]"
          />

          <button
            aria-label="Закрыть"
            className="absolute right-3 top-[max(0.75rem,env(safe-area-inset-top))] flex h-10 w-10 items-center justify-center rounded-full bg-zinc-900/80 text-zinc-400 transition-colors hover:text-zinc-100"
          >
            <X size={18} />
          </button>

          <div onClick={(e) => e.stopPropagation()} className="relative flex flex-col items-center px-8 text-center [perspective:900px]">
            {/* Щит: пружинный вход + 3D-наклон за пальцем + плавание */}
            <m.div
              initial={{ scale: 0.4, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.6, opacity: 0 }}
              transition={{ type: "spring", stiffness: 220, damping: 20 }}
              style={{ rotateX, rotateY, transformStyle: "preserve-3d" }}
              className="relative"
            >
              <m.div
                animate={{ y: [0, -10, 0] }}
                transition={{ duration: 4.2, repeat: Infinity, ease: "easeInOut" }}
                className="relative"
              >
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  src="/icon-512.png"
                  alt="eFootLeague"
                  draggable={false}
                  className="h-[68vw] max-h-80 w-[68vw] max-w-80 select-none object-contain drop-shadow-[0_0_50px_rgba(250,204,21,0.4)]"
                />
                {/* Скользящий блик по щиту */}
                <m.div
                  animate={{ x: ["-130%", "130%"] }}
                  transition={{ duration: 2.8, repeat: Infinity, repeatDelay: 1.6, ease: "easeInOut" }}
                  className="pointer-events-none absolute inset-y-0 w-1/3 rotate-[18deg] bg-gradient-to-r from-transparent via-white/25 to-transparent blur-[6px]"
                />
              </m.div>
            </m.div>

            <m.p
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.25, duration: 0.3 }}
              className="mt-8 font-display text-3xl font-black text-zinc-50"
            >
              eFoot<span className="text-gradient-brand">League</span>
            </m.p>
            <m.p
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: 0.4, duration: 0.3 }}
              className="mt-1.5 text-sm font-semibold tracking-wide text-zinc-400"
            >
              {t("logo.tagline")}
            </m.p>
          </div>
        </m.div>
      )}
    </AnimatePresence>
  );
}
