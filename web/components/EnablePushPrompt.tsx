"use client";

import { useEffect, useState } from "react";
import { BellRing, X } from "lucide-react";
import { toast } from "sonner";
import { useAuth } from "@/lib/auth";
import { enablePush, isPushEnabled, permissionDenied, pushSupport } from "@/lib/push";

// Авто-запрос при входе: если пользователь залогинен (даже если давно
// зарегистрирован), push поддерживается, но ещё НЕ включён и не отклонён на
// уровне браузера — предлагаем включить в один тап. Напоминаем при каждом входе;
// «Позже» откладывает всего на 12 часов (в localStorage), поэтому в следующий
// вход зарегистрированному, но не подписанному, снова напомним.
const SNOOZE_KEY = "push_prompt_snooze_until";
const SNOOZE_MS = 12 * 60 * 60 * 1000; // 12 часов

export function EnablePushPrompt() {
  const { user } = useAuth();
  const [show, setShow] = useState(false);
  const [busy, setBusy] = useState(false);
  // "push" — обычный запрос включить пуш; "tg" — пуш недоступен/отклонён и
  // Telegram не привязан: вне приложения человеку нечем доставить уведомление.
  const [mode, setMode] = useState<"push" | "tg">("push");

  useEffect(() => {
    if (!user) { setShow(false); return; }
    // Отложено недавно — не назойливничаем при быстрых перезагрузках.
    const snoozeUntil = Number(localStorage.getItem(SNOOZE_KEY) || 0);
    if (Date.now() < snoozeUntil) return;

    let on = true;
    if (pushSupport() !== "ok" || permissionDenied()) {
      // Пуш в этом браузере не работает (iPhone без установки на «Домой»,
      // отказ в разрешении, старый браузер). Без Telegram уведомления вне
      // приложения не придут вообще — предлагаем привязку.
      if (user.has_telegram) return;
      const t = setTimeout(() => { if (on) { setMode("tg"); setShow(true); } }, 1500);
      return () => { on = false; clearTimeout(t); };
    }

    // Небольшая задержка — не бросаем запрос в лицо сразу на входе.
    const t = setTimeout(async () => {
      const enabled = await isPushEnabled();
      if (on && !enabled) { setMode("push"); setShow(true); }
    }, 1500);
    return () => { on = false; clearTimeout(t); };
  }, [user]);

  if (!show) return null;

  // «Позже» — откладываем всего на 12 ч, чтобы при следующем входе напомнить снова.
  const dismiss = () => {
    localStorage.setItem(SNOOZE_KEY, String(Date.now() + SNOOZE_MS));
    setShow(false);
  };

  const enable = async () => {
    setBusy(true);
    const res = await enablePush();
    setBusy(false);
    if (res === true) {
      localStorage.removeItem(SNOOZE_KEY);
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
            <p className="text-sm font-bold text-zinc-100">
              {mode === "tg" ? "Уведомления вне приложения не приходят" : "Включить уведомления?"}
            </p>
            <p className="mt-0.5 text-xs text-zinc-400">
              {mode === "tg"
                ? "Пуш в этом браузере недоступен или отключён. Привяжите Telegram — вызовы на матч и сообщения будут приходить туда, даже когда приложение закрыто."
                : "Матчи, подтверждения, споры и упоминания в чате будут приходить на телефон — даже когда приложение закрыто."}
            </p>
          </div>
          <button onClick={dismiss} aria-label="Закрыть" className="flex-shrink-0 rounded-md p-1 text-zinc-500 hover:text-zinc-300">
            <X size={16} />
          </button>
        </div>
        <div className="mt-3 flex gap-2">
          {mode === "tg" ? (
            <a
              href="/settings"
              onClick={() => setShow(false)}
              className="flex-1 rounded-lg bg-yellow-400 py-2 text-center text-sm font-bold text-zinc-950 hover:opacity-90 transition-opacity"
            >
              Привязать Telegram
            </a>
          ) : (
            <button
              onClick={enable}
              disabled={busy}
              className="flex-1 rounded-lg bg-yellow-400 py-2 text-sm font-bold text-zinc-950 disabled:opacity-50 hover:opacity-90 transition-opacity"
            >
              {busy ? "Включаю…" : "Включить"}
            </button>
          )}
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
