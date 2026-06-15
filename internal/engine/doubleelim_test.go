package engine

import (
	"testing"
)

func TestDoubleElim_RejectsNonPowerOfTwo(t *testing.T) {
	for _, n := range []int{0, 1, 3, 5, 6, 7, 9, 12} {
		if _, err := DoubleElim(n); err == nil {
			t.Errorf("DoubleElim(%d) должен вернуть ошибку (не степень двойки)", n)
		}
	}
}

func TestDoubleElim_NodeCounts(t *testing.T) {
	// WB = n-1, LB = n-2 (при n≥4, иначе 0), GF = 2 (финал + reset).
	cases := []struct {
		n              int
		wantWB, wantLB int
	}{
		{2, 1, 0},
		{4, 3, 2},
		{8, 7, 6},
		{16, 15, 14},
		{32, 31, 30},
	}
	for _, c := range cases {
		nodes, err := DoubleElim(c.n)
		if err != nil {
			t.Fatalf("DoubleElim(%d): %v", c.n, err)
		}
		var wb, lb, gf int
		for _, nd := range nodes {
			switch nd.Bracket {
			case Winners:
				wb++
			case Losers:
				lb++
			case Grand:
				gf++
			}
		}
		if wb != c.wantWB {
			t.Errorf("n=%d: WB=%d, want %d", c.n, wb, c.wantWB)
		}
		if lb != c.wantLB {
			t.Errorf("n=%d: LB=%d, want %d", c.n, lb, c.wantLB)
		}
		if gf != 2 {
			t.Errorf("n=%d: GF=%d, want 2", c.n, gf)
		}
	}
}

// Каждый сид 1..n должен появиться в WB-раунде 1 ровно один раз.
func TestDoubleElim_SeedsPlacedOnce(t *testing.T) {
	for _, n := range []int{2, 4, 8, 16} {
		nodes, _ := DoubleElim(n)
		seen := make([]bool, n+1)
		for _, nd := range nodes {
			for _, ref := range []Ref{nd.A, nd.B} {
				if ref.Seed > 0 {
					if ref.Seed > n || seen[ref.Seed] {
						t.Fatalf("n=%d: сид %d вне диапазона или дублируется", n, ref.Seed)
					}
					seen[ref.Seed] = true
				}
			}
		}
		for s := 1; s <= n; s++ {
			if !seen[s] {
				t.Errorf("n=%d: сид %d не размещён", n, s)
			}
		}
	}
}

// Каждый проигравший узла верхней сетки должен «опускаться» ровно один раз
// (в нижнюю сетку, либо — для n=2 — в гранд-финал). Иначе игрок выбывал бы
// после одного поражения (нарушение правил двойной элиминации).
func TestDoubleElim_EveryWBLoserDropsExactlyOnce(t *testing.T) {
	for _, n := range []int{2, 4, 8, 16, 32} {
		nodes, _ := DoubleElim(n)
		loserRefCount := map[int]int{}
		wbIDs := map[int]bool{}
		for _, nd := range nodes {
			if nd.Bracket == Winners {
				wbIDs[nd.ID] = true
			}
		}
		for _, nd := range nodes {
			for _, ref := range []Ref{nd.A, nd.B} {
				if ref.Node > 0 && ref.Loser && wbIDs[ref.Node] {
					loserRefCount[ref.Node]++
				}
			}
		}
		for id := range wbIDs {
			if got := loserRefCount[id]; got != 1 {
				t.Errorf("n=%d: проигравший WB-узла %d использован %d раз, want 1", n, id, got)
			}
		}
	}
}

// Все Ref должны ссылаться на существующие, ранее объявленные узлы
// (граф ацикличен и согласован).
func TestDoubleElim_RefsValidAndAcyclic(t *testing.T) {
	for _, n := range []int{2, 4, 8, 16, 32} {
		nodes, _ := DoubleElim(n)
		index := map[int]int{} // node id → позиция в срезе
		for i, nd := range nodes {
			index[nd.ID] = i
		}
		for i, nd := range nodes {
			for _, ref := range []Ref{nd.A, nd.B} {
				if ref.Node == 0 {
					if ref.Seed < 1 {
						t.Errorf("n=%d: узел %d имеет пустой Ref", n, nd.ID)
					}
					continue
				}
				pos, ok := index[ref.Node]
				if !ok {
					t.Errorf("n=%d: узел %d ссылается на несуществующий %d", n, nd.ID, ref.Node)
					continue
				}
				if pos >= i {
					t.Errorf("n=%d: узел %d ссылается на более поздний/себя %d (цикл)", n, nd.ID, ref.Node)
				}
			}
		}
	}
}

// Гранд-финал должен сводить победителя верхней и победителя нижней сетки;
// reset-узел должен существовать и быть помечен.
func TestDoubleElim_GrandFinalWiring(t *testing.T) {
	nodes, _ := DoubleElim(8)
	byID := map[int]Node{}
	var wbFinalID, lbFinalID int
	maxWBRound, maxLBRound := 0, 0
	for _, nd := range nodes {
		byID[nd.ID] = nd
		if nd.Bracket == Winners && nd.Round > maxWBRound {
			maxWBRound, wbFinalID = nd.Round, nd.ID
		}
		if nd.Bracket == Losers && nd.Round > maxLBRound {
			maxLBRound, lbFinalID = nd.Round, nd.ID
		}
	}

	var gf1, gf2 *Node
	for i := range nodes {
		if nodes[i].Bracket == Grand {
			if nodes[i].Round == 1 {
				gf1 = &nodes[i]
			} else {
				gf2 = &nodes[i]
			}
		}
	}
	if gf1 == nil || gf2 == nil {
		t.Fatal("не найдены оба гранд-финала")
	}
	if gf1.A.Node != wbFinalID || gf1.A.Loser {
		t.Errorf("GF1.A должен быть победителем WB-финала (%d), got %+v", wbFinalID, gf1.A)
	}
	if gf1.B.Node != lbFinalID || gf1.B.Loser {
		t.Errorf("GF1.B должен быть победителем LB-финала (%d), got %+v", lbFinalID, gf1.B)
	}
	if !gf2.Reset {
		t.Error("второй гранд-финал должен иметь Reset=true")
	}
	if gf2.A.Node != gf1.ID || gf2.B.Node != gf1.ID {
		t.Error("reset должен сводить участников первого гранд-финала")
	}
}

// n=2: нижней сетки нет, проигравший единственного WB-матча идёт прямо в финал.
func TestDoubleElim_TwoPlayers(t *testing.T) {
	nodes, _ := DoubleElim(2)
	if len(nodes) != 3 { // WB1 + GF1 + reset
		t.Fatalf("n=2: узлов %d, want 3", len(nodes))
	}
	var wbID int
	for _, nd := range nodes {
		if nd.Bracket == Winners {
			wbID = nd.ID
		}
	}
	var gf1 Node
	for _, nd := range nodes {
		if nd.Bracket == Grand && nd.Round == 1 {
			gf1 = nd
		}
	}
	if gf1.A.Node != wbID || gf1.A.Loser {
		t.Errorf("n=2: GF1.A должен быть победителем WB, got %+v", gf1.A)
	}
	if gf1.B.Node != wbID || !gf1.B.Loser {
		t.Errorf("n=2: GF1.B должен быть проигравшим WB, got %+v", gf1.B)
	}
}
