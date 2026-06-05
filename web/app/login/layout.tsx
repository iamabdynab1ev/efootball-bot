import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Войти",
  description: "Вход в eFootball Web League — Google или логин администратора",
};

export default function Layout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
