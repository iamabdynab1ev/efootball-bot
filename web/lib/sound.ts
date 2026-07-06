"use client";

// Звуковой движок уведомлений. Каждому типу события — свой звук из локальных
// ассетов (/sounds/*.wav, генерятся scripts/gen_sounds.py), предзагруженных при
// старте. Звук включён у всех автоматически — без настроек и тумблеров.
// AudioContext разблокируется первым жестом пользователя (autoplay-политика);
// до разблокировки/загрузки — fallback на WebAudio-синтез, чтобы сигнал не
// потерялся.

export type SoundType = "challenge" | "result" | "message" | "sent" | "system";

const SOUND_TYPES: SoundType[] = ["challenge", "result", "message", "sent", "system"];

// ── Загрузка и воспроизведение ────────────────────────────────────────────────

let ctx: AudioContext | null = null;
const buffers = new Map<SoundType, AudioBuffer>();
const raw = new Map<SoundType, ArrayBuffer>();
let preloadStarted = false;
let unlockInstalled = false;

function ensureCtx(): AudioContext | null {
  try {
    ctx = ctx || new AudioContext();
    return ctx;
  } catch {
    return null;
  }
}

async function decodeAll() {
  const c = ensureCtx();
  if (!c) return;
  for (const [type, buf] of Array.from(raw.entries())) {
    if (buffers.has(type)) continue;
    try {
      // slice: decodeAudioData «съедает» буфер в части браузеров.
      buffers.set(type, await c.decodeAudioData(buf.slice(0)));
    } catch { /* останется синтез-fallback */ }
  }
}

// preloadSounds — качаем ассеты сразу при старте приложения (кэшируются
// браузером), декодируем как только появится AudioContext.
export function preloadSounds() {
  if (typeof window === "undefined" || preloadStarted) return;
  preloadStarted = true;
  for (const type of SOUND_TYPES) {
    fetch(`/sounds/${type}.wav`)
      .then((r) => (r.ok ? r.arrayBuffer() : Promise.reject()))
      .then((b) => { raw.set(type, b); void decodeAll(); })
      .catch(() => { /* синтез-fallback */ });
  }
  installUnlock();
}

// installUnlock — autoplay-политика: создаём/возобновляем AudioContext после
// первого взаимодействия пользователя (клик/тап/клавиша).
function installUnlock() {
  if (unlockInstalled || typeof window === "undefined") return;
  unlockInstalled = true;
  const unlock = () => {
    const c = ensureCtx();
    if (c && c.state === "suspended") void c.resume();
    void decodeAll();
    window.removeEventListener("pointerdown", unlock);
    window.removeEventListener("keydown", unlock);
    window.removeEventListener("touchstart", unlock);
  };
  window.addEventListener("pointerdown", unlock, { passive: true });
  window.addEventListener("keydown", unlock);
  window.addEventListener("touchstart", unlock, { passive: true });
}

// Антиспам: пачка событий одного типа — один звук.
const THROTTLE_MS: Record<SoundType, number> = {
  challenge: 1500,
  result: 1500,
  message: 2500, // чат: не чаще раза в 2.5 секунды
  sent: 300,
  system: 1500,
};
const lastPlay: Partial<Record<SoundType, number>> = {};

// playSound — единая точка воспроизведения (force — без троттлинга/проверок).
export function playSound(type: SoundType, force = false) {
  if (typeof window === "undefined") return;
  if (!force) {
    // В фоне работает системный звук web-push — не дублируем. Отправка ("sent")
    // случается только по действию пользователя, вкладка заведомо видима.
    if (type !== "sent" && document.visibilityState !== "visible") return;
    const now = Date.now();
    if (now - (lastPlay[type] ?? 0) < THROTTLE_MS[type]) return;
    lastPlay[type] = now;
  }
  const c = ensureCtx();
  if (!c) return;
  if (c.state === "suspended") void c.resume();

  const buf = buffers.get(type);
  if (buf) {
    try {
      const src = c.createBufferSource();
      src.buffer = buf;
      src.connect(c.destination);
      src.start();
      return;
    } catch { /* попробуем синтез */ }
  }
  synthFallback(c, type);
}

// synthFallback — если ассет ещё не загрузился: узнаваемый минимальный синтез.
function synthFallback(c: AudioContext, type: SoundType) {
  try {
    const t0 = c.currentTime;
    const note = (freq: number, start: number, dur: number, peak: number) => {
      const o = c.createOscillator();
      const g = c.createGain();
      o.type = "sine";
      o.frequency.value = freq;
      g.gain.setValueAtTime(0.0001, t0 + start);
      g.gain.exponentialRampToValueAtTime(peak, t0 + start + 0.012);
      g.gain.exponentialRampToValueAtTime(0.0001, t0 + start + dur);
      o.connect(g).connect(c.destination);
      o.start(t0 + start);
      o.stop(t0 + start + dur + 0.02);
    };
    switch (type) {
      case "challenge":
        note(880, 0, 0.14, 0.09); note(1108.73, 0.1, 0.14, 0.09); note(1318.51, 0.2, 0.24, 0.1);
        break;
      case "result":
        note(783.99, 0, 0.16, 0.08); note(1046.5, 0.12, 0.3, 0.09);
        break;
      case "sent":
        note(1800, 0, 0.08, 0.03);
        break;
      case "message":
      case "system":
      default:
        note(987.77, 0, 0.16, 0.07); note(1318.51, 0.09, 0.24, 0.055);
    }
  } catch { /* аудио недоступно — молчим */ }
}
