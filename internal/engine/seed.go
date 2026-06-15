// Package engine содержит чистую (без БД) турнирную математику: посев сетки,
// расстановку bye, тай-брейки. Эти функции детерминированы и покрыты тестами;
// сервисный слой (service/*) только маппит сиды на реальных пользователей и
// пишет результат в репозитории.
package engine

// SeedOrder возвращает порядок сидов для сетки на bracketSize позиций, где
// bracketSize — степень двойки. Классическая «стандартная» расстановка: сильные
// сиды максимально разведены, 1-й и 2-й могут встретиться только в финале.
//
//	SeedOrder(1) = [1]
//	SeedOrder(2) = [1 2]
//	SeedOrder(4) = [1 4 2 3]
//	SeedOrder(8) = [1 8 4 5 2 7 3 6]
//
// Для не-степени-2 округлять должен вызывающий (через NextPowerOf2) — лишние
// позиции становятся bye.
func SeedOrder(bracketSize int) []int {
	if bracketSize <= 1 {
		return []int{1}
	}
	order := []int{1, 2}
	for len(order) < bracketSize {
		n := len(order) * 2
		next := make([]int, 0, n)
		for _, s := range order {
			next = append(next, s, n+1-s)
		}
		order = next
	}
	return order
}

// Matchup — матч первого раунда между двумя сидами (1-based).
type Matchup struct {
	Slot     int // номер слота в первом раунде (1-based)
	HomeSeed int
	AwaySeed int
}

// Bye — сид, проходящий первый раунд без игры и заранее ставящийся в слот
// следующей стадии.
type Bye struct {
	Seed     int
	NextSlot int  // слот следующей стадии, куда ставится сид
	IsHome   bool // в home- или away-сторону слота
}

// FirstRound раскладывает topK реальных участников по сетке размера
// NextPowerOf2(topK). Возвращает реальные матчи первого раунда и bye для
// топ-сидов, когда topK не степень двойки.
//
// Слоты нумеруются по полному размеру сетки, поэтому формула продвижения
// nextSlot=(slot+1)/2, isHome=slot%2==1 остаётся корректной и для bye, и для
// реальных матчей.
func FirstRound(topK int) (matches []Matchup, byes []Bye) {
	if topK < 2 {
		return nil, nil
	}
	size := nextPow2(topK)
	order := SeedOrder(size)

	for k := 1; k <= size/2; k++ {
		home := order[2*(k-1)]
		away := order[2*k-1]
		homeReal := home <= topK
		awayReal := away <= topK

		switch {
		case homeReal && awayReal:
			matches = append(matches, Matchup{Slot: k, HomeSeed: home, AwaySeed: away})
		case homeReal || awayReal:
			// Ровно один реальный сид → bye. Поскольку size = NextPowerOf2(topK),
			// двух bye в одной паре быть не может (topK > size/2).
			seed := home
			if !homeReal {
				seed = away
			}
			byes = append(byes, Bye{
				Seed:     seed,
				NextSlot: (k + 1) / 2,
				IsHome:   k%2 == 1,
			})
		}
	}
	return matches, byes
}

// nextPow2 — локальная копия models.NextPowerOf2, чтобы engine не зависел от
// models (избегаем цикла импортов и держим пакет чисто-вычислительным).
func nextPow2(n int) int {
	if n <= 1 {
		return 1
	}
	p := 1
	for p < n {
		p *= 2
	}
	return p
}
