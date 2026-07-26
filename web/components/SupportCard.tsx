"use client";

import { useQuery } from "@tanstack/react-query";
import { Phone, MessageCircle, Send, LifeBuoy } from "lucide-react";
import { fetchSupport } from "@/lib/api";
import { useLang } from "@/lib/i18n";

// Нормализация контактов в ссылки.
function telLink(phone: string) {
  return "tel:" + phone.replace(/[^\d+]/g, "");
}
function waLink(v: string) {
  if (/^https?:\/\//.test(v)) return v;
  return "https://wa.me/" + v.replace(/[^\d]/g, "");
}
function tgLink(v: string) {
  if (/^https?:\/\//.test(v)) return v;
  return "https://t.me/" + v.replace(/^@/, "");
}

/**
 * Карточка поддержки для игроков: показывает кнопки «Позвонить / WhatsApp /
 * Telegram» — только те контакты, что задал админ.
 */
export function SupportCard() {
  const { t } = useLang();
  const { data } = useQuery({ queryKey: ["support"], queryFn: fetchSupport, staleTime: 300000 });

  const phone = data?.phone?.trim();
  const wa = data?.whatsapp?.trim();
  const tg = data?.telegram?.trim();
  const hasAny = phone || wa || tg;

  return (
    <div className="rounded-xl card-premium p-4">
      <div className="flex items-center gap-3">
        <div className="flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-xl bg-[#229ED9]/15 text-[#229ED9]">
          <LifeBuoy size={20} />
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-bold text-zinc-100">{t("settings.support")}</p>
          <p className="text-xs text-zinc-400 mt-0.5">{t("settings.supportDesc")}</p>
        </div>
      </div>

      {!hasAny ? (
        <p className="mt-3 text-xs text-zinc-500 text-center py-2">{t("settings.noSupport")}</p>
      ) : (
        <div className="mt-3 grid grid-cols-1 gap-2">
          {phone && (
            <a href={telLink(phone)} className="flex items-center justify-center gap-2 rounded-lg bg-green-500 hover:bg-green-600 py-2.5 text-sm font-bold text-white transition-colors">
              <Phone size={15} /> {t("settings.call")} {phone}
            </a>
          )}
          {wa && (
            <a href={waLink(wa)} target="_blank" rel="noopener noreferrer" className="flex items-center justify-center gap-2 rounded-lg bg-[#25D366] hover:opacity-90 py-2.5 text-sm font-bold text-white transition-opacity">
              <MessageCircle size={15} /> {t("settings.openWhatsapp")}
            </a>
          )}
          {tg && (
            <a href={tgLink(tg)} target="_blank" rel="noopener noreferrer" className="flex items-center justify-center gap-2 rounded-lg bg-[#229ED9] hover:bg-[#1a8bbf] py-2.5 text-sm font-bold text-white transition-colors">
              <Send size={15} /> {t("settings.openTelegram")}
            </a>
          )}
        </div>
      )}
    </div>
  );
}
