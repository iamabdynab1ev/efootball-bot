"use client";

import { Suspense, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { Swords } from "lucide-react";
import { toast } from "sonner";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { BrandLogo } from "@/components/BrandLogo";
import { api, fetchPlayerProfile } from "@/lib/api";
import { useAuth } from "@/lib/auth";

// Реферальный вызов: игрок делится ссылкой /challenge?user=ID в любом чате —
// друг открывает её, видит «тебя вызвали на матч» и принимает в один тап.
// Гость сначала логинится (метка pending_challenge), после входа главная
// возвращает его сюда — вызов не теряется.

function ChallengeContent() {
  const params = useSearchParams();
  const router = useRouter();
  const { user, loading } = useAuth();
  const id = Number(params.get("user"));

  const { data: p, isLoading } = useQuery({
    queryKey: ["player-profile", id],
    queryFn: () => fetchPlayerProfile(id),
    enabled: !!id,
    retry: false,
  });

  const [busy, setBusy] = useState(false);
  const accept = async () => {
    if (!user) {
      // Запоминаем вызов и ведём логиниться — главная вернёт сюда.
      try { localStorage.setItem("pending_challenge", String(id)); } catch { /* private mode */ }
      router.push("/login");
      return;
    }
    if (busy) return;
    setBusy(true);
    try {
      await api.post("/api/friendlies", { opponent_id: id });
      toast.success("Вызов брошен — ждём ответа ⚔️");
      router.replace("/friendlies");
    } catch (e: any) {
      toast.error(e?.response?.data?.error ?? "Не получилось — попробуйте ещё раз");
      setBusy(false);
    }
  };

  if (!id || (!isLoading && !p)) {
    return (
      <div className="py-16 text-center">
        <Swords size={26} className="mx-auto mb-3 text-zinc-600" />
        <p className="text-sm text-zinc-500">Ссылка на вызов устарела или неверна.</p>
        <Link href="/players" className="mt-4 inline-block rounded-lg bg-yellow-400 px-4 py-2 text-sm font-bold text-zinc-900">К игрокам</Link>
      </div>
    );
  }

  const isSelf = user?.id === id;

  return (
    <div className="mx-auto flex min-h-[70vh] max-w-sm flex-col items-center justify-center px-4 text-center">
      <BrandLogo size={56} />
      <p className="mt-6 text-xs font-bold uppercase tracking-[0.3em] text-yellow-400">Товарищеский матч</p>

      <div className="mt-5 w-full rounded-2xl card-premium p-6">
        {p ? (
          <>
            <div className="flex justify-center"><PlayerAvatar displayName={p.display_name} favoriteClub={p.favorite_club} size={72} /></div>
            <h1 className="mt-4 font-display text-2xl font-black text-zinc-50">{p.display_name}</h1>
            <p className="mt-1 text-sm text-zinc-400">{p.rating} ELO · {p.rank}</p>
            <p className="mt-4 text-sm font-semibold text-zinc-200">
              {isSelf ? "Это твоя собственная ссылка-вызов — поделись ею с другом!" : "вызывает тебя на матч в eFootball"}
            </p>
            {!isSelf && (
              <button
                onClick={accept}
                disabled={busy || loading}
                className="volt-grad volt-shadow mt-5 flex w-full items-center justify-center gap-2 rounded-xl py-3 text-sm font-black text-zinc-950 transition-transform active:scale-95 disabled:opacity-60"
              >
                <Swords size={16} />
                {busy ? "Отправляем…" : user ? "Принять вызов" : "Войти и принять вызов"}
              </button>
            )}
            <p className="mt-3 text-[11px] text-zinc-500">Результат матча повлияет на ELO-рейтинг обоих игроков</p>
          </>
        ) : (
          <div className="space-y-3" aria-hidden>
            <div className="skeleton mx-auto h-[72px] w-[72px] rounded-full" />
            <div className="skeleton mx-auto h-6 w-40 rounded" />
            <div className="skeleton mx-auto h-10 w-full rounded-xl" />
          </div>
        )}
      </div>

      <Link href="/" className="mt-4 py-2 text-xs font-semibold text-zinc-500 transition-colors hover:text-zinc-300">
        Что такое eFootLeague? →
      </Link>
    </div>
  );
}

export default function ChallengePage() {
  return (
    <Suspense fallback={<div className="py-10 text-center text-sm text-zinc-500">Загрузка…</div>}>
      <ChallengeContent />
    </Suspense>
  );
}
