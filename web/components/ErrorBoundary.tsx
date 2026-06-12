"use client";

import { Component, ReactNode } from "react";
import { AlertTriangle, RotateCcw } from "lucide-react";

interface Props { children: ReactNode }
interface State { hasError: boolean }

// Класс-компонент без хуков, поэтому строки берём напрямую из словаря
// текущего языка через localStorage (как это делает i18n-провайдер).
function errText(): { title: string; retry: string } {
  let lang = "ru";
  try {
    lang = localStorage.getItem("lang") || "ru";
  } catch { /* SSR/недоступный storage */ }
  if (lang === "uz") return { title: "Nimadir xato ketdi", retry: "Qayta urinish" };
  if (lang === "tg") return { title: "Чизе хато рафт", retry: "Аз нав кӯшиш кунед" };
  return { title: "Что-то пошло не так", retry: "Попробовать снова" };
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError() {
    return { hasError: true };
  }

  render() {
    if (this.state.hasError) {
      const { title, retry } = errText();
      return (
        <div className="flex min-h-[60vh] items-center justify-center">
          <div className="flex flex-col items-center gap-4 text-center">
            <div className="flex h-14 w-14 items-center justify-center rounded-full bg-red-500/10 text-red-400">
              <AlertTriangle size={24} />
            </div>
            <p className="text-sm font-semibold text-zinc-300">{title}</p>
            <button
              onClick={() => window.location.reload()}
              className="inline-flex items-center gap-2 rounded-lg bg-yellow-400 px-4 py-2 text-sm font-bold text-zinc-950 transition-all hover:bg-yellow-300 active:scale-[.98]"
            >
              <RotateCcw size={14} />
              {retry}
            </button>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}
