package services_test

import (
	"testing"

	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/services"
)

func TestGenerateBracket_4Players(t *testing.T) {
	layout, err := services.GenerateBracket(services.BracketSpec{
		TotalPlayers: 4, Min: 2, Max: 4,
		AdvanceMap: map[string]int{"1": 1},
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if layout.TotalRounds != 1 {
		t.Errorf("expected 1 round (final), got %d", layout.TotalRounds)
	}
	if len(layout.Rounds[0].Matches) != 1 {
		t.Errorf("expected 1 match in final, got %d", len(layout.Rounds[0].Matches))
	}
	if layout.Rounds[0].Matches[0].PlayersIn != 4 {
		t.Errorf("expected final of 4 players, got %d", layout.Rounds[0].Matches[0].PlayersIn)
	}
}

func TestGenerateBracket_8Players(t *testing.T) {
	layout, err := services.GenerateBracket(services.BracketSpec{
		TotalPlayers: 8, Min: 2, Max: 4,
		AdvanceMap: map[string]int{"1": 1, "2": 1},
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if layout.TotalRounds < 2 {
		t.Errorf("expected at least 2 rounds, got %d", layout.TotalRounds)
	}
}

func TestGenerateBracket_Determinism(t *testing.T) {
	spec := services.BracketSpec{
		TotalPlayers: 17, Min: 2, Max: 4,
		AdvanceMap: map[string]int{"1": 1, "2": 1, "3": 1},
	}
	a, err := services.GenerateBracket(spec)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b, err := services.GenerateBracket(spec)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(a.Rounds) != len(b.Rounds) {
		t.Fatalf("rounds differ: %d vs %d", len(a.Rounds), len(b.Rounds))
	}
	for i := range a.Rounds {
		if len(a.Rounds[i].Matches) != len(b.Rounds[i].Matches) {
			t.Errorf("round %d matches differ: %d vs %d", i, len(a.Rounds[i].Matches), len(b.Rounds[i].Matches))
		}
	}
}

func TestPRNGShuffle(t *testing.T) {
	a := services.PRNGShuffle(42, 10)
	b := services.PRNGShuffle(42, 10)
	if len(a) != 10 || len(b) != 10 {
		t.Fatalf("len(a)=%d len(b)=%d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("PRNG not stable: a[%d]=%d b[%d]=%d", i, a[i], i, b[i])
		}
	}
	// Different seed → different permutation.
	c := services.PRNGShuffle(43, 10)
	if len(c) == 10 {
		same := true
		for i := range a {
			if a[i] != c[i] {
				same = false
				break
			}
		}
		if same {
			t.Error("different seed produced same permutation")
		}
	}
}