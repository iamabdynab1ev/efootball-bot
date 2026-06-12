"use client";

import { useEffect } from "react";
import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { Trophy, X } from "lucide-react";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { useLang } from "@/lib/i18n";

interface Props {
  open: boolean;
  onClose: () => void;
  championName: string;
  championClub?: string;
  leagueName?: string;
  /** Финальный счёт, например "3:1" (опционально). */
  finalScore?: string;
}

/** Конфетти подгружается динамически (~6KB) только при показе оверлея. */
async function fireConfetti() {
  const confetti = (await import("canvas-confetti")).default;
  const gold = ["#fde047", "#f59e0b", "#fbbf24", "#c8f135"];
  confetti({ particleCount: 90, spread: 75, origin: { y: 0.65 }, colors: gold, zIndex: 100 });
  setTimeout(() => {
    confetti({ particleCount: 60, angle: 60, spread: 60, origin: { x: 0, y: 0.7 }, colors: gold, zIndex: 100 });
    confetti({ particleCount: 60, angle: 120, spread: 60, origin: { x: 1, y: 0.7 }, colors: gold, zIndex: 100 });
  }, 350);
}

/**
 * Полноэкранная церемония чемпиона: виньетка → золотые лучи →
 * spring-трофей → имя. Показывается автоматически при завершении финала
 * (см. страницу лиги) и по кнопке 🎉 на баннере чемпиона.
 */
export function ChampionCelebration({ open, onClose, championName, championClub, leagueName, finalScore }: Props) {
  const { t } = useLang();
  const reduced = useReducedMotion();

  useEffect(() => {
    if (!open) return;
    if (!reduced) fireConfetti();
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, reduced, onClose]);

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="fixed inset-0 z-50 flex items-center justify-center p-6"
          role="dialog"
          aria-modal="true"
          aria-label={t("leagueDetail.bracketChampion")}
          onClick={onClose}
        >
          {/* Виньетка */}
          <div className="absolute inset-0 bg-zinc-950/90 backdrop-blur-sm" />

          {/* Золотые лучи — статичный conic-gradient, без анимации */}
          <div
            aria-hidden="true"
            className="absolute inset-0 opacity-25"
            style={{
              background:
                "conic-gradient(from 0deg at 50% 42%, transparent 0deg, rgb(245 158 11 / 0.5) 8deg, transparent 16deg, transparent 45deg, rgb(245 158 11 / 0.35) 53deg, transparent 61deg, transparent 90deg, rgb(245 158 11 / 0.5) 98deg, transparent 106deg, transparent 135deg, rgb(245 158 11 / 0.35) 143deg, transparent 151deg, transparent 180deg, rgb(245 158 11 / 0.5) 188deg, transparent 196deg, transparent 225deg, rgb(245 158 11 / 0.35) 233deg, transparent 241deg, transparent 270deg, rgb(245 158 11 / 0.5) 278deg, transparent 286deg, transparent 315deg, rgb(245 158 11 / 0.35) 323deg, transparent 331deg, transparent 360deg)",
            }}
          />

          <motion.div
            initial={reduced ? false : { scale: 0.9, y: 12, opacity: 0 }}
            animate={{ scale: 1, y: 0, opacity: 1 }}
            exit={{ scale: 0.95, opacity: 0 }}
            transition={{ type: "spring", stiffness: 220, damping: 20 }}
            className="relative w-full max-w-sm rounded-2xl border border-amber-500/30 bg-zinc-900/95 px-8 py-10 text-center glow-gold"
            onClick={(e) => e.stopPropagation()}
          >
            <button
              onClick={onClose}
              aria-label="✕"
              className="absolute right-3 top-3 rounded-lg p-1.5 text-zinc-500 transition-colors hover:bg-zinc-800 hover:text-zinc-200"
            >
              <X size={16} />
            </button>

            <motion.div
              initial={reduced ? false : { scale: 0, rotate: -25 }}
              animate={{ scale: 1, rotate: 0 }}
              transition={{ type: "spring", stiffness: 240, damping: 12, delay: 0.2 }}
              className="mx-auto mb-5 flex h-20 w-20 items-center justify-center rounded-full text-zinc-900"
              style={{ background: "var(--grad-gold)" }}
            >
              <Trophy size={40} strokeWidth={2.2} />
            </motion.div>

            {leagueName && (
              <p className="mb-1 text-xs font-semibold uppercase tracking-widest text-zinc-500 truncate">
                {leagueName}
              </p>
            )}
            <p className="mb-3 text-[11px] font-bold uppercase tracking-[0.25em] text-amber-400">
              {t("leagueDetail.bracketChampion")}
            </p>

            <div className="mb-2 flex items-center justify-center gap-3 min-w-0">
              <PlayerAvatar displayName={championName} favoriteClub={championClub} size={40} />
              <h2 className="font-display text-3xl font-black text-gradient-gold break-words min-w-0">
                {championName}
              </h2>
            </div>

            {finalScore && (
              <p className="font-display text-lg font-bold text-zinc-400 tabular-nums">{finalScore}</p>
            )}
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
