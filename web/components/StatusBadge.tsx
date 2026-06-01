"use client";

import { Badge } from "@/components/ui/badge";
import { useLang } from "@/lib/i18n";

export function LeagueStatusBadge({ status }: { status: string }) {
  const { t } = useLang();

  const cfg: Record<string, { label: string; variant: string }> = {
    registration: { label: t("status.leagueRegistration"), variant: "blue"    },
    active:       { label: t("status.leagueSeason"),       variant: "green"   },
    finished:     { label: t("status.leagueFinished"),     variant: "purple"  },
    archived:     { label: t("status.leagueArchived"),     variant: "default" },
  };

  const entry = cfg[status] ?? { label: status, variant: "default" };
  return <Badge variant={entry.variant as any}>{entry.label}</Badge>;
}

export function MatchStatusBadge({ status }: { status: string }) {
  const { t } = useLang();

  const cfg: Record<string, { label: string; variant: string }> = {
    scheduled:       { label: t("status.matchScheduled"), variant: "default" },
    pending_confirm: { label: t("status.matchPending"),   variant: "amber"   },
    confirmed:       { label: t("status.matchConfirmed"), variant: "green"   },
    disputed:        { label: t("status.matchDisputed"),  variant: "red"     },
    cancelled:       { label: t("status.matchCancelled"), variant: "default" },
  };

  const entry = cfg[status] ?? { label: status, variant: "default" };
  return <Badge variant={entry.variant as any}>{entry.label}</Badge>;
}

export function statusText(status: string, t: (k: any) => string): string {
  const all: Record<string, string> = {
    scheduled:       t("status.matchScheduled"),
    pending_confirm: t("status.matchPending"),
    confirmed:       t("status.matchConfirmed"),
    disputed:        t("status.matchDisputed"),
    cancelled:       t("status.matchCancelled"),
    registration:    t("status.leagueRegistration"),
    active:          t("status.leagueSeason"),
    finished:        t("status.leagueFinished"),
    archived:        t("status.leagueArchived"),
  };
  return all[status] ?? status;
}
