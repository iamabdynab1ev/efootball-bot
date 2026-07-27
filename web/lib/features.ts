"use client";

import { useEffect, useState } from "react";
import { api } from "./api";

// Флаги каналов с сервера: какие интеграции включены на этом развёртывании.
// Например, TELEGRAM_ENABLED=0 — весь Telegram-UI прячется, канал — WhatsApp.

export interface Features {
  telegram: boolean;
}

let cached: Features | null = null;
let inflight: Promise<Features> | null = null;

async function load(): Promise<Features> {
  if (cached) return cached;
  inflight = inflight ?? api.get("/api/features")
    .then((r) => (cached = { telegram: r.data?.telegram !== false }))
    .catch(() => ({ telegram: true })); // сеть упала — не прячем функции зря
  return inflight;
}

export function useFeatures(): Features {
  const [f, setF] = useState<Features>(cached ?? { telegram: true });
  useEffect(() => {
    let on = true;
    void load().then((v) => { if (on) setF(v); });
    return () => { on = false; };
  }, []);
  return f;
}
