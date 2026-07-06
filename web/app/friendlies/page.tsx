"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { Swords, Check, X, Clock, Trophy } from "lucide-react";
import { toast } from "sonner";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useSSE } from "@/lib/sse";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { cn } from "@/lib/utils";

// Товарищеские матчи: вызовы, активные игры, внесение и подтверждение счёта.
// Подтверждённый результат влияет на ELO-рейтинг обоих игроков.

interface Friendly {
  id: number;
  challenger_id: number;
  opponent_id: number;
  status: string;
  challenger_goals?: number;
  opponent_goals?: number;
  claimed_by?: number;
  created_at: string;
  challenger_name: string;
  challenger_club?: string;
  opponent_name: string;
  opponent_club?: string;
}

function ScoreForm({ f, meChallenger, onDone }: { f: Friendly; meChallenger: boolean; onDone: () => void }) {
  const [my, setMy] = useState("");
  const [opp, setOpp] = useState("");
  const [busy, setBusy] = useState(false);
  const submit = () => {
    const m = Number(my), o = Number(opp);
    if (!Number.isInteger(m) || !Number.isInteger(o) || m < 0 || o < 0 || m > 50 || o > 50) {
      toast.error("Введите корректный счёт");
      return;
    }
    setBusy(true);
    api.post(`/api/friendlies/${f.id}/score`, { my_goals: m, opp_goals: o })
      .then(() => { toast.success("Счёт отправлен — ждём подтверждения соперника"); onDone(); })
      .catch(() => toast.error("Не удалось отправить счёт"))
      .finally(() => setBusy(false));
  };
  return (
    <div className="mt-3 flex items-center gap-2">
      <input value={my} onChange={(e) => setMy(e.target.value.replace(/\D/g, "").slice(0, 2))} inputMode="numeric"
        placeholder="Я" className="h-10 w-14 rounded-lg border border-zinc-700 bg-zinc-950 text-center text-base font-black text-zinc-100 focus:border-yellow-400/70 focus:outline-none" />
      <span className="text-zinc-500">:</span>
      <input value={opp} onChange={(e) => setOpp(e.target.value.replace(/\D/g, "").slice(0, 2))} inputMode="numeric"
        placeholder="Он" className="h-10 w-14 rounded-lg border border-zinc-700 bg-zinc-950 text-center text-base font-black text-zinc-100 focus:border-yellow-400/70 focus:outline-none" />
      <button onClick={submit} disabled={busy || my === "" || opp === ""}
        className="ml-1 rounded-lg volt-grad px-4 py-2.5 text-xs font-bold text-zinc-950 volt-shadow transition-transform active:scale-95 disabled:opacity-50">
        Отправить счёт
      </button>
    </div>
  );
}

