package game

import (
	"strings"
	"testing"
	"time"
)

func TestPlayingAloneWithBots(t *testing.T) {
	botDelay = time.Millisecond
	defer func() { botDelay = 1500 * time.Millisecond }()
	g := New(t.TempDir())
	g.Subscribe("a")
	g.join("a", "Natalia", "#ff00aa")
	// wait until the bots' timers brought the game to the wanted state
	until := func(what string, ok func() bool) {
		t.Helper()
		for range 400 {
			g.mu.Lock()
			done := ok()
			g.mu.Unlock()
			if done {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatalf("bots never got to: %s (mode=%s phase=%s round=%d)", what, g.mode, g.phase, g.round)
	}
	if g.Act("a", "start", "", "") == nil || g.Act("a", "bot", "", "") != nil || !g.players[1].Bot {
		t.Fatal("alone you cannot start; a bot partner fills the seat")
	}
	if !strings.Contains(string(g.view("")), `"online":true,"bot":true`) {
		t.Fatal("a bot counts as present")
	}

	// every mode can be played through alone
	for _, mode := range menu["mode"] {
		if err := g.Act("a", "config", "mode", mode.ID); err != nil {
			t.Fatal(err)
		}
		if err := g.Act("a", "start", "", ""); err != nil {
			t.Fatal(mode.ID, err)
		}
		for round := 1; g.phase != "end"; {
			until(mode.ID+": my turn or a result", func() bool {
				switch {
				case g.phase != "question" || g.round != round:
					return true // revealed, or the round ended without a reveal
				case g.mode == "rozmowy":
					return true
				case g.mode == "wyzwania":
					return g.pending != nil && g.subj == 0 || g.pending == nil && g.subj == 1
				case g.mode == "blef":
					return g.subj == 0 || g.stmts != nil
				case g.mode == "rysowanie":
					return g.subj == 0 || len(g.strokes) > 0
				}
				return true
			})
			g.mu.Lock()
			phase, m, subj, pending, now := g.phase, g.mode, g.subj, g.pending != nil, g.round
			g.mu.Unlock()
			switch {
			case now != round:
				round = now
				continue
			case phase == "reveal":
				all(g, "next")
			case phase == "end":
			case m == "rozmowy":
				all(g, "next")
			case m == "wyzwania" && pending:
				g.Act("a", "answer", "", "1")
			case m == "wyzwania" && subj == 1:
				g.Act("a", "answer", "", "0")
			case m == "wyzwania": // my dare: the bot judges
				until("bot judges", func() bool { return g.round != round || g.phase != "question" })
			case m == "blef" && subj == 0 && g.stmts == nil:
				g.Act("a", "answer", "0", "raz\ndwa\ntrzy")
				until("bot guesses", func() bool { return g.phase != "question" })
			case m == "rysowanie" && subj == 1:
				g.Act("a", "giveup", "", "")
			case m == "rysowanie": // I draw: the bot guesses after a while
				until("bot guesses the drawing", func() bool { return g.phase != "question" })
			case m == "telepatia":
				g.Act("a", "answer", "", "morze")
				until("bot answers", func() bool { return g.phase != "question" })
			default:
				g.Act("a", "answer", "", "0")
				until("bot answers", func() bool { return g.phase != "question" || g.round != round })
			}
			g.mu.Lock()
			round = g.round
			g.mu.Unlock()
		}
		all(g, "quit")
	}
	if len(g.store.history) != 0 {
		t.Fatal("try-outs with a bot must not count in the statistics")
	}

	// a couple of bots to try the ranking alone; a real partner replaces the bot
	if g.Act("a", "botpair", "", "") != nil || len(g.seated()) != 2 || !g.multi() {
		t.Fatal("bot couple")
	}
	g.Act("a", "config", "mode", "wolisz")
	g.Act("a", "start", "", "")
	g.Act("a", "answer", "", "0")
	until("everyone answered", func() bool { return g.phase == "reveal" })
	all(g, "quit")
	if g.Act("a", "botpair", "", "") != nil || len(g.seated()) != 1 {
		t.Fatal("removing the bot couple")
	}
	g.Subscribe("b")
	if g.join("b", "Jan", "#00ffaa") != nil || g.players[1].Bot || g.players[1].Name != "Jan" {
		t.Fatal("a real partner takes the bot's seat")
	}
}
