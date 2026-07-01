"use client";

import { memo, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { Send, Trash2, ChevronDown, Check, CheckCheck, Pencil, X } from "lucide-react";
import { api } from "@/lib/api";
import { markRead, sendTyping, deleteChatMessage, editChatMessage, type ChatMessage, type ChatMember } from "@/lib/chat";
import { useSSE } from "@/lib/sse";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { cn } from "@/lib/utils";

function fmtTime(iso: string) {
  return new Date(iso).toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
}

function dayLabel(iso: string) {
  const d = new Date(iso);
  const today = new Date();
  const yest = new Date(); yest.setDate(today.getDate() - 1);
  if (d.toDateString() === today.toDateString()) return "Сегодня";
  if (d.toDateString() === yest.toDateString()) return "Вчера";
  return d.toLocaleDateString("ru-RU", { day: "2-digit", month: "long" });
}

// Подсветка @упоминаний.
function renderBody(body: string) {
  return body.split(/(@[\wА-Яа-яЁё]+)/g).map((p, i) =>
    p.startsWith("@")
      ? <span key={i} className="font-semibold text-yellow-400">{p}</span>
      : <span key={i}>{p}</span>
  );
}

interface RowData {
  m: ChatMessage;
  own: boolean;
  grouped: boolean;  // подряд от того же автора в пределах 5 мин — тесная группа
  showDay: boolean;  // сменился день — показать разделитель
  day: string;
  showName: boolean; // показывать имя автора (в ЛС/у своих не нужно)
}

// MessageRow мемоизирован: при новом сообщении перерисовывается только оно, а не
// весь список (объекты старых сообщений сохраняют идентичность).
const MessageRow = memo(function MessageRow({
  m, own, grouped, showDay, day, showName, isAdmin, onDelete, onEdit, showReceipts, otherReads,
}: RowData & {
  isAdmin: boolean;
  onDelete: (id: number, own: boolean) => void;
  onEdit: (m: ChatMessage) => void;
  showReceipts: boolean;
  otherReads: number[];
}) {
  // Сколько собеседников прочитали это сообщение (для ✓/✓✓ и счётчика в группе).
  const receipts = own && showReceipts && !m.deleted;
  const total = otherReads.length;
  const readers = receipts ? otherReads.filter((v) => v >= m.id).length : 0;
  const allRead = receipts && total > 0 && readers === total;
  return (
    <div>
      {showDay && (
        <div className="flex justify-center py-2.5">
          <span className="chat-pill rounded-full px-3 py-1 text-[10px] font-semibold tracking-wide text-zinc-400">{day}</span>
        </div>
      )}
      <div className={cn("msg-in flex items-end gap-2 group", own ? "flex-row-reverse" : "", grouped ? "mt-0.5" : "mt-2.5")}>
        {/* Аватар = логотип клуба автора; резервируем место у сгруппированных */}
        {grouped
          ? <div className="w-[26px] flex-shrink-0" />
          : <PlayerAvatar displayName={m.author_name} favoriteClub={m.author_club} size={26} />}
        <div className={cn(
          "min-w-0 max-w-[85%] sm:max-w-[70%] px-3.5 py-2 rounded-2xl shadow-sm shadow-black/20",
          own ? "bubble-out rounded-br-md" : "bubble-in rounded-bl-md",
          grouped && (own ? "rounded-tr-md" : "rounded-tl-md"),
        )}>
          {showName && !grouped && (
            <p className={cn("text-[11px] font-semibold mb-0.5", own ? "text-right text-yellow-300/90" : "text-yellow-400/90")}>
              {own ? "Вы" : m.author_name}
            </p>
          )}
          {m.deleted ? (
            <p className="text-xs italic text-zinc-500">сообщение удалено</p>
          ) : (
            <p className="text-[15px] leading-snug text-zinc-100 whitespace-pre-wrap break-words [overflow-wrap:anywhere]">
              {renderBody(m.body)}
            </p>
          )}
          <div className={cn("flex items-center gap-1 mt-0.5", own ? "justify-end" : "")}>
            {m.edited && !m.deleted && <span className="text-[10px] text-zinc-600 italic">изменено</span>}
            <span className="text-[10px] text-zinc-500">{fmtTime(m.created_at)}</span>
            {receipts && (
              readers === 0
                ? <Check size={13} className="text-zinc-500" />
                : (
                  <span className={cn("flex items-center gap-0.5", allRead ? "text-sky-400" : "text-sky-400/60")}>
                    <CheckCheck size={13} />
                    {total > 1 && <span className="text-[9px] font-semibold">{readers}</span>}
                  </span>
                )
            )}
          </div>
        </div>
        {!m.deleted && (own || isAdmin) && (
          <div className="flex items-center gap-0.5 self-center opacity-70 sm:opacity-0 sm:group-hover:opacity-100 transition-opacity">
            {own && (
              <button
                onClick={() => onEdit(m)}
                className="rounded p-1 text-zinc-600 hover:text-yellow-400"
                title="Редактировать"
              >
                <Pencil size={13} />
              </button>
            )}
            <button
              onClick={() => onDelete(m.id, own)}
              className="rounded p-1 text-zinc-600 hover:text-red-400"
              title="Удалить сообщение"
            >
              <Trash2 size={13} />
            </button>
          </div>
        )}
      </div>
    </div>
  );
});

export interface ChatThreadProps {
  messages: ChatMessage[];
  hasMore: boolean;
  loadOlder: () => Promise<void> | void;
  send: (body: string) => Promise<void>;
  currentUserId?: number;
  isAdmin?: boolean;
  archived?: boolean;
  archivedNote?: string;
  /** Участники для @упоминаний (в ЛС не передаём — упоминания не нужны). */
  members?: ChatMember[];
  /** Показывать имя автора над чужими сообщениями (в ЛС не нужно). */
  showAuthorNames?: boolean;
  /** Плейсхолдер поля ввода. */
  placeholder?: string;
  /** Активна ли комната (для сброса скролла/ввода при переключении). */
  resetKey?: string | number;
  /** id комнаты — включает отметки прочтения и «печатает…» (для ЛС). */
  roomId?: number;
  /** Показывать ✓/✓✓ на своих сообщениях. */
  showReceipts?: boolean;
  /** Начальный прогресс прочтения участников {userId: lastRead} (для ✓✓). */
  initialReads?: Record<number, number>;
  /** Слать/принимать «печатает…». */
  showTyping?: boolean;
  /** Колбэк: собеседник печатает / перестал (для статуса в шапке). */
  onPeerTyping?: (typing: boolean) => void;
}

// ChatThread — переиспользуемая лента диалога: сообщения (bottom-anchored),
// авто-скролл, кнопка «вниз», @упоминания, поле ввода. Используется и в чате
// турнира (на комнату), и в личных сообщениях.
export function ChatThread({
  messages, hasMore, loadOlder, send,
  currentUserId, isAdmin = false, archived = false,
  archivedNote = "Чат в архиве", members = [], showAuthorNames = true,
  placeholder = "Сообщение…", resetKey,
  roomId, showReceipts = false, initialReads, showTyping = false, onPeerTyping,
}: ChatThreadProps) {
  const [text, setText] = useState("");
  const [sending, setSending] = useState(false);
  const [mentionQuery, setMentionQuery] = useState<string | null>(null);
  const [showJump, setShowJump] = useState(false);
  const [editing, setEditing] = useState<{ id: number } | null>(null); // правка своего сообщения
  // Прогресс прочтения по участникам {userId: lastRead}. Для ЛС — один собеседник,
  // для группы — все участники (сколько прочитали каждое сообщение).
  const [readsByUser, setReadsByUser] = useState<Record<number, number>>(initialReads ?? {});
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const typingSentRef = useRef(0);   // троттлинг отправки «печатает…»
  const readSentRef = useRef(0);     // до какого id уже отметили прочтение
  const typingTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Начальный прогресс может приехать позже монтирования — вливаем по максимуму.
  useEffect(() => {
    if (!initialReads) return;
    setReadsByUser((prev) => {
      let changed = false;
      const next = { ...prev };
      for (const [uid, lr] of Object.entries(initialReads)) {
        if ((next[+uid] ?? 0) < lr) { next[+uid] = lr; changed = true; }
      }
      return changed ? next : prev;
    });
  }, [initialReads]);

  // Живое обновление ✓✓: кто-то из участников дочитал.
  useSSE("chat_read", (d: any) => {
    if (!showReceipts || roomId == null || !d || d.room_id !== roomId) return;
    const uid = Number(d.user_id) || 0;
    const lr = Number(d.last_read_id) || 0;
    if (!uid) return;
    setReadsByUser((prev) => (prev[uid] >= lr ? prev : { ...prev, [uid]: lr }));
  }, showReceipts && roomId != null);

  // Массив last_read остальных участников (без себя) — для подсчёта прочтений.
  const otherReads = useMemo(
    () => Object.entries(readsByUser)
      .filter(([uid]) => Number(uid) !== currentUserId)
      .map(([, lr]) => lr),
    [readsByUser, currentUserId],
  );

  // «печатает…»: показываем и гасим через 3.5с тишины.
  useSSE("chat_typing", (d: any) => {
    if (!showTyping || roomId == null || !d || d.room_id !== roomId) return;
    onPeerTyping?.(true);
    if (typingTimerRef.current) clearTimeout(typingTimerRef.current);
    typingTimerRef.current = setTimeout(() => onPeerTyping?.(false), 3500);
  }, showTyping && roomId != null);

  useEffect(() => () => { if (typingTimerRef.current) clearTimeout(typingTimerRef.current); }, []);

  // Отмечаем прочитанным до последнего сообщения (при открытии и на новые входящие).
  useEffect(() => {
    if (roomId == null || messages.length === 0) return;
    const lastId = messages[messages.length - 1].id;
    if (lastId > readSentRef.current) {
      readSentRef.current = lastId;
      markRead(roomId, lastId);
    }
  }, [roomId, messages]);

  const scrollRef = useRef<HTMLDivElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);
  const lastIdRef = useRef(0);
  const stickRef = useRef(true);
  const prevHeightRef = useRef(0);

  const rows = useMemo<RowData[]>(() => {
    const out: RowData[] = [];
    let prevAuthor: number | null | undefined;
    let prevDay = "";
    let prevTime = 0;
    for (const m of messages) {
      const day = dayLabel(m.created_at);
      const t = new Date(m.created_at).getTime();
      const grouped = m.user_id === prevAuthor && day === prevDay && t - prevTime < 5 * 60 * 1000;
      out.push({
        m, own: m.user_id === currentUserId, grouped,
        showDay: day !== prevDay, day, showName: showAuthorNames,
      });
      prevAuthor = m.user_id; prevDay = day; prevTime = t;
    }
    return out;
  }, [messages, currentUserId, showAuthorNames]);

  const onScroll = useCallback(() => {
    const el = scrollRef.current;
    if (!el) return;
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 120;
    stickRef.current = nearBottom;
    setShowJump(!nearBottom);
  }, []);

  const jumpToBottom = useCallback(() => {
    stickRef.current = true;
    setShowJump(false);
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, []);

  // Автоскролл вниз при новом сообщении — только если пользователь у низа.
  useEffect(() => {
    const lastId = messages.length ? messages[messages.length - 1].id : 0;
    if (lastId === lastIdRef.current) return;
    const initial = lastIdRef.current === 0;
    const grew = lastId > lastIdRef.current;
    lastIdRef.current = lastId;
    if (grew && (initial || stickRef.current)) {
      bottomRef.current?.scrollIntoView({ behavior: initial ? "auto" : "smooth" });
    }
  }, [messages]);

  // Смена комнаты — сбрасываем состояние скролла/ввода/прочтения.
  useEffect(() => {
    lastIdRef.current = 0;
    stickRef.current = true;
    readSentRef.current = 0;
    setText(""); setMentionQuery(null); setShowJump(false); setEditing(null);
    onPeerTyping?.(false);
  }, [resetKey]); // eslint-disable-line react-hooks/exhaustive-deps

  // Автовысота поля ввода.
  useLayoutEffect(() => {
    const el = inputRef.current;
    if (!el) return;
    el.style.height = "0px";
    el.style.height = Math.min(el.scrollHeight, 120) + "px";
  }, [text]);

  const onChange = (v: string) => {
    setText(v);
    // «печатает…»: троттлим отправку до раза в 2с.
    if (showTyping && roomId != null && v.trim()) {
      const now = Date.now();
      if (now - typingSentRef.current > 2000) {
        typingSentRef.current = now;
        sendTyping(roomId);
      }
    }
    if (members.length === 0) return; // упоминания не нужны (напр. в ЛС)
    const caret = inputRef.current?.selectionStart ?? v.length;
    const m = v.slice(0, caret).match(/@([\wА-Яа-яЁё]*)$/);
    setMentionQuery(m ? m[1].toLowerCase() : null);
  };

  const mentionMatches = useMemo(() => {
    if (mentionQuery == null) return [];
    // Показываем всех подходящих участников (список прокручивается), а не первые 6.
    return members.filter((m) => m.display_name.toLowerCase().includes(mentionQuery)).slice(0, 50);
  }, [mentionQuery, members]);

  const applyMention = (name: string) => {
    const first = name.split(" ")[0];
    setText((t) => t.replace(/@([\wА-Яа-яЁё]*)$/, `@${first} `));
    setMentionQuery(null);
    inputRef.current?.focus();
  };

  const submit = async () => {
    const body = text.trim();
    if (!body || sending || archived) return;
    setSending(true);
    try {
      if (editing) {
        // Сохраняем правку своего сообщения (текст обновится по SSE chat_edited).
        await editChatMessage(editing.id, body);
        setEditing(null);
        setText("");
        setMentionQuery(null);
      } else {
        stickRef.current = true;
        await send(body);
        setText("");
        setMentionQuery(null);
        requestAnimationFrame(() => bottomRef.current?.scrollIntoView({ behavior: "smooth" }));
      }
    } finally {
      setSending(false);
    }
  };

  const cancelEdit = () => { setEditing(null); setText(""); };

  const onLoadOlder = useCallback(async () => {
    prevHeightRef.current = scrollRef.current?.scrollHeight ?? 0;
    await loadOlder();
  }, [loadOlder]);

  // После подгрузки старых — сохраняем визуальную позицию (без прыжка).
  useLayoutEffect(() => {
    const el = scrollRef.current;
    if (el && prevHeightRef.current) {
      el.scrollTop += el.scrollHeight - prevHeightRef.current;
      prevHeightRef.current = 0;
    }
  }, [messages]);

  const onDelete = useCallback((id: number, own: boolean) => {
    if (!window.confirm("Удалить сообщение?")) return;
    // Своё — пользовательский роут; чужое (админ) — админский.
    const p = own
      ? deleteChatMessage(id)
      : api.delete(`/api/admin/chat/messages/${id}`);
    p.catch(() => { /* no-op */ });
  }, []);

  const onEdit = useCallback((m: ChatMessage) => {
    setEditing({ id: m.id });
    setText(m.body);
    setMentionQuery(null);
    requestAnimationFrame(() => inputRef.current?.focus());
  }, []);

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* Сообщения. Внутренний spacer flex-1 прижимает переписку к низу. */}
      <div className="relative flex-1 min-h-0">
        <div ref={scrollRef} onScroll={onScroll} className="chat-surface h-full overflow-y-auto overscroll-contain">
          <div className="flex min-h-full flex-col px-3 py-3">
            <div className="flex-1" />
            {hasMore && (
              <div className="text-center pb-3">
                <button onClick={onLoadOlder} className="chat-pill rounded-full px-3.5 py-1.5 text-[11px] font-medium text-zinc-300 hover:text-white transition-colors">
                  Показать ранние
                </button>
              </div>
            )}
            {rows.length === 0 ? (
              <div className="flex flex-col items-center gap-2 py-14 text-center">
                <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-yellow-400/10 text-yellow-400">
                  <Send size={20} />
                </div>
                <p className="text-sm font-medium text-zinc-400">Пока нет сообщений</p>
                <p className="text-xs text-zinc-600">Напишите первым — начните общение</p>
              </div>
            ) : (
              rows.map((row) => (
                <MessageRow key={row.m.id} {...row} isAdmin={isAdmin} onDelete={onDelete} onEdit={onEdit}
                  showReceipts={showReceipts} otherReads={otherReads} />
              ))
            )}
            <div ref={bottomRef} />
          </div>
        </div>
        {showJump && (
          <button
            onClick={jumpToBottom}
            aria-label="Вниз к последним сообщениям"
            className="chat-pill absolute bottom-3 right-3 flex h-10 w-10 items-center justify-center rounded-full text-zinc-200 hover:text-yellow-400 transition-colors"
          >
            <ChevronDown size={19} />
          </button>
        )}
      </div>

      {/* Ввод */}
      {archived ? (
        <div className="border-t border-zinc-800 px-4 py-3 text-center text-xs text-zinc-500 flex-shrink-0">
          {archivedNote}
        </div>
      ) : (
        <div className="relative border-t border-white/5 bg-zinc-950/30 p-2 flex-shrink-0 pb-[max(0.5rem,env(safe-area-inset-bottom))]">
          {editing && (
            <div className="mb-1.5 flex items-center gap-2 rounded-xl border-l-2 border-yellow-400/70 bg-zinc-800/50 px-3 py-1.5">
              <Pencil size={13} className="flex-shrink-0 text-yellow-400" />
              <span className="flex-1 text-xs text-zinc-400">Редактирование сообщения</span>
              <button onClick={cancelEdit} aria-label="Отменить" className="rounded p-0.5 text-zinc-500 hover:text-zinc-200">
                <X size={14} />
              </button>
            </div>
          )}
          {mentionMatches.length > 0 && (
            <div className="absolute bottom-full left-2 right-2 mb-2 max-h-52 overflow-y-auto rounded-2xl border border-white/10 bg-zinc-900/95 backdrop-blur shadow-2xl shadow-black/50">
              {mentionMatches.map((m) => (
                <button
                  key={m.user_id}
                  onClick={() => applyMention(m.display_name)}
                  className="flex w-full items-center gap-2.5 px-3 py-2 text-left text-sm text-zinc-200 hover:bg-white/5 transition-colors"
                >
                  <PlayerAvatar displayName={m.display_name} favoriteClub={m.favorite_club} size={22} />
                  <span className="truncate">{m.display_name}</span>
                </button>
              ))}
            </div>
          )}
          <div className="flex items-end gap-2">
            <div className="flex flex-1 items-end rounded-[1.4rem] border border-zinc-700/70 bg-zinc-950/70 px-3.5 py-1 transition-colors focus-within:border-yellow-400/70 focus-within:ring-2 focus-within:ring-yellow-400/15">
              <textarea
                ref={inputRef}
                rows={1}
                value={text}
                onChange={(e) => onChange(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); submit(); } }}
                placeholder={placeholder}
                maxLength={2000}
                className="flex-1 resize-none border-0 bg-transparent py-1.5 text-[15px] text-zinc-100 placeholder-zinc-600 focus:outline-none leading-snug max-h-[120px]"
              />
            </div>
            <button
              onClick={submit}
              disabled={sending || !text.trim()}
              className={cn(
                "flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-full text-zinc-950 transition-all active:scale-90",
                "bg-gradient-to-br from-[#d9ff3d] to-[#a3cc1e]",
                text.trim() ? "shadow-[0_3px_14px_rgba(200,241,53,0.35)]" : "opacity-40",
              )}
              aria-label={editing ? "Сохранить" : "Отправить"}
            >
              {editing ? <Check size={18} strokeWidth={2.5} /> : <Send size={17} />}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
