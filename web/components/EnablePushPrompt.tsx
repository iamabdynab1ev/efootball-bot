"use client";

import { useEffect, useState } from "react";
import { BellRing, X } from "lucide-react";
import { toast } from "sonner";
import { useAuth } from "@/lib/auth";
import { enablePush, isPushEnabled, permissionDenied, pushSupport } from "@/lib/push";

// Авто-запрос при входе: если пользователь залогинен, push поддерживается, но
// ещё не включён и не отклонён на уровне браузера — предлагаем включить в один
// тап. «Позже» прячет до следующего входа (флаг в sessionStorage — не назойливо
// в рамках сессии, но при новом входе спросим снова).
const DISMISS_KEY = "push_prompt_dismissed";

export function EnablePushPrompt() {
  const { user } = useAuth();
  const [show, setShow] = useState(false);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!user) { setShow(false); return; }
    if (sessionStorage.getItem(DISMISS_KEY)) return;
    if (pushSupport() !== "ok" || permissionDenied()) return;

    let on = true;
    // Небольшая задержка — не бросаем запрос в лицо сразу на логине.
    const t = setTimeout(async () => {
      const enabled = await isPushEnabled();
      if (on && !enabled) setShow(true);
    }, 1500);
    return () => { on = false; clearTimeout(t); };
  }, [user]);

  if (!show) return null;

  const dismiss = () => {
    sessionStorage.setItem(DISMISS_KEY, "1");
    setShow(false);
  };

  const enable = async () => {
    setBusy(true);
    const res = await enablePush();
    setBusy(false);
    if (res === true) {
      toast.success("Уведомления включены");
      setShow(false);
    } else if (res === "denied") {
      toast.error("Разрешение отклонено — можно включить позже в настройках браузера");
      dismiss();
    } else {
      toast.error("Не удалось включить уведомления");
    }
  };

  return (
    <div className="fixed inset-x-3 bottom-[calc(76px+env(safe-area-inset-bottom))] z-50 lg:inset-x-auto lg:right-6 lg:bottom-6 lg:w-[380px]">
      <div className="rounded-2xl border border-zinc-700 bg-zinc-900 p-4 shadow-2xl shadow-black/50">
        <div className="flex items-start gap-3">
          <div className="flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-xl bg-yellow-400/15 text-yellow-400">
            <BellRing size={20} />
          </div>
          <div className="min-w-0 flex-1">
            <p className="text-sm font-bold text-zinc-100">Включить уведомления?</p>
            <p className="mt-0.5 text-xs text-zinc-400">
              Матчи, подтверждения, споры и упоминания в чате будут приходить на телефон — даже когда приложение закрыто.
            </p>
          </div>
          <button onClick={dismiss} aria-label="Закрыть" className="flex-shrink-0 rounded-md p-1 text-zinc-500 hover:text-zinc-300">
            <X size={16} />
          </button>
        </div>
        <div className="mt-3 flex gap-2">
          <button
            onClick={enable}
            disabled={busy}
            className="flex-1 rounded-lg bg-yellow-400 py-2 text-sm font-bold text-zinc-950 disabled:opacity-50 hover:opacity-90 transition-opacity"
          >
            {busy ? "Включаю…" : "Включить"}
          </button>
          <button
            onClick={dismiss}
            className="rounded-lg border border-zinc-700 px-4 py-2 text-sm font-semibold text-zinc-300 hover:bg-zinc-800 transition-colors"
          >
            Позже
          </button>
        </div>
      </div>
    </div>
  );
}
