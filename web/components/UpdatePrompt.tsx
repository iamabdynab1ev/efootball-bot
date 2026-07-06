"use client";

import { useEffect, useState } from "react";
import { RefreshCw, Sparkles } from "lucide-react";

// UpdatePrompt — «у нас вышло обновление»: клиент периодически сравнивает
// свою версию сборки (зашита в бандл) с /version.json на сервере. Если на
// сервере новее — показываем плашку; «Обновить» чистит кэши (иконки, статика,
// service worker) и перезагружает страницу. Так у всех игроков всегда свежая
// версия — включая новый логотип — без ручной чистки кэша.
export function UpdatePrompt() {
  const [stale, setStale] = useState(false);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    const mine = process.env.NEXT_PUBLIC_BUILD_TS;
    if (!mine || process.env.NODE_ENV !== "production") return;
    let on = true;
    const check = () => {
      fetch(`/version.json?t=${Date.now()}`, { cache: "no-store" })
        .then((r) => r.json())
        .then((d) => { if (on && d?.v && d.v !== mine) setStale(true); })
        .catch(() => { /* оффлайн/сеть — проверим в следующий раз */ });
    };
    check();
    const t = setInterval(check, 10 * 60 * 1000); // раз в 10 минут
    const onVis = () => { if (document.visibilityState === "visible") check(); };
    document.addEventListener("visibilitychange", onVis);
    return () => { on = false; clearInterval(t); document.removeEventListener("visibilitychange", onVis); };
  }, []);

  const update = async () => {
    setBusy(true);
    try {
      // Обновляем service worker и сносим все кэши (статика, иконки, логотип).
      const regs = (await navigator.serviceWorker?.getRegistrations?.()) ?? [];
      await Promise.all(regs.map((r) => r.update().catch(() => {})));
      if (typeof caches !== "undefined") {
        const keys = await caches.keys();
        await Promise.all(keys.map((k) => caches.delete(k)));
      }
    } catch { /* всё равно перезагружаемся */ }
    // Cache-busting параметр гарантирует свежий index.html.
    window.location.href = `${window.location.pathname}?updated=${Date.now()}`;
  };

  if (!stale) return null;

  return (
    <div className="fixed inset-x-0 bottom-[calc(5.5rem+env(safe-area-inset-bottom))] z-[80] flex justify-center px-4 lg:bottom-6">
      <div className="msg-in flex w-full max-w-md items-center gap-3 rounded-2xl border border-yellow-400/30 bg-zinc-900/95 px-4 py-3 shadow-2xl shadow-black/50 backdrop-blur">
        <span className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full bg-yellow-400/15 text-yellow-400">
          <Sparkles size={17} />
        </span>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-bold text-zinc-100">Приложение обновлено</p>
          <p className="text-[11px] text-zinc-500">Нажмите, чтобы получить новую версию</p>
        </div>
        <button
          onClick={update}
          disabled={busy}
          className="flex flex-shrink-0 items-center gap-1.5 rounded-full volt-grad px-4 py-2 text-[12px] font-bold text-zinc-950 volt-shadow transition-transform active:scale-95 disabled:opacity-60"
        >
          <RefreshCw size={13} className={busy ? "animate-spin" : ""} />
          {busy ? "Обновляем…" : "Обновить"}
        </button>
      </div>
    </div>
  );
}
