// Фирменное написание бренда: eFoot + League вольтовым акцентом.
// Используется везде, где бренд набирается текстом (шапка, логин, интро);
// в canvas-карточках то же самое рисует drawBrandName из lib/shareCards.
export function BrandName({ className = "" }: { className?: string }) {
  return (
    <span className={className}>
      eFoot<span className="text-yellow-400">League</span>
    </span>
  );
}
