package api

import "testing"

func TestPresenceRefcount(t *testing.T) {
	p := &presenceTracker{conns: map[int64]int{}}

	// Первое соединение → стал онлайн.
	if !p.connect(7) {
		t.Fatal("первое соединение должно дать переход в онлайн")
	}
	// Второе соединение того же юзера (другая вкладка) → не новый онлайн.
	if p.connect(7) {
		t.Fatal("второе соединение не должно повторно «включать» онлайн")
	}
	// Первое снятие → ещё онлайн (осталась одна вкладка).
	if p.disconnect(7) {
		t.Fatal("при оставшемся соединении не должно быть перехода в офлайн")
	}
	// Второе снятие → офлайн.
	if !p.disconnect(7) {
		t.Fatal("последнее снятие должно дать переход в офлайн")
	}
	if len(p.online()) != 0 {
		t.Fatalf("после офлайна список пуст, получили %v", p.online())
	}
}

func TestPresenceOnlineSnapshot(t *testing.T) {
	p := &presenceTracker{conns: map[int64]int{}}
	p.connect(1)
	p.connect(2)
	p.connect(2)
	got := map[int64]bool{}
	for _, id := range p.online() {
		got[id] = true
	}
	if len(got) != 2 || !got[1] || !got[2] {
		t.Fatalf("ожидал {1,2} онлайн, получил %v", got)
	}
}
