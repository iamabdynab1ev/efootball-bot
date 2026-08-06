"use client";

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { ChevronLeft, MessageSquare } from "lucide-react";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { ChatPanel } from "@/components/ChatPanel";
import { tr } from "@/lib/i18n";

function ChatPage() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const id = Number(searchParams.get("id"));
  const { user } = useAuth();
  const [title, setTitle] = useState(tr("misc.tournamentChat"));

  useEffect(() => {
    if (!id) return;
    api.get(`/api/leagues/${id}`)
      .then((r) => { if (r.data?.name) setTitle(r.data.name); })
      .catch(() => { /* оставляем дефолтный заголовок */ });
  }, [id]);

  const goBack = () => {
    // Назад — на страницу лиги (или в историю, если пришли оттуда).
    if (typeof window !== "undefined" && window.history.length > 1) router.back();
    else router.push(`/leagues/details?id=${id}`);
  };

  if (!id) {
    return <div className="py-10 text-center text-sm text-zinc-500">{tr("leagueDetail.notFound")}</div>;
  }

  return (
    // Мобайл: полноэкранный оверлей поверх апп-баров и нижней навигации —
    // весь экран занят чатом, как в мессенджере. Десктоп: обычный поток внутри
    // layout (сайдбар остаётся), высота под вьюпорт.
    <div className="fixed inset-0 z-50 flex flex-col bg-zinc-950 lg:static lg:z-auto lg:-my-8 lg:h-[calc(100dvh-2rem)] lg:min-h-[440px]">
      <header className="flex items-center gap-2 px-3 py-2.5 flex-shrink-0 border-b border-white/5 bg-zinc-950/95 backdrop-blur-sm shadow-sm shadow-black/20 pt-[max(0.625rem,env(safe-area-inset-top))] lg:border-0 lg:bg-transparent lg:px-0 lg:pt-0">
        <button
          onClick={goBack}
          aria-label={tr("messages.back")}
          className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100 transition-colors -ml-1"
        >
          <ChevronLeft size={22} />
        </button>
        <div className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-yellow-400/15 text-yellow-400">
          <MessageSquare size={16} />
        </div>
        <div className="min-w-0">
          <p className="truncate text-sm font-bold text-zinc-100 leading-tight">{title}</p>
          <p className="text-[11px] text-zinc-500 leading-tight">{tr("misc.tournamentChat")}</p>
        </div>
      </header>

      <div className="flex-1 min-h-0">
        <ChatPanel leagueId={id} currentUserId={user?.id} isAdmin={user?.is_admin} variant="full" />
      </div>
    </div>
  );
}

export default function Page() {
  return (
    <Suspense fallback={<div className="py-10 text-center text-sm text-zinc-500">{tr("common.loading")}</div>}>
      <ChatPage />
    </Suspense>
  );
}
