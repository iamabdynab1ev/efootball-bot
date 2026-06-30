"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { Send, Trash2, MessageSquare, Archive } from "lucide-react";
import { api } from "@/lib/api";
import { useChatRoom, useChatRooms, type ChatMessage } from "@/lib/chat";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { cn } from "@/lib/utils";

interface Props {
  leagueId: number;
  currentUserId?: number;
  isAdmin?: boolean;
}

interface Member { user_id: number; display_name: string; favorite_club?: string }

function fmtTime(iso: string) {
  return new Date(iso).toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
}

// Подсветка @упоминаний в тексте сообщения.
function renderBody(body: string) {
  const parts = body.split(/(@[\wА-Яа-яЁё]+)/g);
  return parts.map((p, i) =>
    p.startsWith("@")
      ? <span key={i} className="font-semibold text-yellow-400">{p}</span>
      : <span key={i}>{p}</span>
  );
}

export function ChatPanel({ leagueId, currentUserId, isAdmin }: Props) {
  const { rooms, loading: roomsLoading } = useChatRooms(leagueId);
  const [roomId, setRoomId] = useState<number | null>(null);
  const room = useMemo(() => rooms.find((r) => r.id === roomId) ?? null, [rooms, roomId]);

  // По умолчанию — первая доступная комната (обычно «Общий чат»).
  useEffect(() => {
    if (roomId == null && rooms.length > 0) setRoomId(rooms[0].id);
  }, [rooms, roomId]);

  const { messages, hasMore, send, loadOlder } = useChatRoom(roomId);

  // Кандидаты для @упоминаний — имена участников лиги (из таблицы).
  const [members, setMembers] = useState<Member[]>([]);
  useEffect(() => {
    api.get(`/api/leagues/${leagueId}/standings`)
      .then((r) => setMembers((r.data.standings ?? []).map((s: any) => ({
        user_id: s.user_id, display_name: s.display_name, favorite_club: s.favorite_club,
      }))))
      .catch(() => setMembers([]));
  }, [leagueId]);

  const [text, setText] = useState("");
  const [sending, setSending] = useState(false);
  const [mentionQuery, setMentionQuery] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Авто-скролл вниз при новом сообщении (но не при подгрузке старых сверху).
  const bottomRef = useRef<HTMLDivElement>(null);
  const lastIdRef = useRef(0);
  useEffect(() => {
    const lastId = messages.length ? messages[messages.length - 1].id : 0;
    if (lastId !== lastIdRef.current) {
      lastIdRef.current = lastId;
      bottomRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages]);

  const onChange = (v: string) => {
    setText(v);
    const m = v.match(/@([\wА-Яа-яЁё]*)$/); // токен после @ в конце ввода
    setMentionQuery(m ? m[1].toLowerCase() : null);
  };

  const mentionMatches = useMemo(() => {
    if (mentionQuery == null) return [];
    return members
      .filter((m) => m.display_name.toLowerCase().includes(mentionQuery))
      .slice(0, 6);
  }, [mentionQuery, members]);

  const applyMention = (name: string) => {
    const first = name.split(" ")[0];
    setText((t) => t.replace(/@([\wА-Яа-яЁё]*)$/, `@${first} `));
    setMentionQuery(null);
    inputRef.current?.focus();
  };

  const submit = async () => {
    const body = text.trim();
    if (!body || sending || !room || room.archived) return;
    setSending(true);
    try {
      await send(body);
      setText("");
      setMentionQuery(null);
    } finally {
      setSending(false);
    }
  };

  const onDelete = async (m: ChatMessage) => {
    try { await api.delete(`/api/admin/chat/messages/${m.id}`); } catch { /* no-op */ }
  };

  if (roomsLoading) {
    return <div className="py-8 text-center text-sm text-zinc-500">Загрузка чата…</div>;
  }
  if (rooms.length === 0) {
    return (
      <div className="rounded-xl border border-zinc-800 bg-zinc-900 px-4 py-10 text-center">
        <MessageSquare size={22} className="mx-auto mb-2 text-zinc-600" />
        <p className="text-sm text-zinc-500">Чат станет доступен после жеребьёвки и распределения по группам.</p>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden flex flex-col h-[min(70vh,560px)]">
      {/* Комнаты */}
      {rooms.length > 1 && (
        <div className="flex gap-1 border-b border-zinc-800 px-2 py-2 overflow-x-auto">
          {rooms.map((r) => (
            <button
              key={r.id}
              onClick={() => setRoomId(r.id)}
              className={cn(
                "flex items-center gap-1.5 whitespace-nowrap rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors",
                r.id === roomId ? "bg-yellow-400 text-zinc-900" : "text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"
              )}
            >
              {r.title}
              {r.archived && <Archive size={11} className="opacity-70" />}
            </button>
          ))}
        </div>
      )}

      {/* Сообщения */}
      <div className="flex-1 overflow-y-auto px-3 py-3 space-y-2">
        {hasMore && (
          <div className="text-center">
            <button onClick={loadOlder} className="rounded-lg border border-zinc-700 px-3 py-1 text-[11px] text-zinc-400 hover:text-zinc-200">
              Показать ранние
            </button>
          </div>
        )}
        {messages.length === 0 ? (
          <div className="py-10 text-center text-sm text-zinc-600">Пока нет сообщений — начните общение</div>
        ) : (
          messages.map((m) => {
            const own = m.user_id === currentUserId;
            return (
              <div key={m.id} className={cn("flex items-end gap-2 group", own ? "flex-row-reverse" : "")}>
                {!own && <PlayerAvatar displayName={m.author_name} favoriteClub={m.author_club} size={26} />}
                <div className={cn("max-w-[78%] rounded-2xl px-3 py-1.5",
                  own ? "bg-yellow-500/15 rounded-br-sm" : "bg-zinc-800 rounded-bl-sm")}>
                  {!own && <p className="text-[11px] font-semibold text-zinc-400 mb-0.5">{m.author_name}</p>}
                  {m.deleted ? (
                    <p className="text-xs italic text-zinc-600">сообщение удалено</p>
                  ) : (
                    <p className="text-sm text-zinc-100 break-words whitespace-pre-wrap">{renderBody(m.body)}</p>
                  )}
                  <p className={cn("text-[10px] text-zinc-600 mt-0.5", own ? "text-right" : "")}>{fmtTime(m.created_at)}</p>
                </div>
                {isAdmin && !m.deleted && (
                  <button
                    onClick={() => onDelete(m)}
                    className="opacity-0 group-hover:opacity-100 transition-opacity rounded p-1 text-zinc-600 hover:text-red-400"
                    title="Удалить сообщение"
                  >
                    <Trash2 size={13} />
                  </button>
                )}
              </div>
            );
          })
        )}
        <div ref={bottomRef} />
      </div>

      {/* Ввод */}
      {room?.archived ? (
        <div className="border-t border-zinc-800 px-4 py-3 text-center text-xs text-zinc-500">
          <Archive size={13} className="inline mr-1" /> Турнир завершён — чат в архиве
        </div>
      ) : (
        <div className="relative border-t border-zinc-800 p-2">
          {mentionMatches.length > 0 && (
            <div className="absolute bottom-full left-2 mb-1 w-56 rounded-lg border border-zinc-700 bg-zinc-900 shadow-xl overflow-hidden">
              {mentionMatches.map((m) => (
                <button
                  key={m.user_id}
                  onClick={() => applyMention(m.display_name)}
                  className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm text-zinc-200 hover:bg-zinc-800"
                >
                  <PlayerAvatar displayName={m.display_name} favoriteClub={m.favorite_club} size={20} />
                  <span className="truncate">{m.display_name}</span>
                </button>
              ))}
            </div>
          )}
          <div className="flex items-center gap-2">
            <input
              ref={inputRef}
              value={text}
              onChange={(e) => onChange(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); submit(); } }}
              placeholder="Сообщение…  @имя для упоминания"
              maxLength={2000}
              className="flex-1 rounded-lg border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm text-zinc-100 placeholder-zinc-600 focus:border-yellow-400 focus:outline-none"
            />
            <button
              onClick={submit}
              disabled={sending || !text.trim()}
              className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-yellow-400 text-zinc-900 disabled:opacity-40 transition-opacity"
              aria-label="Отправить"
            >
              <Send size={16} />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
