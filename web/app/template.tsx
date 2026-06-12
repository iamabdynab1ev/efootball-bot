"use client";

import { motion, useReducedMotion } from "framer-motion";

/**
 * Переход между страницами: лёгкий fade + подъём на 4px (180ms).
 * template.tsx перемонтируется на каждую навигацию — полностью совместим
 * со static export. Exit-анимаций в app router нет — и не нужно.
 */
export default function Template({ children }: { children: React.ReactNode }) {
  const reduced = useReducedMotion();
  return (
    <motion.div
      initial={reduced ? false : { opacity: 0, y: 4 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.18, ease: "easeOut" }}
    >
      {children}
    </motion.div>
  );
}
