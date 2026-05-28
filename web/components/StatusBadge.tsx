import { Badge } from "@/components/ui/badge";

const LEAGUE: Record<string, { label: string; variant: string }> = {
  registration: { label: "Набор",      variant: "blue"    },
  active:       { label: "Сезон",      variant: "green"   },
  finished:     { label: "Завершена",  variant: "purple"  },
  archived:     { label: "Архив",      variant: "default" },
};

const MATCH: Record<string, { label: string; variant: string }> = {
  scheduled:       { label: "Запланирован",     variant: "default" },
  pending_confirm: { label: "Ждёт подтвер.",    variant: "amber"   },
  confirmed:       { label: "Подтверждён",      variant: "green"   },
  disputed:        { label: "Спор",             variant: "red"     },
  cancelled:       { label: "Отменён",          variant: "default" },
};

export function LeagueStatusBadge({ status }: { status: string }) {
  const cfg = LEAGUE[status] ?? { label: status, variant: "default" };
  return <Badge variant={cfg.variant as any}>{cfg.label}</Badge>;
}

export function MatchStatusBadge({ status }: { status: string }) {
  const cfg = MATCH[status] ?? { label: status, variant: "default" };
  return <Badge variant={cfg.variant as any}>{cfg.label}</Badge>;
}

export function statusText(status: string) {
  return MATCH[status]?.label ?? LEAGUE[status]?.label ?? status;
}
