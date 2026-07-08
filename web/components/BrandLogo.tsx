// Фирменный знак проекта — золотой щит «Champions Cup» (icon-192 из PWA-набора,
// прозрачный фон). Без круглой обёртки — сам силуэт щита; мягкое золотое
// drop-shadow по контуру даёт премиальный «3D-шильдик» в шапке и на карточках.
export function BrandLogo({ size = 32, className = "" }: { size?: number; className?: string }) {
  return (
    // eslint-disable-next-line @next/next/no-img-element
    <img
      src="/icon-192.png"
      alt="eFootLeague"
      width={size}
      height={size}
      draggable={false}
      className={`flex-shrink-0 select-none drop-shadow-[0_1px_4px_rgba(212,175,55,0.35)] ${className}`}
      style={{ width: size, height: size }}
    />
  );
}
