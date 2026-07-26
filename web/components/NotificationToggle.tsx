"use client";

import { useEffect, useState } from "react";
import { Bell, BellRing, BellOff } from "lucide-react";
import { toast } from "sonner";
import { pushTest } from "@/lib/api";
import { disablePush, enablePush, isPushEnabled, pushSupport } from "@/lib/push";
import { useLang } from "@/lib/i18n";

type State = "loading" | "unsupported" | "ios-install" | "off" | "on" | "busy";

export function NotificationToggle() {
  const { t } = useLang();
  const [state, setState] = useState<State>("loading");

  useEffect(() => {
    const support = pushSupport();
    if (support !== "ok") {
      setState(support === "ios-install" ? "ios-install" : "unsupported");
      return;
    }
    isPushEnabled().then((on) => setState(on ? "on" : "off")).catch(() => setState("unsupported"));
  }, []);

  async function enable() {
    setState("busy");
    const res = await enablePush();
    if (res === true) {
      setState("on");
      toast.success(t("push.enabled"));
    } else {
      toast.error(res === "denied" ? t("push.denied") : t("push.error"));
      setState("off");
    }
  }

  async function disable() {
    setState("busy");
    try {
      await disablePush();
      setState("off");
      toast.success(t("push.disabled"));
    } catch {
      setState("off");
    }
  }

  if (state === "loading" || state === "unsupported") return null;

  // iPhone: push требует установки на главный экран
  if (state === "ios-install") {
    return (
      <div className="rounded-xl card-premium p-4">
        <div className="flex items-center gap-3">
          <div className="flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-xl bg-zinc-800 text-zinc-400">
            <Bell size={20} />
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-bold text-zinc-100">{t("push.title")}</p>
            <p className="text-xs text-amber-400/90 mt-0.5">{t("push.iosInstall")}</p>
          </div>
        </div>
      </div>
    );
  }

  const on = state === "on";

  return (
    <div className="rounded-xl card-premium p-4">
      <div className="flex items-center gap-3">
        <div className={`flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-xl ${on ? "bg-green-500/15 text-green-400" : "bg-zinc-800 text-zinc-400"}`}>
          {on ? <BellRing size={20} /> : <Bell size={20} />}
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-bold text-zinc-100">{t("push.title")}</p>
          <p className="text-xs text-zinc-400 mt-0.5">{on ? t("push.onDesc") : t("push.offDesc")}</p>
        </div>
        <button
          onClick={on ? disable : enable}
          disabled={state === "busy"}
          className={`flex-shrink-0 rounded-lg px-3 py-2 text-sm font-bold transition-opacity hover:opacity-90 disabled:opacity-50 ${
            on ? "bg-zinc-800 text-zinc-300 border border-zinc-700" : "bg-yellow-400 text-zinc-950"
          }`}
        >
          {on ? <BellOff size={15} /> : t("push.enable")}
        </button>
      </div>
      {on && (
        <button
          onClick={async () => { try { await pushTest(); toast.success(t("push.testSent")); } catch { toast.error(t("push.error")); } }}
          className="mt-3 w-full rounded-lg border border-zinc-700 py-2 text-xs font-semibold text-zinc-300 hover:bg-zinc-800 transition-colors"
        >
          {t("push.testButton")}
        </button>
      )}
    </div>
  );
}
