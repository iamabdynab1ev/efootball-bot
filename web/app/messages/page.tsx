"use client";

import { Suspense, useMemo } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { ChevronLeft, MessageSquare } from "lucide-react";
import { useAuth } from "@/lib/auth";
import { useChatRoom, useDirectRooms } from "@/lib/chat";
import { ChatThread } from "@/components/ChatThread";
import { PlayerAvatar } from "@/components/PlayerAvatar";

function fmtWhen(iso?: string) {
  if (!iso) return "";
  const d = new Date(iso);
  const today = new Date();
  if (d.toDateString() === today.toDateString()) {
    return d.toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
  }
  return d.toLocaleDateString("ru-RU", { day: "2-digit", month: "2-digit" });
}

// Открытый диалог — полноэкранный оверлей на мобиле (как чат турнира).
function Thread({ roomId, otherName, otherClub }: { roomId: number; otherName: string; otherClub?: string }) {
  const { user } = useAuth();
  const router = useRouter();
  const { messages, hasMore, send, loadOlder } = useChatRoom(roomId);

  const goBack = () => router.push("/messages");

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-zinc-950 lg:static lg:z-auto lg:-my-8 lg:h-[calc(100dvh-2rem)] lg:min-h-[440px]">
      <header className="flex items-center gap-2.5 px-3 py-2.5 flex-shrink-0 border-b border-zinc-800 bg-zinc-950/95 backdrop-blur-sm pt-[max(0.625rem,env(safe-area-inset-top))] lg:border-0 lg:bg-transparent lg:px-0 lg:pt-0">
        <button
          onClick={goBack}
          aria-label="Назад"
          className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100 transition-colors -ml-1"
        >
          <ChevronLeft size={22} />
        </button>
        <PlayerAvatar displayName={otherName || "?"} favoriteClub={otherClub} size={34} />
        <div className="min-w-0">
          <p className="truncate text-sm font-bold text-zinc-100 leading-tight">{otherName || "Диалог"}</p>
          <p className="text-[11px] text-zinc-500 leading-tight">Личные сообщения</p>
        </div>
      </header>

      <div className="flex-1 min-h-0">
        <ChatThread
          messages={messages}
          hasMore={hasMore}
          loadOlder={loadOlder}
          send={send}
          currentUserId={user?.id}
          isAdmin={user?.is_admin}
          showAuthorNames={false}
          resetKey={roomId}
        />
      </div>
    </div>
  );
}

function MessagesInner() {
  const { user } = useAuth();
  const searchParams = useSearchParams();
  const { rooms, loading } = useDirectRooms();
  const roomId = Number(searchParams.get("room")) || null;

  const active = useMemo(() => rooms.find((r) => r.room_id === roomId) ?? null, [rooms, roomId]);

  if (!user) {
    return (
      <div className="py-16 text-center">
        <MessageSquare size={26} className="mx-auto mb-3 text-zinc-600" />
        <p className="text-sm text-zinc-500">Войдите, чтобы читать личные сообщения.</p>
        <Link href="/login" className="mt-4 inline-block rounded-lg bg-yellow-400 px-4 py-2 text-sm font-bold text-zinc-900">Войти</Link>
      </div>
    );
  }

  // Открытый диалог (в т.ч. по deep-link из уведомления).
  if (roomId) {
    return <Thread roomId={roomId} otherName={active?.other_name ?? ""} otherClub={active?.other_club} />;
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-yellow-400/15 text-yellow-400">
          <MessageSquare size={18} />
        </div>
        <div>
          <h1 className="text-lg font-bold text-zinc-100 leading-tight">Сообщения</h1>
          <p className="text-[11px] text-zinc-500">Личные чаты с соперниками</p>
        </div>
      </div>

      {loading ? (
        <div className="py-10 text-center text-sm text-zinc-500">Загрузка…</div>
      ) : rooms.length === 0 ? (
        <div className="rounded-xl border border-zinc-800 bg-zinc-900 px-4 py-12 text-center">
          <MessageSquare size={24} className="mx-auto mb-3 text-zinc-600" />
          <p className="text-sm text-zinc-400 font-medium">Пока нет диалогов</p>
          <p className="mt-1 text-xs text-zinc-500">Откройте матч и нажмите «Написать сопернику», чтобы начать личный чат.</p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-zinc-800 bg-zinc-900 divide-y divide-zinc-800/60">
          {rooms.map((r) => (
            <Link
              key={r.room_id}
              href={`/messages?room=${r.room_id}`}
              className="flex items-center gap-3 px-3.5 py-3 hover:bg-zinc-800/40 transition-colors"
            >
              <PlayerAvatar displayName={r.other_name} favoriteClub={r.other_club} size={44} />
              <div className="min-w-0 flex-1">
                <div className="flex items-center justify-between gap-2">
                  <p className="truncate text-sm font-semibold text-zinc-100">{r.other_name}</p>
                  <span className="flex-shrink-0 text-[10px] text-zinc-500">{fmtWhen(r.last_at)}</span>
                </div>
                <p className="truncate text-xs text-zinc-500 mt-0.5">
                  {r.last_author_id === user.id ? "Вы: " : ""}{r.last_body || "Нет сообщений"}
                </p>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

export default function Page() {
  return (
    <Suspense fallback={<div className="py-10 text-center text-sm text-zinc-500">Загрузка…</div>}>
      <MessagesInner />
    </Suspense>
  );
}
