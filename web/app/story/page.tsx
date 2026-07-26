"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import dynamic from "next/dynamic";
import { m, useMotionValueEvent, useScroll, useTransform } from "framer-motion";
import { ChevronDown, Swords, TrendingUp, Trophy, X } from "lucide-react";
import { BrandLogo } from "@/components/BrandLogo";
import { useAuth } from "@/lib/auth";
import { tr } from "@/lib/i18n";

// Настоящее 3D (three.js) грузим лениво и только при поддержке WebGL —
// слабые устройства получают CSS-версию сцены, интро работает у всех.
const Scene3D = dynamic(() => import("@/components/story/Scene3D"), { ssr: false });

// «Финал. 90-я минута» — кинематографичное скролл-интро (в духе
// why.zero.university, но лёгкое: без WebGL, только scroll-driven анимации).
// Скролл двигает матч: титры → «Последняя атака» → УДАР → полёт мяча →
// ГОООЛ со вспышкой, тряской камеры и конфетти → призыв начать играть.
// Работает плавно и на слабых телефонах (90% аудитории — смартфоны).

// Золотое конфетти в момент гола (динамический импорт — не грузим зря).
async function fireGoalConfetti() {
  const confetti = (await import("canvas-confetti")).default;
  const colors = ["#c8f135", "#facc15", "#ffffff"];
  confetti({ particleCount: 120, spread: 85, origin: { x: 0.7, y: 0.45 }, colors, zIndex: 60 });
  setTimeout(() => {
    confetti({ particleCount: 70, angle: 60, spread: 60, origin: { x: 0, y: 0.8 }, colors, zIndex: 60 });
    confetti({ particleCount: 70, angle: 120, spread: 60, origin: { x: 1, y: 0.8 }, colors, zIndex: 60 });
  }, 250);
}

// Рисованный футбольный мяч: градиентная сфера, пятиугольники, блик.
function Ball({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 100 100" className={className} aria-hidden="true">
      <defs>
        <radialGradient id="ballShade" cx="38%" cy="32%" r="75%">
          <stop offset="0%" stopColor="#ffffff" />
          <stop offset="55%" stopColor="#e8e8ec" />
          <stop offset="100%" stopColor="#8e8e99" />
        </radialGradient>
      </defs>
      <circle cx="50" cy="50" r="48" fill="url(#ballShade)" />
      {/* Пятиугольники классического мяча */}
      <g fill="#17181d">
        <path d="M50,32 L64,42 L59,58 L41,58 L36,42 Z" />
        <path d="M50,2 L60,7 L57,18 L43,18 L40,7 Z" />
        <path d="M88,30 L96,42 L90,52 L79,46 L80,34 Z" />
        <path d="M12,30 L20,34 L21,46 L10,52 L4,42 Z" />
        <path d="M74,78 L80,66 L92,68 L90,80 L79,88 Z" />
        <path d="M26,78 L21,88 L10,80 L8,68 L20,66 Z" />
        <path d="M43,94 L57,94 L60,86 L50,80 L40,86 Z" />
      </g>
      {/* Швы */}
      <g stroke="#17181d" strokeWidth="2.4" fill="none" opacity="0.8">
        <path d="M50,32 L50,18" /><path d="M64,42 L80,34" /><path d="M59,58 L74,78 M41,58 L26,78" /><path d="M36,42 L21,34" />
      </g>
      <circle cx="50" cy="50" r="48" fill="none" stroke="#0c0d12" strokeWidth="2.5" />
      {/* Блик */}
      <ellipse cx="36" cy="26" rx="14" ry="9" fill="#ffffff" opacity="0.55" transform="rotate(-25 36 26)" />
    </svg>
  );
}

