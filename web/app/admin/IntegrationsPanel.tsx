"use client";

import { useEffect, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { MessageCircle, Send, CheckCircle2, RefreshCw, Loader2, Users } from "lucide-react";
import { toast } from "sonner";
import { api } from "@/lib/api";
import { cn } from "@/lib/utils";

interface WAGroup { jid: string; name: string }
interface Integrations {
  telegram: { connected: boolean; chat_id?: number };
  whatsapp: { enabled: boolean; status?: string; group_jid?: string };
}

const WA_STATUS_LABEL: Record<string, string> = {
  connecting: "Подключение…",
  qr: "Ожидает сканирования QR",
  connected: "Подключено",
  logged_out: "Разлогинен — нужен повторный вход (перезапустите сервис и отсканируйте QR)",
  error: "Ошибка подключения — повторяем…",
};

// QR приходит защищённым эндпоинтом (нужен JWT) — тянем blob через axios,
// потому что <img src> не умеет отправлять Authorization.
function useWAQR(active: boolean) {
  const [url, setUrl] = useState<string | null>(null);
  useEffect(() => {
    if (!active) { setUrl(null); return; }
    let objectUrl: string | null = null;
    let on = true;
    const load = () => {
      api.get("/api/admin/wa/qr", { responseType: "blob" })
        .then((r) => {
          if (!on) return;
          if (objectUrl) URL.revokeObjectURL(objectUrl);
          objectUrl = URL.createObjectURL(r.data);
          setUrl(objectUrl);
        })
        .catch(() => { if (on) setUrl(null); });
    };
    load();
    const t = setInterval(load, 15000); // QR-код обновляется примерно раз в минуту
    return () => {
      on = false;
      clearInterval(t);
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [active]);
  return url;
}

export function IntegrationsPanel() {
  const qc = useQueryClient();
  const { data, isLoading } = useQuery<Integrations>({
    queryKey: ["admin", "integrations"],
    queryFn: async () => (await api.get("/api/admin/integrations")).data,
    refetchInterval: 10000,
  });

  const wa = data?.whatsapp;
  const tg = data?.telegram;
  const qrUrl = useWAQR(wa?.status === "qr");

  const [groups, setGroups] = useState<WAGroup[] | null>(null);
  const [loadingGroups, setLoadingGroups] = useState(false);
  const [saving, setSaving] = useState(false);

  const loadGroups = () => {
    setLoadingGroups(true);
    api.get("/api/admin/wa/groups")
      .then((r) => setGroups(r.data.groups ?? []))
      .catch(() => toast.error("Не удалось получить список групп"))
      .finally(() => setLoadingGroups(false));
  };

  const selectGroup = (jid: string) => {
    setSaving(true);
    api.post("/api/admin/wa/group", { jid })
      .then(() => {
        toast.success(jid ? "Группа WhatsApp подключена" : "Группа отключена");
        qc.invalidateQueries({ queryKey: ["admin", "integrations"] });
      })
      .catch(() => toast.error("Не удалось сохранить группу"))
      .finally(() => setSaving(false));
  };

  const logoutWA = () => {
    if (!window.confirm("Отвязать аккаунт WhatsApp? Сессия удалится, для повторного подключения нужно будет отсканировать новый QR.")) return;
    setSaving(true);
    api.post("/api/admin/wa/logout", {})
      .then(() => {
        toast.success("Аккаунт WhatsApp отвязан — через минуту появится новый QR");
        setGroups(null);
        qc.invalidateQueries({ queryKey: ["admin", "integrations"] });
      })
      .catch(() => toast.error("Не удалось отвязать аккаунт"))
      .finally(() => setSaving(false));
  };

  const disconnectTG = () => {
    if (!window.confirm("Отключить Telegram-группу от уведомлений?")) return;
    api.post("/api/admin/tg/disconnect", {})
      .then(() => {
        toast.success("Telegram-группа отключена");
        qc.invalidateQueries({ queryKey: ["admin", "integrations"] });
      })
      .catch(() => toast.error("Не удалось отключить"));
  };

  if (isLoading) {
    return <div className="py-10 text-center text-sm text-zinc-500">Загрузка…</div>;
  }

  return (
    <div className="space-y-4">
      <p className="text-xs text-zinc-500">
        Групповые уведомления: результаты матчей, жеребьёвки, напоминания о дедлайнах и объявления
        автоматически отправляются в подключённые группы.
      </p>

      {/* Telegram */}
      <div className="rounded-xl card-premium p-4">
        <h2 className="mb-2 flex items-center gap-2 text-sm font-semibold text-zinc-200">
          <Send size={15} className="text-sky-400" /> Telegram-группа
          {tg?.connected && <CheckCircle2 size={14} className="text-green-400" />}
        </h2>
        {tg?.connected ? (
          <p className="text-sm text-zinc-400">
            Подключена (chat_id: <span className="tabular-nums text-zinc-300">{tg.chat_id}</span>).
            <button onClick={disconnectTG} className="ml-3 text-xs text-red-400 hover:underline">отключить</button>
          </p>
        ) : (
          <ol className="list-decimal space-y-1 pl-5 text-sm text-zinc-400">
            <li>Добавьте бота турнира в вашу Telegram-группу.</li>
            <li>Отправьте в группе команду <code className="rounded bg-zinc-800 px-1.5 py-0.5 text-xs text-zinc-300">/connect</code> (от аккаунта администратора).</li>
          </ol>
        )}
      </div>

      {/* WhatsApp */}
      <div className="rounded-xl card-premium p-4">
        <h2 className="mb-2 flex items-center gap-2 text-sm font-semibold text-zinc-200">
          <MessageCircle size={15} className="text-green-400" /> WhatsApp-группа
          {wa?.enabled && wa.status === "connected" && <CheckCircle2 size={14} className="text-green-400" />}
        </h2>

        {!wa?.enabled ? (
          <div className="space-y-2 text-sm text-zinc-400">
            <p>Канал выключен. Чтобы включить, задайте в Render переменную окружения <code className="rounded bg-zinc-800 px-1.5 py-0.5 text-xs text-zinc-300">WA_ENABLED=1</code> и перезапустите сервис.</p>
            <p className="text-xs text-orange-400/90">⚠️ Используйте отдельный номер (не личный): интеграция неофициальная, аккаунт может быть заблокирован WhatsApp.</p>
          </div>
        ) : (
          <div className="space-y-3">
            <p className="text-sm text-zinc-400">
              Статус: <span className={cn("font-medium", wa.status === "connected" ? "text-green-400" : "text-yellow-400")}>
                {WA_STATUS_LABEL[wa.status ?? ""] ?? wa.status}
              </span>
            </p>

            {wa.status === "qr" && (
              <div className="space-y-2">
                <p className="text-xs text-zinc-500">
                  WhatsApp (на отдельном номере) → Настройки → Связанные устройства → Привязать устройство → отсканируйте:
                </p>
                {qrUrl ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={qrUrl} alt="QR для входа в WhatsApp" className="h-56 w-56 rounded-lg bg-white p-2" />
                ) : (
                  <div className="flex h-56 w-56 items-center justify-center rounded-lg bg-zinc-800">
                    <Loader2 size={22} className="animate-spin text-zinc-500" />
                  </div>
                )}
              </div>
            )}

            {wa.status === "connected" && (
              <div className="space-y-2">
                <button
                  onClick={logoutWA}
                  disabled={saving}
                  className="rounded-lg border border-red-500/30 px-3 py-1.5 text-xs font-medium text-red-400 transition-colors hover:bg-red-500/10"
                >
                  Отвязать аккаунт WhatsApp
                </button>
                {wa.group_jid ? (
                  <p className="text-sm text-zinc-400">
                    Группа подключена: <span className="text-zinc-300">{groups?.find((g) => g.jid === wa.group_jid)?.name ?? wa.group_jid}</span>
                    <button onClick={() => selectGroup("")} disabled={saving} className="ml-3 text-xs text-red-400 hover:underline">
                      отключить
                    </button>
                  </p>
                ) : (
                  <p className="text-sm text-zinc-400">Группа не выбрана — выберите, куда слать уведомления.</p>
                )}
                <button
                  onClick={loadGroups}
                  disabled={loadingGroups}
                  className="flex items-center gap-1.5 rounded-lg bg-zinc-800 px-3 py-1.5 text-xs font-medium text-zinc-200 hover:bg-zinc-700 transition-colors"
                >
                  {loadingGroups ? <Loader2 size={13} className="animate-spin" /> : <RefreshCw size={13} />}
                  Показать группы аккаунта
                </button>
                {groups && (
                  groups.length === 0 ? (
                    <p className="text-xs text-zinc-500">Аккаунт не состоит ни в одной группе — добавьте его в вашу группу WhatsApp.</p>
                  ) : (
                    <div className="divide-y divide-zinc-800 overflow-hidden rounded-lg border border-zinc-800">
                      {groups.map((g) => (
                        <button
                          key={g.jid}
                          onClick={() => selectGroup(g.jid)}
                          disabled={saving}
                          className={cn(
                            "flex w-full items-center gap-2 px-3 py-2 text-left text-sm transition-colors hover:bg-zinc-800/60",
                            g.jid === wa.group_jid ? "text-yellow-400" : "text-zinc-300",
                          )}
                        >
                          <Users size={14} className="flex-shrink-0 text-zinc-500" />
                          <span className="min-w-0 flex-1 truncate">{g.name || g.jid}</span>
                          {g.jid === wa.group_jid && <CheckCircle2 size={14} className="flex-shrink-0" />}
                        </button>
                      ))}
                    </div>
                  )
                )}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
