package game

import "fmt"

// The party's money is drawn as five coin piles; the first 60 coins are stacked in a
// fixed order, more only change the number (C: o2d0e; ScummVM: giveCredits).
const (
	coinPiles     = 5
	maxStacked    = 60
	stackCycle    = 12
	creditSteps   = 30 // a gain is counted up in about this many steps
	coinSound     = 101
	creditsPage   = 6
	creditsLeft   = 292
	creditsTop    = 97
	creditsRight  = 316
	creditsBottom = 118
	creditsFill   = 252
	creditsTextX  = 305
	creditsTextY  = 98
	creditsColour = 254
	creditsCopyW  = 25
	creditsCopyH  = 22
	creditsClipW  = 320
	creditsClipH  = 200
	coinPileWidth = 5
)

var (
	coinPileX     = [coinPiles]int{296, 308, 299, 305, 302}             // (C: g_09c1)
	coinPileY     = [coinPiles]int{115, 115, 116, 116, 117}             // (C: g_09cb)
	coinPileOrder = [stackCycle]int{4, 4, 4, 4, 2, 2, 2, 3, 3, 0, 1, 1} // (C: g_09d5)
	coinColours   = [coinPileWidth]int{210, 209, 208, 209, 210}
)

// coinPile is which pile coin n (counting from 0) lands on.
func coinPile(n int) int {
	k := coinPileOrder[n%stackCycle] - n/stackCycle
	if k < 0 {
		k += coinPiles
	}
	return k
}

// drawCredits draws the coin piles and the sum on a page; on page 6 it is then copied to
// the screen (C: ovr_2d0e_0020).
func (g *Game) drawCredits(page int) {
	s := g.Screen
	old := s.Current
	g.clearCredits(page)
	s.Current = page
	s.SetLineClip(0, 0, creditsClipW, creditsClipH)
	for i := range coinPiles {
		if h := g.coinPiles[i]; h != 0 {
			x, y := coinPileX[i], coinPileY[i]
			for k, colour := range coinColours {
				s.DrawLine(x+k, y, x+k, y-(h-1), colour)
			}
		}
	}
	font := g.Text.SetFont(g.SmallFont)
	g.printAt(fmt.Sprintf("%d", g.credits), creditsTextX, creditsTextY, creditsColour, 0, printCentred)
	g.Text.SetFont(font)
	s.Current = old
	if page == creditsPage {
		s.HideMouseIn(creditsLeft, creditsTop, creditsLeft+creditsCopyW, creditsTop+creditsCopyH)
		s.CopyRect(creditsLeft, creditsTop, creditsLeft, creditsTop, creditsCopyW, creditsCopyH, creditsPage, 0)
		s.ShowMouseIn()
	}
}

// clearCredits fills the credits area of a page (C: ovr_2d0e_0025). The original saved
// the current page in a variable it never restored from, leaving the page switched.
func (g *Game) clearCredits(page int) {
	g.Screen.Current = page
	g.Screen.FillRect(creditsLeft, creditsTop, creditsRight, creditsBottom, creditsFill)
}

// recountCredits rebuilds the piles from the sum (C: ovr_2d0e_002a).
func (g *Game) recountCredits() {
	g.coinPiles = [coinPiles]int{}
	n := g.credits
	g.credits = 0
	g.giveCredits(n, false)
}

// giveCredits adds money, with the coin sound and the piles growing step by step if
// animate (C: ovr_2d0e_002f).
func (g *Game) giveCredits(n int, animate bool) {
	if animate {
		g.playSound(coinSound, 0)
	}
	step := max(n/creditSteps, 1)
	for n != 0 {
		step = min(step, n)
		if g.credits < maxStacked {
			for range step {
				if g.credits < maxStacked {
					g.coinPiles[coinPile(g.credits)]++
				}
				g.credits++
			}
		} else {
			g.credits += step
		}
		if animate {
			g.drawCredits(creditsPage)
			if n != 0 {
				g.waitTicks(1)
			}
		}
		n -= step
	}
}

// takeCredits removes money, at most what the party has (C: ovr_2d0e_0034).
func (g *Game) takeCredits(n int, animate bool) {
	if animate {
		g.playSound(coinSound, 0)
	}
	n = min(n, g.credits)
	step := max(n/creditSteps, 1)
	for g.credits > 0 && n != 0 {
		step = min(step, n)
		if g.credits-step < maxStacked {
			for range step {
				g.credits--
				if g.credits < maxStacked {
					g.coinPiles[coinPile(g.credits)]--
				}
			}
		} else {
			g.credits -= step
		}
		if animate {
			g.drawCredits(creditsPage)
			if n != 0 {
				g.waitTicks(1)
			}
		}
		n -= step
	}
}
