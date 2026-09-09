package httpapi

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"otklik/internal/domain"
)

func TestPresenceList(t *testing.T) {
	h := newPresenceHub()
	appeal := uuid.New()
	op, expert := uuid.New(), uuid.New()
	h.touch(appeal, op, "operator", domain.RoleOperator, false)
	h.touch(appeal, expert, "psychologist1", domain.RoleExpert, true)

	// Оператор видит только эксперта, с флагом «печатает».
	seen := h.list(appeal, op)
	if len(seen) != 1 {
		t.Fatalf("присутствующих = %d, want 1: %v", len(seen), seen)
	}
	if seen[0]["login"] != "psychologist1" {
		t.Errorf("login = %v", seen[0]["login"])
	}
	if seen[0]["typing"] != true {
		t.Errorf("typing = %v, want true", seen[0]["typing"])
	}
	if seen[0]["role"] != string(domain.RoleExpert) {
		t.Errorf("role = %v", seen[0]["role"])
	}

	// Эксперт видит оператора (без «печатает»).
	other := h.list(appeal, expert)
	if len(other) != 1 || other[0]["login"] != "operator" {
		t.Errorf("эксперт должен видеть оператора: %v", other)
	}

	// Чужое обращение не видно.
	if got := h.list(uuid.New(), op); len(got) != 0 {
		t.Errorf("присутствие утекло в чужое обращение: %v", got)
	}
}

func TestPresenceTTLExpiry(t *testing.T) {
	h := newPresenceHub()
	appeal, u := uuid.New(), uuid.New()
	h.touch(appeal, u, "operator", domain.RoleOperator, false)

	// Имитируем пропавший heartbeat.
	h.mu.Lock()
	key := presenceKey{appeal: appeal, user: u}
	e := h.entries[key]
	e.lastSeen = time.Now().Add(-2 * presenceTTL)
	h.entries[key] = e
	h.mu.Unlock()

	if got := h.list(appeal, uuid.Nil); len(got) != 0 {
		t.Errorf("истёкшее присутствие должно удаляться: %v", got)
	}
}

func TestPresenceOverwrite(t *testing.T) {
	h := newPresenceHub()
	appeal, u := uuid.New(), uuid.New()
	h.touch(appeal, u, "op", domain.RoleOperator, false)
	h.touch(appeal, u, "op", domain.RoleOperator, true)

	h.mu.Lock()
	n := len(h.entries)
	h.mu.Unlock()
	if n != 1 {
		t.Errorf("повторный touch должен обновлять запись, а не создавать: записей %d", n)
	}
}

func TestPresenceConcurrent(t *testing.T) {
	h := newPresenceHub()
	appeal := uuid.New()
	const n = 16
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.touch(appeal, uuid.New(), "user", domain.RoleExpert, false)
		}()
	}
	wg.Wait()
	if got := len(h.list(appeal, uuid.Nil)); got != n {
		t.Errorf("после гонки присутствующих = %d, want %d", got, n)
	}
}
