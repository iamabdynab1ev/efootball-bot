import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Зал Славы",
  description: "Чемпионы и лучшие игроки прошедших сезонов eFootLeague",
};

export default function Layout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
