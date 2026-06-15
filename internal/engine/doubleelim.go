package engine

import "fmt"

// Bracket — к какой части сетки двойной элиминации относится матч.
type Bracket int

const (
	Winners Bracket = iota // верхняя сетка
	Losers                 // нижняя сетка (выбывание после второго поражения)
	Grand                  // гранд-финал (+ возможный reset)
)

func (b Bracket) String() string {
	switch b {
	case Winners:
		return "winners"
	case Losers:
		return "losers"
	case Grand:
		return "grand"
	default:
		return "unknown"
	}
}

// Ref — откуда берётся участник матча: либо начальный сид (Seed>0), либо
// результат другого узла (Node>0): победитель, или проигравший при Loser=true.
type Ref struct {
	Seed  int  // >0: начальный сид (1-based)
	Node  int  // >0: id узла-источника
	Loser bool // true: брать проигравшего узла Node (иначе победителя)
}

// FromSeed / FromWinner / FromLoser — конструкторы Ref для читаемости.
func FromSeed(s int) Ref    { return Ref{Seed: s} }
func FromWinner(id int) Ref { return Ref{Node: id} }
func FromLoser(id int) Ref  { return Ref{Node: id, Loser: true} }

// Node — один матч в графе двойной элиминации. A и B — источники участников.
// Победитель/проигравший узла используются другими узлами через Ref.
type Node struct {
	ID      int
	Bracket Bracket
	Round   int // 1-based внутри своей сетки; для Grand: 1 = финал, 2 = reset
	Order   int // 1-based позиция в раунде
	A, B    Ref
	// Reset=true у второго гранд-финала: играется ТОЛЬКО если в первом победил
	// представитель нижней сетки (у верхнего тогда одно поражение — нужен
	// решающий матч). Интеграция решает, активировать ли его.
	Reset bool
}

// DoubleElim строит полный граф матчей двойной элиминации для n участников.
// n должно быть степенью двойки ≥ 2 (вызывающий дополняет bye до степени двойки
// на уровне посева, как в одиночной элиминации).
//
// Структура: верхняя сетка (n-1 матчей), нижняя сетка (n-2 матча при n≥4),
// гранд-финал + условный reset. Сиды расставляются стандартным посевом
// (SeedOrder), проигравшие верхней сетки опускаются в нижнюю.
func DoubleElim(n int) ([]Node, error) {
	if n < 2 || n&(n-1) != 0 {
		return nil, fmt.Errorf("DoubleElim: n=%d, нужна степень двойки ≥ 2", n)
	}

	// L = log2(n) — число раундов верхней сетки.
	L := 0
	for (1 << L) < n {
		L++
	}

	var nodes []Node
	id := 0
	add := func(b Bracket, round, order int, a, bref Ref, reset bool) int {
		id++
		nodes = append(nodes, Node{ID: id, Bracket: b, Round: round, Order: order, A: a, B: bref, Reset: reset})
		return id
	}

	wb := map[[2]int]int{} // (round, order) → node id (верхняя сетка)
	lb := map[[2]int]int{} // (round, order) → node id (нижняя сетка)

	// ── Верхняя сетка ───────────────────────────────────────────────────────
	order := SeedOrder(n)
	for j := 1; j <= n/2; j++ {
		wb[[2]int{1, j}] = add(Winners, 1, j, FromSeed(order[2*j-2]), FromSeed(order[2*j-1]), false)
	}
	for r := 2; r <= L; r++ {
		cnt := n >> r // n / 2^r
		for j := 1; j <= cnt; j++ {
			wb[[2]int{r, j}] = add(Winners, r, j,
				FromWinner(wb[[2]int{r - 1, 2*j - 1}]),
				FromWinner(wb[[2]int{r - 1, 2 * j}]), false)
		}
	}
	wbFinal := wb[[2]int{L, 1}]

	// ── Нижняя сетка ────────────────────────────────────────────────────────
	// Раунды чередуются: нечётный — «minor» (играют survivors нижней сетки),
	// чётный — «major» (survivor нижней встречает проигравшего верхней).
	if L >= 2 {
		totalLB := 2 * (L - 1)
		prevCnt := 0
		for r := 1; r <= totalLB; r++ {
			switch {
			case r == 1:
				// minor: пары из проигравших WB-раунда 1.
				cnt := n / 4
				for j := 1; j <= cnt; j++ {
					lb[[2]int{1, j}] = add(Losers, 1, j,
						FromLoser(wb[[2]int{1, 2*j - 1}]),
						FromLoser(wb[[2]int{1, 2 * j}]), false)
				}
				prevCnt = cnt
			case r%2 == 0:
				// major: проигравший WB-раунда (k+1) против победителя LB(r-1).
				k := r / 2
				cnt := prevCnt
				for j := 1; j <= cnt; j++ {
					lb[[2]int{r, j}] = add(Losers, r, j,
						FromLoser(wb[[2]int{k + 1, j}]),
						FromWinner(lb[[2]int{r - 1, j}]), false)
				}
				prevCnt = cnt
			default:
				// minor (r≥3): пары из победителей LB(r-1).
				cnt := prevCnt / 2
				for j := 1; j <= cnt; j++ {
					lb[[2]int{r, j}] = add(Losers, r, j,
						FromWinner(lb[[2]int{r - 1, 2*j - 1}]),
						FromWinner(lb[[2]int{r - 1, 2 * j}]), false)
				}
				prevCnt = cnt
			}
		}
	}

	// ── Гранд-финал ─────────────────────────────────────────────────────────
	var gfB Ref
	if L >= 2 {
		gfB = FromWinner(lb[[2]int{2 * (L - 1), 1}]) // победитель нижней сетки
	} else {
		gfB = FromLoser(wbFinal) // n=2: нижней сетки нет, проигравший WB-финала
	}
	gf1 := add(Grand, 1, 1, FromWinner(wbFinal), gfB, false)
	// reset: те же двое (победитель и проигравший GF1). Активируется только если
	// GF1 выиграл представитель нижней сетки.
	add(Grand, 2, 1, FromWinner(gf1), FromLoser(gf1), true)

	return nodes, nil
}
