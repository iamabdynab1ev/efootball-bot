"use client";

import { useEffect, useState } from "react";
import { AnimatePresence, m } from "framer-motion";
import { playSound } from "@/lib/sound";

// Полноэкранный момент получения награды: сервер выдал трофей/достижение →
// SSE-уведомление типа "award" → notifications.ts кидает событие
// "award:celebrate" → здесь взрыв: лучи, медаль пружиной, конфетти, фанфара.
// Игрок должен почувствовать, что победа того стоила.

interface AwardInfo {
  title: string; // «🏆 Новый трофей!» | «🏅 Новое достижение!»
  body: string;  // «🥈 «Серебро» · Лига Чемпионов ⚽»
}

async function fireAwardConfetti() {
  const confetti = (await import("canvas-confetti")).default;
  const colors = ["#facc15", "#c8f135", "#ffffff", "#f59e0b"];
  confetti({ particleCount: 140, spread: 100, origin: { y: 0.4 }, colors, zIndex: 90 });
  setTimeout(() => {
    confetti({ particleCount: 80, angle: 60, spread: 65, origin: { x: 0, y: 0.75 }, colors, zIndex: 90 });
    confetti({ particleCount: 80, angle: 120, spread: 65, origin: { x: 1, y: 0.75 }, colors, zIndex: 90 });
  }, 300);
}

export function AwardCelebration() {
  const [award, setAward] = useState<AwardInfo | null>(null);

  useEffect(() => {
    const onAward = (e: Event) => {
      const d = (e as CustomEvent).detail as AwardInfo | undefined;
      if (!d?.title) return;
      setAward(d);
      playSound("result", true); // фанфара — форсируем мимо троттлинга
      void fireAwardConfetti();
    };
    window.addEventListener("award:celebrate", onAward);
    return () => window.removeEventListener("award:celebrate", onAward);
  }, []);

  // Разбор тела уведомления: ведущий эмодзи, имя в «кавычках», лига после «·».
  const emoji = award?.body.match(/^\S+/)?.[0] ?? "🏆";
  const name = award?.body.match(/«([^»]+)»/)?.[1] ?? award?.body ?? "";
  const context = award?.body.split("·")[1]?.trim() ?? "";
  const kind = award?.title.includes("достижение") ? "Новое достижение" : "Новый трофей";

  return (
    <AnimatePresence>
      {award && (
        <m.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.25 }}
          onClick={() => setAward(null)}
          className="fixed inset-0 z-[80] flex items-center justify-center overflow-hidden bg-black/85 backdrop-blur-sm"
          role="dialog"
          aria-label={kind}
        >
          {/* Вращающиеся золотые лучи за медалью */}
          <m.div
            animate={{ rotate: 360 }}
            transition={{ duration: 18, repeat: Infinity, ease: "linear" }}
            className="pointer-events-none absolute h-[160vmax] w-[160vmax] opacity-20 [background:repeating-conic-gradient(from_0deg,rgba(250,204,21,0.9)_0deg_9deg,transparent_9deg_24deg)] [mask-image:radial-gradient(circle,black_0%,transparent_58%)]"
          />

          <div className="relative flex flex-col items-center px-8 text-center">
            {/* Медаль: пружинный вылет с лёгким поворотом */}
            <m.div
              initial={{ scale: 0, rotate: -20 }}
              animate={{ scale: 1, rotate: 0 }}
              transition={{ type: "spring", stiffness: 260, damping: 16, delay: 0.05 }}
              className="flex h-36 w-36 items-center justify-center rounded-full bg-gradient-to-br from-yellow-200 via-amber-400 to-yellow-600 shadow-[0_0_70px_rgba(250,204,21,0.45)] ring-4 ring-yellow-300/60"
            >
              <span className="text-[64px] leading-none drop-shadow-[0_4px_10px_rgba(0,0,0,0.45)]">{emoji}</span>
            </m.div>

            <m.p
              initial={{ opacity: 0, y: 14 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.25, duration: 0.3 }}
              className="mt-7 text-xs font-black uppercase tracking-[0.35em] text-yellow-400"
            >
              {kind}
            </m.p>
            <m.h2
              initial={{ opacity: 0, y: 14 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.35, duration: 0.3 }}
              className="mt-2 font-display text-4xl font-black text-zinc-50 drop-shadow-[0_0_30px_rgba(250,204,21,0.3)]"
            >
              {name}
            </m.h2>
            {context && (
              <m.p
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ delay: 0.45, duration: 0.3 }}
                className="mt-2 text-sm font-semibold text-zinc-400"
              >
                {context}
              </m.p>
            )}

            <m.button
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.55, duration: 0.3 }}
              onClick={() => setAward(null)}
              className="volt-grad volt-shadow mt-8 rounded-xl px-8 py-3 text-sm font-black text-zinc-950 transition-transform active:scale-95"
            >
              Забрать 🎉
            </m.button>
            <m.p
              initial={{ opacity: 0 }}
              animate={{ opacity: 0.6 }}
              transition={{ delay: 0.8 }}
              className="mt-3 text-[11px] text-zinc-500"
            >
              Награда уже в твоей витрине трофеев
            </m.p>
          </div>
        </m.div>
      )}
    </AnimatePresence>
  );
}