export default function FriendliesPage() {
  const { user } = useAuth();
  const [list, setList] = useState<Friendly[] | null>(null);

  const reload = useCallback(() => {
    api.get("/api/friendlies")
      .then((r) => setList(r.data.friendlies ?? []))
      .catch(() => setList((p) => p ?? []));
  }, []);

  useEffect(() => { if (user) reload(); }, [user, reload]);
  useSSE("notification", () => reload(), !!user); // вызов/счёт/подтверждение — живое обновление

  const act = (id: number, action: string) => {
    api.post(`/api/friendlies/${id}/${action}`, {})
      .then(() => reload())
      .catch((e) => toast.error(e?.response?.data?.error ?? "Не получилось — обновите страницу"));
  };

  if (!user) {
    return (
      <div className="py-16 text-center">
        <Swords size={26} className="mx-auto mb-3 text-zinc-600" />
        <p className="text-sm text-zinc-500">Войдите, чтобы вызывать друзей на товарищеские матчи.</p>
        <Link href="/login" className="mt-4 inline-block rounded-lg bg-yellow-400 px-4 py-2 text-sm font-bold text-zinc-900">Войти</Link>
      </div>
    );
  }

  const me = user.id;
  const incoming = (list ?? []).filter((f) => f.status === "pending" && f.opponent_id === me);
  const outgoing = (list ?? []).filter((f) => f.status === "pending" && f.challenger_id === me);
  const active = (list ?? []).filter((f) => f.status === "accepted" || f.status === "score_claimed");
  const history = (list ?? []).filter((f) => f.status === "confirmed" || f.status === "declined" || f.status === "expired").slice(0, 15);

  const other = (f: Friendly) => f.challenger_id === me
    ? { name: f.opponent_name, club: f.opponent_club }
    : { name: f.challenger_name, club: f.challenger_club };

  const Card = ({ f, children }: { f: Friendly; children?: React.ReactNode }) => {
    const o = other(f);
    return (
      <div className="rounded-xl border border-zinc-800 bg-zinc-900 px-4 py-3">
        <div className="flex items-center gap-3">
          <PlayerAvatar displayName={o.name} favoriteClub={o.club} size={36} />
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-semibold text-zinc-100">{o.name}</p>
            <p className="text-[11px] text-zinc-500">{new Date(f.created_at).toLocaleDateString("ru-RU", { day: "numeric", month: "short" })}</p>
          </div>
        </div>
        {children}
      </div>
    );
  };

  const section = (title: string, items: JSX.Element[]) => items.length > 0 && (
    <section className="space-y-2">
      <h2 className="text-xs font-bold uppercase tracking-wider text-zinc-400">{title}</h2>
      {items}
    </section>
  );

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div>
        <p className="mb-1 text-xs font-semibold uppercase tracking-widest text-zinc-500">Без турнира · влияет на рейтинг</p>
        <h1 className="flex items-center gap-2 font-display text-2xl font-bold text-zinc-100">
          <Swords size={22} className="text-yellow-400" /> Товарищеские матчи
        </h1>
        <p className="mt-1 text-sm text-zinc-500">
          Вызови любого игрока со страницы его профиля (раздел «Игроки» → кнопка «Вызвать на матч»).
        </p>
      </div>

      {list === null ? (
        <div className="space-y-2" aria-hidden>
          {[0, 1, 2].map((i) => <div key={i} className="skeleton h-16 rounded-xl" />)}
        </div>
      ) : list.length === 0 ? (
        <div className="rounded-xl border border-zinc-800 bg-zinc-900 px-4 py-12 text-center">
          <Swords size={24} className="mx-auto mb-3 text-zinc-600" />
          <p className="text-sm font-medium text-zinc-400">Пока нет товарищеских матчей</p>
          <p className="mt-1 text-xs text-zinc-500">Откройте профиль игрока и бросьте вызов ⚔️</p>
          <Link href="/players" className="mt-4 inline-block rounded-lg bg-yellow-400 px-4 py-2 text-xs font-bold text-zinc-900">К игрокам</Link>
        </div>
      ) : (
        <>
          {section("⚔️ Вам бросили вызов", incoming.map((f) => (
            <Card key={f.id} f={f}>
              <div className="mt-3 flex gap-2">
                <button onClick={() => act(f.id, "accept")} className="flex flex-1 items-center justify-center gap-1.5 rounded-lg volt-grad py-2.5 text-xs font-bold text-zinc-950 active:scale-95 transition-transform">
                  <Check size={14} /> Принять
                </button>
                <button onClick={() => act(f.id, "decline")} className="flex flex-1 items-center justify-center gap-1.5 rounded-lg border border-zinc-700 py-2.5 text-xs font-semibold text-zinc-400 hover:text-red-400 hover:border-red-500/40 transition-colors">
                  <X size={14} /> Отклонить
                </button>
              </div>
            </Card>
          )))}

          {section("📤 Ваши вызовы", outgoing.map((f) => (
            <Card key={f.id} f={f}>
              <div className="mt-2 flex items-center justify-between">
                <span className="flex items-center gap-1.5 text-[11px] text-zinc-500"><Clock size={12} /> Ждём ответа…</span>
                <button onClick={() => act(f.id, "cancel")} className="text-[11px] font-semibold text-zinc-500 hover:text-red-400 transition-colors">Отменить</button>
              </div>
            </Card>
          )))}

          {section("🎮 Активные — играйте и вносите счёт", active.map((f) => {
            const meChallenger = f.challenger_id === me;
            const claimedByMe = f.claimed_by === me;
            return (
              <Card key={f.id} f={f}>
                {f.status === "accepted" && (
                  <>
                    <p className="mt-2 text-[11px] text-zinc-500">Сыграйте в eFootball, затем внесите счёт (любой из вас).</p>
                    <ScoreForm f={f} meChallenger={meChallenger} onDone={reload} />
                  </>
                )}
                {f.status === "score_claimed" && (
                  <div className="mt-3">
                    <p className="text-center text-xl font-black tabular-nums text-zinc-100">
                      {meChallenger ? `${f.challenger_goals}:${f.opponent_goals}` : `${f.opponent_goals}:${f.challenger_goals}`}
                    </p>
                    {claimedByMe ? (
                      <p className="mt-1 text-center text-[11px] text-zinc-500">Ждём подтверждения соперника… Без ответа за 48 ч матч истечёт, и вызов снова станет доступен.</p>
                    ) : (
                      <div className="mt-2 flex gap-2">
                        <button onClick={() => act(f.id, "confirm")} className="flex flex-1 items-center justify-center gap-1.5 rounded-lg volt-grad py-2.5 text-xs font-bold text-zinc-950 active:scale-95 transition-transform">
                          <Check size={14} /> Подтвердить
                        </button>
                        <button onClick={() => act(f.id, "reject-score")} className="flex flex-1 items-center justify-center gap-1.5 rounded-lg border border-zinc-700 py-2.5 text-xs font-semibold text-zinc-400 hover:text-red-400 transition-colors">
                          <X size={14} /> Оспорить
                        </button>
                      </div>
                    )}
                  </div>
                )}
              </Card>
            );
          }))}

          {section("📜 История", history.map((f) => {
            const meChallenger = f.challenger_id === me;
            const my = meChallenger ? f.challenger_goals : f.opponent_goals;
            const op = meChallenger ? f.opponent_goals : f.challenger_goals;
            const won = f.status === "confirmed" && (my ?? 0) > (op ?? 0);
            const lost = f.status === "confirmed" && (my ?? 0) < (op ?? 0);
            return (
              <Card key={f.id} f={f}>
                <div className="mt-2 flex items-center justify-between">
                  {f.status === "declined" ? (
                    <span className="text-[11px] text-zinc-500">Вызов отклонён</span>
                  ) : f.status === "expired" ? (
                    <span className="text-[11px] text-zinc-500">⌛ Матч истёк — можно вызвать заново</span>
                  ) : (
                    <>
                      <span className={cn("text-lg font-black tabular-nums", won ? "text-green-400" : lost ? "text-red-400" : "text-zinc-300")}>
                        {my}:{op}
                      </span>
                      <span className="flex items-center gap-1 text-[11px] text-zinc-500"><Trophy size={12} className="text-yellow-500" /> рейтинг обновлён</span>
                    </>
                  )}
                </div>
              </Card>
            );
          }))}
        </>
      )}
    </div>
  );
}
