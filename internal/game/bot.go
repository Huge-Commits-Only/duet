package game

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Bots stand in for people, so the game can be tried out alone: one as your partner,
// two more as a second couple. They answer at random after a short pause.
// Games with a bot are not written to the history.

// botDelay is how long a bot "thinks". A var so tests can shorten it.
var botDelay = 1500 * time.Millisecond

const botColor = "#8a8fa3"

func (g *Game) hasBots() bool {
	for _, p := range g.playing {
		for _, pl := range p.players {
			if pl.Bot {
				return true
			}
		}
	}
	return false
}

// toggleBot gives a player who sits alone a bot for a partner, or takes that bot away.
func (g *Game) toggleBot(p *pair) error {
	if i := slices.IndexFunc(p.players, func(pl *player) bool { return pl.Bot }); i >= 0 {
		p.players = slices.Delete(p.players, i, i+1)
		p.stakes = [2]string{}
		return nil
	}
	if len(p.players) != 1 {
		return ErrBad
	}
	n := slices.Index(g.pairs[:], p)
	p.players = append(p.players, &player{ID: fmt.Sprintf("bot:%d:1", n), Name: "Bot", Color: botColor, Bot: true})
	return nil
}

// toggleBotPair seats two bots as another couple, or removes every all-bot couple.
func (g *Game) toggleBotPair() error {
	allBots := func(p *pair) bool {
		return len(p.players) > 0 && !slices.ContainsFunc(p.players, func(pl *player) bool { return !pl.Bot })
	}
	removed := false
	for _, p := range g.pairs {
		if allBots(p) {
			p.players, removed = nil, true
		}
	}
	if removed {
		return nil
	}
	for n, p := range g.pairs {
		if len(p.players) == 0 {
			for seat, name := range []string{"Bot Ala", "Bot Olek"} {
				p.players = append(p.players, &player{ID: fmt.Sprintf("bot:%d:%d", n, seat), Name: name, Color: botColor, Bot: true})
			}
			return nil
		}
	}
	return ErrFull
}

// botMove is what the bot at seat me of couple p does next, and after how long; act "" = nothing to do.
func (g *Game) botMove(p *pair, me int) (act, key, value string, wait time.Duration) {
	if g.phase != "question" {
		return
	}
	pick := func(n int) string { return strconv.Itoa(rand.IntN(n)) }
	switch g.mode {
	case "rozmowy":
		return // the person moves the cards on
	case "rysowanie":
		if me != g.subj { // let the person draw for a while, then "recognise" it
			return "answer", "", g.cur.Q, 8 * botDelay
		}
		if len(g.strokes) == 0 { // a bot cannot draw the word, but it shows that drawings arrive
			pts := make([]string, 0, 64)
			for i := 0; i <= 31; i++ {
				pts = append(pts, strconv.Itoa(500+int(300*float64(i%2*2-1)*float64(31-i)/31)), strconv.Itoa(200+i*20))
			}
			return "draw", "new", strings.Join(pts, ","), botDelay
		}
		return
	case "wyzwania":
		switch {
		case g.pending != nil && me == g.subj:
			return "answer", "", pick(2), botDelay
		case g.pending == nil && me != g.subj: // time for the person to actually do it
			return "answer", "", "0", 7 * botDelay
		}
		return
	case "blef":
		switch {
		case g.stmts == nil && me == g.subj:
			return "answer", pick(3), "Byłem kiedyś w Japonii\nUmiem żonglować trzema piłkami\nBoję się motyli", botDelay
		case g.stmts != nil && me != g.subj && p.answers[me] < 0:
			return "answer", "", pick(3), 3 * botDelay
		}
		return
	}
	if p.answers[me] >= 0 {
		return
	}
	switch g.mode {
	case "telepatia":
		if other := p.texts[1-me]; other != "" && rand.IntN(2) == 0 {
			return "answer", "", other, botDelay // now and then the bot "reads your mind"
		}
		return "answer", "", "słońce", botDelay
	case "skala":
		return "answer", "", pick(101), botDelay
	case "znasz":
		return "answer", "", pick(len(g.cur.Options)), botDelay
	}
	return "answer", "", pick(2), botDelay
}

// botsThink schedules the next move of every bot that has one. Caller holds mu.
func (g *Game) botsThink() {
	for _, p := range g.playing {
		for me, pl := range p.players {
			if !pl.Bot {
				continue
			}
			act, _, _, wait := g.botMove(p, me)
			// one timer per situation: the same question may be looked at many times while others act
			stage := fmt.Sprint(g.epoch, g.round, g.pending != nil, g.stmts != nil, len(g.strokes) == 0)
			if act == "" || g.botStage[pl.ID] == stage {
				continue
			}
			g.botStage[pl.ID] = stage
			id, epoch, round := pl.ID, g.epoch, g.round
			time.AfterFunc(wait, func() {
				g.mu.Lock()
				defer g.mu.Unlock()
				if g.epoch != epoch || g.round != round { // the game moved on while the bot was thinking
					return
				}
				p, me := g.find(id)
				if p == nil {
					return
				}
				if act, key, value, _ := g.botMove(p, me); act != "" && g.do(id, act, key, value) == nil {
					g.botsThink()
					g.save()
					g.broadcast()
				}
			})
		}
	}
}
