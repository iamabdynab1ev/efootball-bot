"use client";

import { Suspense, useEffect, useMemo, useRef, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { AnimatePresence, m } from "framer-motion";
import { Share2, Trophy, X } from "lucide-react";
import { toast } from "sonner";
import { fetchSeasonSummary, type SeasonSummary } from "@/lib/api";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { shareTrophyCard } from "@/lib/shareCards";
import { playSound } from "@/lib/sound";
import { useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";

// Церемония закрытия сезона — премиум-шоу на весь экран: заставка с итогами,
// номинации по одной (конфетти + фанфара на каждой), финал с чемпионами лиг.
// Тап — следующая сцена, авто-переход через 6 секунд, прогресс-точки сверху.

const NOM_META: Record<string, { emoji: string; grad: string }> = {
  season_elo_growth:   { emoji: "🚀", grad: "from-sky-300 via-blue-400 to-indigo-600" },
  season_best_defense: { emoji: "🧱", grad: "from-slate-200 via-slate-400 to-slate-600" },
  season_top_scorer:   { emoji: "⚽", grad: "from-lime-200 via-lime-400 to-emerald-600" },
  season_player:       { emoji: "👑", grad: "from-yellow-200 via-amber-400 to-yellow-600" },
};

async function burst(big = false) {
  const confetti = (await import("canvas-confetti")).default;
  const colors = ["#facc15", "#c8f135", "#ffffff", "#f59e0b"];
  confetti({ particleCount: big ? 220 : 120, spread: big ? 120 : 90, origin: { y: 0.45 }, colors, zIndex: 90 });
  if (big) {
    setTimeout(() => {
      confetti({ particleCount: 90, angle: 60, spread: 60, origin: { x: 0, y: 0.8 }, colors, zIndex: 90 });
      confetti({ particleCount: 90, angle: 120, spread: 60, origin: { x: 1, y: 0.8 }, colors, zIndex: 90 });
    }, 350);
  }
}

function CeremonyInner() {
  const params = useSearchParams();
  const router = useRouter();
  const { t } = useLang();
  const id = Number(params.get("id"));
  const [data, setData] = useState<SeasonSummary | null>(null);
  const [err, setErr] = useState(false);
  const [stage, setStage] = useState(0);
  const timer = useRef<number | null>(null);

  useEffect(() => {
    if (!id) { setErr(true); return; }
    fetchSeasonSummary(id).then(setData).catch(() => setErr(true));
  }, [id]);

  // Сцены: 0 — заставка; 1..N — номинации; N+1 — финал с чемпионами.
  const stagesTotal = (data?.nominations.length ?? 0) + 2;
  const isFinale = data ? stage === stagesTotal - 1 : false;

  const next = () => setStage((s) => Math.min(s + 1, stagesTotal - 1));

  // Авто-переход + звук/конфетти на каждой сцене.
  useEffect(() => {
    if (!data) return;
    if (stage > 0) {
      playSound("result", true);
      void burst(isFinale || stage === stagesTotal - 2);
    }
    if (!isFinale) {
      timer.current = window.setTimeout(next, 6000);
      return () => { if (timer.current) window.clearTimeout(timer.current); };
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [stage, data]);

  const nomLabel = (type: string) => t(`seasonNoms.${type.replace("season_", "")}` as never) as string;

  const share = (name: string, type: string) => {
    shareTrophyCard({
      emoji: NOM_META[type]?.emoji ?? "🏆",
      label: nomLabel(type),
      playerName: name,
      context: data?.season.name ?? "",
    }).catch(() => toast.error(t("award.shareFail")));
  };

  if (err) {
    return (
      <div className="flex min-h-[70vh] flex-col items-center justify-center gap-3 text-zinc-500">
        <Trophy size={32} className="text-zinc-700" />
        <p className="text-sm">{t("season.notFound")}</p>
      </div>
    );
  }
  if (!data) {
    return <div className="flex min-h-[70vh] items-center justify-center"><div className="skeleton h-40 w-72 rounded-2xl" /></div>;
  }

  const nom = stage >= 1 && stage <= data.nominations.length ? data.nominations[stage - 1] : null;
  const meta = nom ? (NOM_META[nom.type] ?? NOM_META.season_player) : null;

  return (
    <div
      className="fixed inset-0 z-[70] flex flex-col overflow-hidden bg-black/95"
      onClick={next}
      role="presentation"
    >
      {/* Золотые лучи */}
      <m.div
        animate={{ rotate: 360 }}
        transition={{ duration: 24, repeat: Infinity, ease: "linear" }}
        className="pointer-events-none absolute left-1/2 top-1/2 h-[180vmax] w-[180vmax] -translate-x-1/2 -translate-y-1/2 opacity-15 [background:repeating-conic-gradient(from_0deg,rgba(250,204,21,0.8)_0deg_8deg,transparent_8deg_26deg)] [mask-image:radial-gradient(circle,black_0%,transparent_55%)]"
      />

      {/* Прогресс + выход */}
      <div className="relative z-10 flex items-center gap-2 px-4 pt-[max(1rem,env(safe-area-inset-top))]">
        <div className="flex flex-1 items-center gap-1.5">
          {Array.from({ length: stagesTotal }).map((_, i) => (
            <span key={i} className={cn(
              "h-1 flex-1 rounded-full transition-colors",
              i <= stage ? "bg-yellow-400" : "bg-zinc-800",
            )} />
          ))}
        </div>
        <button
          onClick={(e) => { e.stopPropagation(); router.push("/hall-of-fame"); }}
          aria-label={t("season.skip")}
          className="rounded-full bg-white/10 p-2 text-zinc-300 transition-colors hover:bg-white/20"
        >
          <X size={16} />
        </button>
      </div>

      <div className="relative z-10 flex flex-1 items-center justify-center px-6 pb-16">
        <AnimatePresence mode="wait">
          {/* ── Заставка ── */}
          {stage === 0 && (
            <m.div key="intro" initial={{ opacity: 0, scale: 0.94 }} animate={{ opacity: 1, scale: 1 }} exit={{ opacity: 0, y: -18 }} transition={{ duration: 0.4 }} className="text-center">
              <m.div
                initial={{ scale: 0, rotate: -15 }} animate={{ scale: 1, rotate: 0 }}
                transition={{ type: "spring", stiffness: 220, damping: 14, delay: 0.2 }}
                className="mx-auto flex h-24 w-24 items-center justify-center rounded-full bg-gradient-to-br from-yellow-200 via-amber-400 to-yellow-600 shadow-[0_0_60px_rgba(250,204,21,0.4)]"
              >
                <Trophy size={44} className="text-zinc-900" />
              </m.div>
              <p className="mt-8 text-xs font-black uppercase tracking-[0.4em] text-yellow-400">{t("season.ceremony")}</p>
              <h1 className="mt-3 font-display text-4xl font-black text-zinc-50">{data.season.name}</h1>
              <p className="mt-2 text-sm font-semibold uppercase tracking-[0.25em] text-zinc-400">{t("season.finished")}</p>
              <div className="mt-8 flex items-center justify-center gap-6 text-center">
                {[
                  { v: data.totals.matches, l: t("season.matches") },
                  { v: data.totals.goals, l: t("season.goals") },
                  { v: data.totals.players, l: t("season.players") },
                ].map((x, i) => (
                  <m.div key={i} initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.5 + i * 0.15 }}>
                    <p className="font-display text-3xl font-black text-yellow-400 tabular-nums">{x.v}</p>
                    <p className="mt-1 text-[10px] font-bold uppercase tracking-widest text-zinc-500">{x.l}</p>
                  </m.div>
                ))}
              </div>
              <p className="mt-10 animate-pulse text-[11px] uppercase tracking-[0.3em] text-zinc-600">{t("season.tapNext")}</p>
            </m.div>
          )}

          {/* ── Номинация ── */}
          {nom && meta && (
            <m.div key={nom.type} initial={{ opacity: 0, y: 24 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -18 }} transition={{ duration: 0.35 }} className="text-center">
              <p className="text-xs font-black uppercase tracking-[0.35em] text-yellow-400">{nomLabel(nom.type)}</p>
              <m.div
                initial={{ scale: 0 }} animate={{ scale: 1 }}
                transition={{ type: "spring", stiffness: 240, damping: 15, delay: 0.15 }}
                className={cn("mx-auto mt-7 flex h-32 w-32 items-center justify-center rounded-full bg-gradient-to-br ring-4 ring-white/20 shadow-[0_0_60px_rgba(250,204,21,0.35)]", meta.grad)}
              >
                <span className="text-[56px] leading-none drop-shadow-[0_4px_10px_rgba(0,0,0,0.5)]">{meta.emoji}</span>
              </m.div>
              <m.div initial={{ opacity: 0, y: 14 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.4 }} className="mt-6 flex items-center justify-center gap-3">
                <PlayerAvatar displayName={nom.name} favoriteClub={nom.club} size={40} />
                <h2 className="font-display text-3xl font-black text-zinc-50">{nom.name}</h2>
              </m.div>
              <m.p initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: 0.55 }} className="mt-2 text-sm font-bold text-zinc-400">
                {nomValueLabel(nom.type, nom.value, t)}
              </m.p>
              <m.button
                initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: 0.7 }}
                onClick={(e) => { e.stopPropagation(); share(nom.name, nom.type); }}
                className="mx-auto mt-6 flex items-center gap-2 rounded-xl border border-yellow-400/40 px-4 py-2 text-xs font-bold text-yellow-400 transition-transform active:scale-95"
              >
                <Share2 size={14} /> {t("season.share")}
              </m.button>
            </m.div>
          )}

          {/* ── Финал: чемпионы лиг ── */}
          {isFinale && (
            <m.div key="finale" initial={{ opacity: 0, scale: 0.96 }} animate={{ opacity: 1, scale: 1 }} transition={{ duration: 0.4 }} className="w-full max-w-md text-center">
              <p className="text-xs font-black uppercase tracking-[0.35em] text-yellow-400">{t("season.champions")}</p>
              <div className="mt-6 space-y-2.5">
                {data.champions.map((c, i) => (
                  <m.div
                    key={c.league_id}
                    initial={{ opacity: 0, x: -18 }} animate={{ opacity: 1, x: 0 }} transition={{ delay: 0.2 + i * 0.15 }}
                    className="flex items-center gap-3 rounded-xl border border-amber-500/25 bg-amber-500/5 px-4 py-3"
                  >
                    <span className="text-xl" aria-hidden>🏆</span>
                    <PlayerAvatar displayName={c.name} favoriteClub={c.club} size={34} />
                    <div className="min-w-0 flex-1 text-left">
                      <p className="truncate text-sm font-black text-zinc-100">{c.name}</p>
                      <p className="truncate text-[11px] text-zinc-500">{c.league_name}</p>
                    </div>
                  </m.div>
                ))}
                {data.champions.length === 0 && (
                  <p className="text-sm text-zinc-500">{t("season.noChampions")}</p>
                )}
              </div>
              <m.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.6 }} className="mt-8 flex justify-center gap-2">
                <button
                  onClick={(e) => { e.stopPropagation(); router.push("/hall-of-fame"); }}
                  className="volt-grad volt-shadow rounded-xl px-6 py-3 text-sm font-black text-zinc-950 transition-transform active:scale-95"
                >
                  {t("season.toHall")}
                </button>
              </m.div>
              <m.p initial={{ opacity: 0 }} animate={{ opacity: 0.6 }} transition={{ delay: 0.9 }} className="mt-4 text-[11px] text-zinc-500">
                {t("season.newSeason")}
              </m.p>
            </m.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}

function nomValueLabel(type: string, value: number, t: (k: never) => string): string {
  switch (type) {
    case "season_top_scorer":   return `${value} ${t("seasonNoms.goalsSuffix" as never)}`;
    case "season_best_defense": return `${value} ${t("seasonNoms.concededSuffix" as never)}`;
    case "season_elo_growth":   return `+${value} ELO`;
    default:                    return `${value} ${t("seasonNoms.pointsSuffix" as never)}`;
  }
}

export default function SeasonCeremonyPage() {
  return (
    <Suspense fallback={<div className="py-10 text-center text-sm text-zinc-500">…</div>}>
      <CeremonyInner />
    </Suspense>
  );
}
