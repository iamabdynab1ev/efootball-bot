"use client";

import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Phone, Save } from "lucide-react";
import { toast } from "sonner";
import { fetchSupport, adminSetSupport, SupportContact } from "@/lib/api";
import { useLang } from "@/lib/i18n";

/**
 * Админ-форма контактов поддержки: телефон / WhatsApp / Telegram.
 * То, что задано здесь, видят игроки в разделе «Поддержка».
 */
export function AdminSupportForm() {
  const { t } = useLang();
  const qc = useQueryClient();
  const { data } = useQuery({ queryKey: ["support"], queryFn: fetchSupport, staleTime: 300000 });
  const [form, setForm] = useState<SupportContact>({ phone: "", whatsapp: "", telegram: "" });

  useEffect(() => {
    if (data) setForm({ phone: data.phone || "", whatsapp: data.whatsapp || "", telegram: data.telegram || "" });
  }, [data]);

  const mutation = useMutation({
    mutationFn: () => adminSetSupport(form),
    onSuccess: () => {
      toast.success(t("settings.supportSaved"));
      qc.invalidateQueries({ queryKey: ["support"] });
    },
    onError: () => toast.error(t("settings.broadcastError")),
  });

  const field = (label: string, key: keyof SupportContact, placeholder: string) => (
    <div className="space-y-1">
      <label className="text-xs font-medium text-zinc-400">{label}</label>
      <input
        value={form[key]}
        onChange={(e) => setForm({ ...form, [key]: e.target.value })}
        placeholder={placeholder}
        maxLength={100}
        className="w-full rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-2 text-sm text-zinc-100 placeholder-zinc-600 outline-none focus:border-yellow-500"
      />
    </div>
  );

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-900 p-4 space-y-3">
      <div className="flex items-center gap-2 text-sm font-bold text-zinc-100">
        <Phone size={16} className="text-yellow-400" /> {t("settings.editSupport")}
      </div>
      {field(t("settings.supportPhoneLabel"), "phone", "+992 90 123 45 67")}
      {field(t("settings.supportWhatsappLabel"), "whatsapp", "+992901234567")}
      {field(t("settings.supportTelegramLabel"), "telegram", "@username")}
      <button
        onClick={() => mutation.mutate()}
        disabled={mutation.isPending}
        className="flex items-center justify-center gap-2 rounded-lg bg-yellow-400 px-4 py-2 text-sm font-bold text-zinc-950 transition-opacity hover:opacity-90 disabled:opacity-50"
      >
        <Save size={15} /> {mutation.isPending ? "..." : t("settings.saveSupport")}
      </button>
    </div>
  );
}
