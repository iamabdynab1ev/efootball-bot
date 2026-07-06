"use client";

// Звук уведомления — мягкий двухтоновый «динь», синтезируется WebAudio
// (без аудиофайлов и лишних запросов). Играет только когда вкладка видима:
// в фоне работает системный звук web-push, дублировать его не нужно.

let ctx: AudioContext | null = null;
let lastPlay = 0;

const KEY = "notif_sound";

export function notifSoundEnabled(): boolean {
  if (typeof window === "undefined") return false;
  return localStorage.getItem(KEY) !== "0"; // по умолчанию включён
}

export function setNotifSoundEnabled(on: boolean) {
  localStorage.setItem(KEY, on ? "1" : "0");
}

// force — для кнопки «Проверить» в настройках (играет даже при выключенном
// тумблере и без троттлинга; заодно «разблокирует» AudioContext жестом).
export function playNotifySound(force = false) {
  if (typeof window === "undefined") return;
  if (!force) {
    if (!notifSoundEnabled()) return;
    if (document.visibilityState !== "visible") return;
    const now = Date.now();
    if (now - lastPlay < 1500) return; // пачка событий — один звук
    lastPlay = now;
  }
  try {
    ctx = ctx || new AudioContext();
    if (ctx.state === "suspended") void ctx.resume();
    const t0 = ctx.currentTime;
    const note = (freq: number, start: number, dur: number, peak: number) => {
      const o = ctx!.createOscillator();
      const g = ctx!.createGain();
      o.type = "sine";
      o.frequency.value = freq;
      g.gain.setValueAtTime(0.0001, t0 + start);
      g.gain.exponentialRampToValueAtTime(peak, t0 + start + 0.012);
      g.gain.exponentialRampToValueAtTime(0.0001, t0 + start + dur);
      o.connect(g).connect(ctx!.destination);
      o.start(t0 + start);
      o.stop(t0 + start + dur + 0.02);
    };
    note(987.77, 0, 0.16, 0.07);      // си (B5)
    note(1318.51, 0.09, 0.24, 0.055); // ми (E6)
  } catch { /* аудио недоступно — молчим */ }
}