export default function StoryPage() {
  const { user } = useAuth();
  const scrollRef = useRef<HTMLDivElement>(null);
  const trackRef = useRef<HTMLDivElement>(null);
  const { scrollYProgress: p } = useScroll({
    container: scrollRef,
    target: trackRef,
    offset: ["start start", "end end"],
  });

  // Интро показывается гостям один раз при заходе — помечаем просмотр.
  useEffect(() => {
    try { localStorage.setItem("story_seen", "1"); } catch { /* private mode */ }
  }, []);

  // WebGL-детект: null — проверяем, true — 3D-сцена, false — CSS-фолбэк.
  const [webgl, setWebgl] = useState<boolean | null>(null);
  useEffect(() => {
    try {
      const c = document.createElement("canvas");
      setWebgl(!!(c.getContext("webgl2") || c.getContext("webgl")));
    } catch {
      setWebgl(false);
    }
  }, []);

  // Табло: секунды тикают по скроллу, гол фиксируется один раз.
  const [clock, setClock] = useState("89:00");
  const [scored, setScored] = useState(false);
  useMotionValueEvent(p, "change", (v) => {
    if (v > 0.62) {
      if (!scored) { setScored(true); void fireGoalConfetti(); }
      setClock("90:00");
      return;
    }
    if (scored) setScored(false); // проскроллили назад — матч «переигрывается»
    const t = Math.min(Math.max((v - 0.16) / 0.46, 0), 1);
    setClock(`89:${String(Math.floor(t * 59)).padStart(2, "0")}`);
  });

  // ── Таймлайн сцен (прогресс p 0..1) ────────────────────────────────
  const titleOpacity = useTransform(p, [0, 0.12, 0.18], [1, 1, 0]);
  const titleY = useTransform(p, [0, 0.18], ["0vh", "-8vh"]);
  const hintOpacity = useTransform(p, [0, 0.07], [1, 0]);
  const boardOpacity = useTransform(p, [0.14, 0.2], [0, 1]);

  // «Камера»: медленный наезд в сторону ворот + тряска в момент гола.
  const camScale = useTransform(p, [0.3, 0.62], [1, 1.13]);
  const camX = useTransform(p, [0.62, 0.632, 0.644, 0.656, 0.67], [0, -9, 7, -4, 0]);

  // Кинетические титры.
  const attackOpacity = useTransform(p, [0.17, 0.21, 0.28, 0.31], [0, 1, 1, 0]);
  const attackX = useTransform(p, [0.17, 0.31], ["14vw", "-14vw"]);
  const strikeOpacity = useTransform(p, [0.43, 0.455, 0.5, 0.52], [0, 1, 1, 0]);
  const strikeScale = useTransform(p, [0.43, 0.47], [2.6, 1]);

  // Карточка легенды № 7 — выходит на удар (фото: CC BY-SA, атрибуция в финале).
  const cardOpacity = useTransform(p, [0.3, 0.335, 0.41, 0.45], [0, 1, 1, 0]);
  const cardY = useTransform(p, [0.3, 0.45], ["14vh", "-6vh"]);
  const cardRotate = useTransform(p, [0.3, 0.45], [-8, -1]);
  const cardScale = useTransform(p, [0.3, 0.36], [0.85, 1]);

  // Ворота выезжают справа.
  const goalX = useTransform(p, [0.28, 0.42], ["130%", "0%"]);
  const netRipple = useTransform(p, [0.615, 0.64, 0.68], [1, 1.05, 1]);

  // Мяч: катится издалека → точка удара → полёт по дуге в девятку.
  const ballX = useTransform(p, [0.18, 0.45, 0.63], ["-38vw", "-6vw", "24vw"]);
  const ballY = useTransform(p, [0.18, 0.45, 0.54, 0.63], ["0vh", "0vh", "-26vh", "-13vh"]);
  const ballRotate = useTransform(p, [0.18, 0.63], [0, 1080]);
  const ballScale = useTransform(p, [0.18, 0.3, 0.45, 0.63], [0.45, 0.8, 1, 0.6]);
  const ballOpacity = useTransform(p, [0.15, 0.19, 0.66, 0.7], [0, 1, 1, 0]);
  // Тень на газоне: отрывается от мяча в полёте.
  const shadowScale = useTransform(p, [0.18, 0.45, 0.54, 0.63], [0.45, 1, 0.4, 0.55]);
  const shadowOpacity = useTransform(p, [0.15, 0.19, 0.45, 0.54, 0.66], [0, 0.5, 0.5, 0.2, 0]);
  const trailOpacity = useTransform(p, [0.45, 0.5, 0.63], [0, 0.7, 0]);

  // ГОЛ: вспышка + надпись.
  const flashOpacity = useTransform(p, [0.6, 0.622, 0.655], [0, 0.9, 0]);
  const goalTextScale = useTransform(p, [0.62, 0.68], [0.4, 1]);
  const goalTextOpacity = useTransform(p, [0.62, 0.66, 0.8, 0.86], [0, 1, 1, 0]);

  // Финал: призыв к действию.
  const ctaOpacity = useTransform(p, [0.84, 0.94], [0, 1]);
  const ctaY = useTransform(p, [0.84, 0.94], ["6vh", "0vh"]);

  return (
    <div ref={scrollRef} className="fixed inset-0 z-50 overflow-y-auto overscroll-contain bg-[#05070b] [scroll-behavior:auto]">
      {/* Выход из истории */}
      <Link
        href={user ? "/" : "/login"}
        aria-label="Закрыть"
        className="fixed right-3 top-[max(0.75rem,env(safe-area-inset-top))] z-40 flex h-10 w-10 items-center justify-center rounded-full bg-zinc-900/80 text-zinc-400 backdrop-blur-sm transition-colors hover:text-zinc-100"
      >
        <X size={18} />
      </Link>

      <div ref={trackRef} className="relative h-[520vh]">
        <div className="sticky top-0 h-dvh overflow-hidden">

          {/* ── Настоящее 3D (WebGL) — камера летит за мячом ── */}
          {webgl && <Scene3D progress={p} />}

          {/* ── CSS-фолбэк для устройств без WebGL ── */}
          {webgl === false && (
          <m.div style={{ scale: camScale, x: camX }} className="absolute inset-0 [transform-origin:68%_72%]">

            {/* Ночное небо над чашей стадиона */}
            <div className="absolute inset-0 bg-[radial-gradient(130%_100%_at_50%_-15%,#101d33_0%,#0a1220_45%,#05070b_100%)]" />

            {/* Дальняя трибуна: силуэт чаши + «вспышки телефонов» */}
            <div className="absolute inset-x-[-10%] top-[30%] h-[24%] rounded-[100%_100%_0_0/60%_60%_0_0] bg-gradient-to-b from-[#131a26] to-[#0a0f18]" />
            <div className="absolute inset-x-0 top-[33%] h-[18%] opacity-50 [background:radial-gradient(circle_at_8%_30%,rgba(255,255,255,0.55)_0.6px,transparent_1.4px),radial-gradient(circle_at_33%_72%,rgba(255,255,255,0.4)_0.5px,transparent_1.2px),radial-gradient(circle_at_58%_24%,rgba(255,255,255,0.5)_0.6px,transparent_1.4px),radial-gradient(circle_at_82%_62%,rgba(255,255,255,0.42)_0.5px,transparent_1.2px)] [background-size:70px_44px,90px_56px,80px_40px,100px_64px]" />

            {/* Прожекторы: точки света + лучи */}
            <div className="absolute left-[12%] top-[24%] h-3 w-16 rounded-full bg-white/70 blur-[6px]" />
            <div className="absolute right-[12%] top-[24%] h-3 w-16 rounded-full bg-white/70 blur-[6px]" />
            <div className="absolute left-[6%] top-[22%] h-[55vh] w-40 rotate-[18deg] bg-gradient-to-b from-white/12 to-transparent blur-2xl" />
            <div className="absolute right-[6%] top-[22%] h-[55vh] w-40 -rotate-[18deg] bg-gradient-to-b from-white/10 to-transparent blur-2xl" />

            {/* Свечение горизонта на стыке трибун и поля */}
            <div className="absolute inset-x-0 bottom-[52%] h-10 bg-gradient-to-t from-white/10 to-transparent blur-md" />

            {/* Газон в перспективе */}
            <div className="absolute inset-x-0 bottom-0 h-[54%] [perspective:700px]">
              <div className="absolute -inset-x-[25%] bottom-[-12%] top-0 origin-bottom [transform:rotateX(48deg)] bg-[repeating-linear-gradient(90deg,#0c1a11_0,#0c1a11_9vw,#102416_9vw,#102416_18vw)]">
                {/* Разметка: боковая, штрафная, круг */}
                <div className="absolute inset-x-[10%] top-[8%] h-[3px] bg-white/20" />
                <div className="absolute left-1/2 top-[8%] h-[52%] w-[34%] -translate-x-1/2 border-[3px] border-t-0 border-white/15" />
                <div className="absolute left-1/2 top-[38%] h-24 w-24 -translate-x-1/2 rounded-full border-[3px] border-white/10" />
              </div>
              {/* Дымка над газоном + виньетка */}
              <div className="absolute inset-0 bg-gradient-to-b from-[#05070b] via-transparent to-black/60" />
            </div>

            {/* ── Ворота (перспектива, сетка вздрагивает на голе) ── */}
            <m.div style={{ x: goalX }} className="absolute bottom-[9%] right-[2vw] z-[5] h-[30vh] w-[34vw] max-w-[330px] [perspective:600px]">
              <m.div
                style={{ scale: netRipple }}
                className="h-full w-full origin-bottom-right rounded-tl-lg border-l-[6px] border-t-[6px] border-white/90 shadow-[inset_0_0_40px_rgba(0,0,0,0.5)] [transform:rotateY(-16deg)] [background:repeating-linear-gradient(0deg,rgba(255,255,255,0.16)_0,rgba(255,255,255,0.16)_1px,transparent_1px,transparent_12px),repeating-linear-gradient(90deg,rgba(255,255,255,0.16)_0,rgba(255,255,255,0.16)_1px,transparent_1px,transparent_12px)]"
              />
            </m.div>

            {/* ── Тень мяча на газоне ── */}
            <m.div
              style={{ x: ballX, scale: shadowScale, opacity: shadowOpacity }}
              className="absolute bottom-[10%] left-1/2 z-[6] -ml-10 h-4 w-20 rounded-[100%] bg-black/80 blur-[5px]"
            />

            {/* ── Мяч ── */}
            <m.div
              style={{ x: ballX, y: ballY, opacity: ballOpacity, scale: ballScale }}
              className="absolute bottom-[10.5%] left-1/2 z-10 -ml-10"
            >
              {/* Неоновый след удара */}
              <m.div style={{ opacity: trailOpacity }} className="absolute right-14 top-1/2 h-2 w-44 -translate-y-1/2 rotate-[24deg] rounded-full bg-gradient-to-l from-yellow-300 via-yellow-400/50 to-transparent blur-[3px]" />
              <m.div style={{ rotate: ballRotate }} className="h-20 w-20 drop-shadow-[0_14px_22px_rgba(0,0,0,0.65)]">
                <Ball className="h-full w-full" />
              </m.div>
            </m.div>
          </m.div>
          )}

          {/* ── Табло ── */}
          <m.div style={{ opacity: boardOpacity }} className="absolute left-1/2 top-[max(1rem,env(safe-area-inset-top))] z-20 -translate-x-1/2">
            <div className={`flex items-center gap-2 whitespace-nowrap rounded-xl border px-3 py-2 font-display backdrop-blur-sm transition-colors duration-300 ${scored ? "border-yellow-400/60 bg-yellow-400/10" : "border-zinc-700/60 bg-zinc-950/70"}`}>
              <span className="text-xs font-black text-zinc-100">{tr("story.you")}</span>
              <span className={`text-lg font-black leading-none tabular-nums ${scored ? "text-yellow-400" : "text-zinc-100"}`}>{scored ? "1:0" : "0:0"}</span>
              <span className="text-xs font-black text-zinc-400">{tr("story.rival")}</span>
              <span className="rounded bg-zinc-800 px-1.5 py-0.5 text-[11px] font-bold tabular-nums text-yellow-400">{clock}</span>
            </div>
          </m.div>

          {/* ── Титры ── */}
          <m.div style={{ opacity: titleOpacity, y: titleY }} className="absolute inset-0 z-10 flex flex-col items-center justify-center px-6 text-center">
            <BrandLogo size={84} />
            <p className="mt-5 text-xs font-bold uppercase tracking-[0.35em] text-yellow-400">{tr("story.kicker")}</p>
            <h1 className="mt-2 font-display text-4xl font-black leading-tight text-zinc-50 sm:text-5xl">
              {tr("story.title1")}<br />{tr("story.title2")}
            </h1>
            <p className="mt-3 max-w-xs text-sm text-zinc-400">
              {tr("story.hint")}
            </p>
          </m.div>
          <m.div style={{ opacity: hintOpacity }} className="absolute inset-x-0 bottom-[max(1.5rem,env(safe-area-inset-bottom))] z-10 flex flex-col items-center gap-1 text-zinc-500">
            <span className="text-[10px] font-bold uppercase tracking-[0.3em]">{tr("story.scroll")}</span>
            <m.span animate={{ y: [0, 6, 0] }} transition={{ duration: 1.4, repeat: Infinity, ease: "easeInOut" }}>
              <ChevronDown size={18} />
            </m.span>
          </m.div>

          {/* ── Кинетические титры ── */}
          <m.p
            style={{ opacity: attackOpacity, x: attackX }}
            className="absolute inset-x-0 top-[22%] z-10 text-center font-display text-3xl font-black uppercase italic tracking-widest text-zinc-100/90 sm:text-4xl"
          >
            {tr("story.lastAttack")}
          </m.p>
          <m.p
            style={{ opacity: strikeOpacity, scale: strikeScale }}
            className="absolute inset-x-0 top-[30%] z-10 text-center font-display text-6xl font-black uppercase italic text-yellow-400 drop-shadow-[0_0_30px_rgba(200,241,53,0.45)] sm:text-7xl"
          >
            {tr("story.strike")}
          </m.p>

          {/* ── Карточка легенды № 7 (стиль игровых карт) ── */}
          <m.div
            style={{ opacity: cardOpacity, y: cardY, rotate: cardRotate, scale: cardScale }}
            className="pointer-events-none absolute inset-0 z-10 flex items-center justify-center px-6"
          >
            <div className="w-60 overflow-hidden rounded-2xl border border-yellow-400/60 bg-gradient-to-b from-zinc-800 to-zinc-950 shadow-[0_0_70px_rgba(200,241,53,0.2)]">
              <div className="relative">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img src="/story/legend7.jpg" alt={tr("story.legend")} className="h-64 w-full object-cover object-top" />
                <div className="absolute inset-0 bg-gradient-to-t from-zinc-950 via-transparent to-transparent" />
                <span className="absolute left-3 top-1 font-display text-5xl font-black text-yellow-400 drop-shadow-[0_2px_8px_rgba(0,0,0,0.8)]">7</span>
              </div>
              <div className="px-4 pb-3.5 pt-1 text-center">
                <p className="font-display text-lg font-black tracking-wide text-zinc-50">{tr("story.legend")}</p>
                <p className="mt-0.5 text-[10px] font-bold uppercase tracking-[0.3em] text-yellow-400">{tr("story.legendSub")}</p>
              </div>
            </div>
          </m.div>

          {/* ── ГОЛ ── */}
          <m.div style={{ opacity: flashOpacity }} className="pointer-events-none absolute inset-0 z-20 bg-white" />
          <m.div style={{ opacity: goalTextOpacity, scale: goalTextScale }} className="absolute inset-0 z-20 flex flex-col items-center justify-center px-6 text-center">
            <p className="font-display text-6xl font-black tracking-tight text-yellow-400 drop-shadow-[0_0_40px_rgba(200,241,53,0.5)] sm:text-7xl">
              {tr("story.goal")}
            </p>
            <p className="mt-3 text-sm font-semibold uppercase tracking-[0.25em] text-zinc-300">{tr("story.lastMinute")}</p>
          </m.div>

          {/* ── Призыв ── */}
          <m.div style={{ opacity: ctaOpacity, y: ctaY }} className="absolute inset-0 z-30 flex items-center justify-center px-6">
            <div className="w-full max-w-sm rounded-2xl border border-zinc-800 bg-zinc-950/90 p-6 text-center shadow-2xl shadow-black/60 backdrop-blur-md">
              <p className="text-xs font-bold uppercase tracking-[0.3em] text-yellow-400">{tr("story.yourTurn")}</p>
              <h2 className="mt-2 font-display text-2xl font-black text-zinc-50">{tr("story.playReal")}</h2>
              <div className="mt-5 space-y-2.5 text-left">
                {[
                  { Icon: Trophy, text: tr("misc.feat1") },
                  { Icon: Swords, text: tr("misc.feat2") },
                  { Icon: TrendingUp, text: tr("misc.feat3") },
                ].map(({ Icon, text }) => (
                  <div key={text} className="flex items-center gap-3 rounded-lg card-premium px-3 py-2.5">
                    <Icon size={16} className="flex-shrink-0 text-yellow-400" />
                    <span className="text-sm font-medium text-zinc-200">{text}</span>
                  </div>
                ))}
              </div>
              <Link
                href={user ? "/" : "/login"}
                className="volt-grad volt-shadow mt-5 block rounded-xl py-3 text-sm font-black text-zinc-950 transition-transform active:scale-95"
              >
                {user ? tr("misc.ctaHome") : tr("misc.ctaStart")}
              </Link>
              <Link href="/leagues" className="mt-2 block py-2 text-xs font-semibold text-zinc-400 transition-colors hover:text-zinc-200">
                {tr("story.seeLeagues")}
              </Link>
              {/* Обязательная атрибуция свободной лицензии фото */}
              <p className="mt-2 text-[9px] leading-relaxed text-zinc-600">
                Фото: Анна Нэсси · Wikimedia Commons · CC BY-SA 3.0
              </p>
            </div>
          </m.div>

        </div>
      </div>
    </div>
  );
}
