// Фирменный знак проекта — золотой кубок «Champions Cup» (icon-192 из PWA-набора,
// один файл на всё приложение). Круглый, как клубные эмблемы, с тонкой золотой
// окантовкой и мягким свечением — «3D-шильдик» в шапке и на карточках.
export function BrandLogo({ size = 32, className = "" }: { size?: number; className?: string }) {
  return (
    // eslint-disable-next-line @next/next/no-img-element
    <img
      src="/icon-192.png"
      alt="eFootLeague"
      width={size}
      height={size}
      draggable={false}
      className={`flex-shrink-0 select-none rounded-full ring-1 ring-yellow-500/40 shadow-[0_0_10px_rgba(212,175,55,0.25)] ${className}`}
      style={{ width: size, height: size }}
    />
  );
}
