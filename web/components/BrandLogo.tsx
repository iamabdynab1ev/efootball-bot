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
      className={`flex-shrink-0 select-none [filter:drop-shadow(0_1px_3px_rgba(212,175,55,0.45))_drop-shadow(0_0_10px_rgba(250,204,21,0.22))] ${className}`}
      style={{ width: size, height: size }}
    />
  );
}
