#!/usr/bin/env python3
"""Генератор звуковых ассетов уведомлений (web/public/sounds/*.wav).

Каждому типу события — свой характер:
  challenge — заметный «боевой» арпеджио-сигнал (вызов на матч);
  result    — короткая фанфара (итог/счёт матча);
  message   — мягкий двухтоновый «динь» (новое сообщение);
  sent      — тихий короткий «свист» (подтверждение отправки);
  system    — нейтральный одиночный блип (системные события).

Запуск: python3 scripts/gen_sounds.py  (перезаписывает существующие файлы).
"""
import math
import os
import struct
import wave

SR = 24000  # 24 кГц mono 16-bit — маленькие файлы, для UI-сигналов достаточно


def note(freq, dur, peak, attack=0.012, harmonics=(1.0,)):
    """Синусоида с экспоненциальной атакой/затуханием и обертонами."""
    n = int(SR * dur)
    out = []
    for i in range(n):
        t = i / SR
        env = (t / attack) if t < attack else math.exp(-4.5 * (t - attack) / dur)
        s = 0.0
        for k, amp in enumerate(harmonics, start=1):
            s += amp * math.sin(2 * math.pi * freq * k * t)
        out.append(peak * env * s)
    return out


def sweep(f0, f1, dur, peak):
    """Частотный свип (для «свиста» отправки)."""
    n = int(SR * dur)
    out, phase = [], 0.0
    for i in range(n):
        t = i / SR
        f = f0 + (f1 - f0) * (t / dur)
        phase += 2 * math.pi * f / SR
        env = math.sin(math.pi * t / dur)  # колокол
        out.append(peak * env * math.sin(phase))
    return out


def mix(*tracks):
    """Наложение дорожек: (offset_sec, samples)."""
    total = max(int(off * SR) + len(s) for off, s in tracks)
    out = [0.0] * total
    for off, s in tracks:
        base = int(off * SR)
        for i, v in enumerate(s):
            out[base + i] += v
    return out


def write(name, samples):
    path = os.path.join(os.path.dirname(__file__), "..", "web", "public", "sounds", name)
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with wave.open(path, "w") as w:
        w.setnchannels(1)
        w.setsampwidth(2)
        w.setframerate(SR)
        frames = b"".join(
            struct.pack("<h", max(-32767, min(32767, int(v * 32767)))) for v in samples
        )
        w.writeframes(frames)
    print(f"  {name}: {len(samples)/SR:.2f}s, {os.path.getsize(path)//1024} KB")


A5, Cs6, E6, A6 = 880.0, 1108.73, 1318.51, 1760.0
G5, C6, B5 = 783.99, 1046.50, 987.77

print("Генерирую web/public/sounds/:")

# Вызов на матч — восходящее арпеджио дважды, ярко и настойчиво.
write("challenge.wav", mix(
    (0.00, note(A5, 0.14, 0.30, harmonics=(1.0, 0.25))),
    (0.10, note(Cs6, 0.14, 0.30, harmonics=(1.0, 0.25))),
    (0.20, note(E6, 0.22, 0.34, harmonics=(1.0, 0.25))),
    (0.42, note(E6, 0.12, 0.26, harmonics=(1.0, 0.25))),
    (0.52, note(A6, 0.30, 0.34, harmonics=(1.0, 0.2))),
))

# Результат матча — короткая фанфара: квинта, затем мажорное трезвучие.
write("result.wav", mix(
    (0.00, note(G5, 0.16, 0.28, harmonics=(1.0, 0.3))),
    (0.12, note(C6, 0.34, 0.30, harmonics=(1.0, 0.3))),
    (0.12, note(E6, 0.34, 0.16, harmonics=(1.0,))),
))

# Сообщение — привычный мягкий «динь» (как прежний синтезированный).
write("message.wav", mix(
    (0.00, note(B5, 0.16, 0.20)),
    (0.09, note(E6, 0.26, 0.16)),
))

# Отправка — едва слышный короткий «свист» вверх.
write("sent.wav", sweep(1400, 2400, 0.10, 0.10))

# Системное — нейтральный одиночный блип.
write("system.wav", note(E6, 0.28, 0.20, harmonics=(1.0, 0.15)))

print("Готово.")
