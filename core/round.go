package core

import (
	"math"
	"math/rand/v2"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"

	pb "github.com/dqso/ticking-clue/proto/gen"
)

// Gameplay tuning. These are kept together and are easy to adjust; several
// of them are placeholders until the balance is decided.
const (
	// roundDurationA/B/C are the times the player starts the round with,
	// chosen by the CEFR group of the highest enabled level (see
	// roundDurationFor).
	roundDurationA = 16 * time.Minute
	roundDurationB = 19 * time.Minute
	roundDurationC = 22 * time.Minute
	// wrongGuessPenalty is lost for an unknown or too far word.
	wrongGuessPenalty = 15 * time.Second
	// guessRewardBase is the reward for a word one link away; words farther
	// away are worth proportionally less (see guessReward). Placeholder.
	guessRewardBase = 25 * time.Second
	// maxGuessDist is the deepest a guessed word may sit from the hidden
	// word to still count as a hit; farther words are treated as a miss.
	maxGuessDist = 4
	// feedbackFlashTime is how long the last guess result stays on screen.
	feedbackFlashTime = 1200 * time.Millisecond
)

// hintTimeScaleStep is how much each "speed" hint (word length, arrow colors)
// multiplies the timer drain rate. Instead of costing a fixed chunk of time,
// those hints make time run faster, and the debuff stacks: two of them give
// 1.5 * 1.5 = 2.25x.
const hintTimeScaleStep = 1.5

// roundState is the lifecycle of a single round.
type roundState int

const (
	roundPlaying roundState = iota
	roundWon
	roundLost
)

// guessResult explains how the last submitted guess was resolved. It drives
// the small feedback line under the input.
type guessResult int

const (
	guessNone guessResult = iota
	guessWin
	guessReward
	guessPenalty
	// guessKnown is a word already shown on the graph or already rejected; it
	// is not taken again and costs nothing.
	guessKnown
)

// revealedNode is one word shown around the hidden one. The shown words form
// a tree growing out of the center: parent is the node this one hangs from
// (-1 means the hidden center), skipped is how many intermediate vertices of
// the shortest path stay hidden on the connecting arrow, and depth is the
// number of arrows between the center and this node (its ring).
type revealedNode struct {
	node   *Node
	parent int
	// path is the shortest path used to reveal this node (path[0] is the
	// hidden center, path[len-1] is node). It is kept so the whole tree can be
	// rebuilt when a later reveal turns out to be an intermediate vertex.
	path    []*Node
	skipped int
	depth   int
	// edge is the relation type of the arrow when it is a single link
	// (skipped == 0); multi-hop arrows keep EDGE_TYPE_UNSPECIFIED.
	edge pb.EdgeType
	// angle is the direction (radians) of this node from the center. Nodes
	// reached through the same first neighbor share one ray, so the direction
	// stays stable; the pixel position is angle plus a depth-based radius.
	angle float64
	// pinned marks a node the player has dragged; px, py are then its
	// position relative to the center in zoom-1 units (pan and zoom excluded)
	// and the auto-layout is skipped for it.
	pinned bool
	px, py float64
	// byHint marks a node revealed by a hint (not guessed by the player); its
	// cloud masks the hidden word's tokens, a self-guessed one shows them.
	byHint bool
}

// round is the state and rules of one game round. It is deliberately free of
// any drawing so the logic can be reasoned about (and later tested) alone.
type round struct {
	graph  *Graph
	hidden *Node

	// levels are the CEFR levels the player enabled. The link hint mostly
	// reveals neighbors of these levels (see linkPreferredChance).
	levels [levelCount]bool

	// links are the direct neighbors of the hidden word (deduplicated, with
	// a non-empty word). They are the pool for the "reveal a link" hint.
	links []*Node
	// nextLink is the neighbor the link hint will reveal next. It is pre-rolled
	// (see pickLink) so the hint cell can show that word's level in advance.
	nextLink *Node
	// revealed holds every shown word in reveal order, so ring positions
	// stay stable as more words appear.
	revealed []revealedNode
	// revealedSet maps a shown node id to its index in revealed, used both
	// for deduplication and to attach far guesses to an existing node.
	revealedSet map[int64]int
	// rayAngle keeps a stable direction per first-hop neighbor id, so every
	// word reached through that neighbor grows along the same ray.
	rayAngle map[int64]float64

	// remaining is the time left in the round. It is the player's resource:
	// it drains every tick, guesses add to it and speed hints spend it faster,
	// and the round is lost once it hits zero.
	remaining time.Duration
	state     roundState
	// surrendered marks a loss caused by the Surrender button rather than the
	// clock, so the end screen can tell the two apart.
	surrendered bool

	// timeScale multiplies how fast the timer drains. It starts at 1; each
	// "speed" hint (word length, arrow colors) multiplies it by
	// hintTimeScaleStep, so those hints make time run faster instead of costing
	// a fixed chunk. The debuff stacks (see hintTimeScaleStep).
	timeScale float64

	// lengthShown reports that the word-length hint was bought.
	lengthShown bool
	// colorsShown reports that the arrow-color hint was bought.
	colorsShown bool
	// posShown reports that the parts-of-speech hint was bought.
	posShown bool
	// revealedLetters are the letters opened by the "reveal a letter" hint;
	// every position of such a letter is shown instead of a box.
	revealedLetters map[rune]bool
	// hiddenTokens are the significant tokens of the hidden word (stopwords
	// removed); hint clouds mask any of these that show up in them.
	hiddenTokens map[string]struct{}

	// guess is the word currently being typed.
	guess string

	// lastResult / lastReason / lastDelta / flash describe the feedback of the
	// last guess: what happened, why, the time change, and how long it shows.
	lastResult guessResult
	lastReason string
	lastDelta  time.Duration
	flash      time.Duration
}

func newRound(graph *Graph, hidden *Node, levels [levelCount]bool) *round {
	r := &round{
		graph:           graph,
		hidden:          hidden,
		levels:          levels,
		remaining:       roundDurationFor(levels),
		revealedSet:     make(map[int64]int),
		rayAngle:        make(map[int64]float64),
		revealedLetters: make(map[rune]bool),
		hiddenTokens:    significantTokens(hidden.Word),
		timeScale:       1,
	}
	r.links = directNeighbors(hidden)
	return r
}

// roundDurationFor returns the starting time picked by the CEFR group of the
// highest enabled level: A (A1, A2), B (B1, B2) or C (C1, C2). With no level
// enabled it falls back to group A.
func roundDurationFor(levels [levelCount]bool) time.Duration {
	max := -1
	for lvl := levelCount - 1; lvl >= 0; lvl-- {
		if levels[lvl] {
			max = lvl
			break
		}
	}
	switch {
	case max >= int(LevelC1):
		return roundDurationC
	case max >= int(LevelB1):
		return roundDurationB
	default:
		return roundDurationA
	}
}

// directNeighbors returns the distinct neighbors of n that have a word,
// skipping self loops and duplicates coming from several edge types.
func directNeighbors(n *Node) []*Node {
	seen := make(map[int64]struct{}, len(n.Links))
	res := make([]*Node, 0, len(n.Links))
	for _, l := range n.Links {
		to := l.To
		if to.Word == "" || to.ID == n.ID {
			continue
		}
		if _, ok := seen[to.ID]; ok {
			continue
		}
		seen[to.ID] = struct{}{}
		res = append(res, to)
	}
	return res
}

// revealStartWords reveals up to n random neighbors of the hidden word for free
// at the start of the round and returns them, so the opening Notes entry can
// list them. Neighbors of the player's enabled levels are picked first.
func (r *round) revealStartWords(n int) []*Node {
	var preferred, others []*Node
	for _, nb := range r.links {
		if _, ok := r.revealedSet[nb.ID]; ok {
			continue
		}
		if r.levels[nb.MaxLevel()] {
			preferred = append(preferred, nb)
		} else {
			others = append(others, nb)
		}
	}
	rand.Shuffle(len(preferred), func(i, j int) { preferred[i], preferred[j] = preferred[j], preferred[i] })
	rand.Shuffle(len(others), func(i, j int) { others[i], others[j] = others[j], others[i] })
	pool := append(preferred, others...)
	if len(pool) > n {
		pool = pool[:n]
	}
	for _, nb := range pool {
		r.revealPath([]*Node{r.hidden, nb}, true)
	}
	return pool
}

// tickDuration is the real time one Update tick stands for. Counting ticks
// (instead of wall clock) keeps the timer frozen while the pause dialog is
// open, because the game scene stops receiving Update there.
func tickDuration() time.Duration {
	tps := ebiten.TPS()
	if tps <= 0 {
		tps = 60
	}
	return time.Second / time.Duration(tps)
}

// update advances the timer by one tick and ends the round on timeout.
func (r *round) update() {
	if r.state != roundPlaying {
		return
	}
	if r.flash > 0 {
		r.flash -= tickDuration()
	}
	// Drain faster while a speed hint is active; timeScale is 1 by default.
	r.remaining -= time.Duration(float64(tickDuration()) * r.timeScale)
	if r.remaining <= 0 {
		r.remaining = 0
		r.state = roundLost
	}
}

// surrender ends the round as a loss right away (the Surrender button). The
// clock is zeroed so the HUD and the end screen agree the round is over.
func (r *round) surrender() {
	if r.state != roundPlaying {
		return
	}
	r.remaining = 0
	r.surrendered = true
	r.state = roundLost
}

// typeRune appends a typed letter to the current guess.
func (r *round) typeRune(ch rune) {
	if r.state != roundPlaying {
		return
	}
	// Accept letters plus the few separators lemmas use: space, hyphen,
	// apostrophe, backtick and comma.
	if !unicode.IsLetter(ch) && !strings.ContainsRune(" -'`,", ch) {
		return
	}
	r.guess += string(unicode.ToLower(ch))
}

// backspace removes the last character of the current guess.
func (r *round) backspace() {
	if r.state != roundPlaying {
		return
	}
	rs := []rune(r.guess)
	if len(rs) > 0 {
		r.guess = string(rs[:len(rs)-1])
	}
}

// guessOutcome reports how submit resolved a guess so the scene can animate it:
// a reward flies the word to node's place; a penalty flies it to the miss list.
type guessOutcome struct {
	kind guessResult
	word string
	node *Node // the guessed graph node on a reward; nil otherwise
	// strikes is how many times the word is crossed out in the miss list:
	// 2 when it is not in the dictionary, 1 when it is known but too far.
	strikes int
}

// submit resolves the current guess, clears the input, and returns the outcome.
func (r *round) submit() guessOutcome {
	if r.state != roundPlaying {
		return guessOutcome{kind: guessNone}
	}
	word := strings.TrimSpace(strings.ToLower(r.guess))
	r.guess = ""
	if word == "" {
		return guessOutcome{kind: guessNone}
	}
	// Tokens the player types are no longer masked in hint clouds.
	r.unmaskTokens(word)
	if sameWord(word, r.hidden.Word) {
		r.state = roundWon
		r.setFeedback(guessWin, "correct!", 0)
		return guessOutcome{kind: guessWin, word: word}
	}
	node := r.graph.ByWord(word)
	if node == nil {
		r.miss("unknown word")
		return guessOutcome{kind: guessPenalty, word: word, strikes: 2}
	}
	path := r.graph.shortestPath(r.hidden, node, maxGuessDist)
	if path == nil {
		// Known word, but not close enough to the hidden one.
		r.miss("too far")
		return guessOutcome{kind: guessPenalty, word: word, strikes: 1}
	}
	// Show the guessed word (and, when far, its skipped path), then reward.
	r.revealPath(path, false)
	reward := rewardForDistance(len(path) - 1)
	r.applyDelta(reward)
	r.setFeedback(guessReward, "found", reward)
	return guessOutcome{kind: guessReward, word: word, node: node}
}

// miss applies the wrong guess penalty and its feedback with the given reason.
func (r *round) miss(reason string) {
	r.applyDelta(-wrongGuessPenalty)
	r.setFeedback(guessPenalty, reason, -wrongGuessPenalty)
}

// unmaskTokens removes the given word's tokens from the hidden mask set, so a
// token the player has typed is no longer hidden inside hint clouds. It reports
// whether any token was actually masked before, i.e. something got revealed.
func (r *round) unmaskTokens(word string) bool {
	unmasked := false
	for _, tok := range strings.FieldsFunc(word, isTokenSep) {
		tok = strings.ToLower(tok)
		if _, masked := r.hiddenTokens[tok]; masked {
			delete(r.hiddenTokens, tok)
			unmasked = true
		}
	}
	return unmasked
}

// revealedWord is the text shown for a revealed node, used for both sizing and
// drawing so the cloud always fits: hint clouds mask the hidden word's tokens
// (turning them into "<?>"), self-guessed ones show the word unchanged.
func (r *round) revealedWord(rn revealedNode) string {
	if rn.byHint {
		return maskWord(rn.node.Word, r.hiddenTokens)
	}
	return rn.node.Word
}

// rewardForDistance returns the time gained for a word at graph distance d.
// Placeholder: closer words are worth more. TODO: tune the formula.
func rewardForDistance(d int) time.Duration {
	if d < 1 {
		d = 1
	}
	return guessRewardBase / time.Duration(d)
}

// revealPath shows the last node of the given shortest path (path[0] is the
// hidden center). The node is stored with its path and the whole tree is then
// rebuilt, so a node revealed on an already shown branch slots into it. A node
// already shown is left untouched.
func (r *round) revealPath(path []*Node, byHint bool) {
	if len(path) < 2 {
		return
	}
	dst := path[len(path)-1]
	if _, ok := r.revealedSet[dst.ID]; ok {
		return
	}
	r.revealedSet[dst.ID] = len(r.revealed)
	r.revealed = append(r.revealed, revealedNode{node: dst, path: path, byHint: byHint})
	r.rebuild()
}

// rebuild recomputes every revealed node's parent, depth, skipped count, edge
// and angle from the stored paths. Each node hangs from the deepest already
// revealed vertex on its path, so revealing an intermediate word re-parents the
// words beyond it. Directions come from the first hop of the path (the ray), so
// words reached through the same neighbor line up instead of scattering.
func (r *round) rebuild() {
	// Process nodes shortest-path-first so a parent is always handled before
	// its children (a parent has a strictly shorter path).
	order := make([]int, len(r.revealed))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return len(r.revealed[order[a]].path) < len(r.revealed[order[b]].path)
	})

	childCount := make(map[int]int) // children already fanned per parent index
	rayCount := make(map[int64]int) // center children already fanned per ray

	for _, i := range order {
		rn := &r.revealed[i]
		p := rn.path
		// Deepest already-revealed vertex strictly between the center and this
		// node; the rest of the path stays hidden and is counted as skipped.
		parentIdx, attach := -1, 0
		for j := 1; j < len(p)-1; j++ {
			if idx, ok := r.revealedSet[p[j].ID]; ok {
				parentIdx, attach = idx, j
			}
		}
		rn.parent = parentIdx
		rn.skipped = (len(p) - 1) - attach - 1
		attachNode := r.hidden
		rn.depth = 1
		if parentIdx >= 0 {
			rn.depth = r.revealed[parentIdx].depth + 1
			attachNode = r.revealed[parentIdx].node
		}
		// A single relation type only when nothing is skipped on the arrow.
		rn.edge = pb.EdgeType_EDGE_TYPE_UNSPECIFIED
		if rn.skipped == 0 {
			rn.edge = attachNode.LinkType(rn.node)
		}
		// Direction: nodes hanging from the center take their ray angle (keyed
		// by the first hop); deeper nodes fan around their parent's angle.
		if parentIdx < 0 {
			ray := p[1].ID
			rn.angle = fanAngle(r.rayAngleFor(ray), rayCount[ray])
			rayCount[ray]++
		} else {
			rn.angle = fanAngle(r.revealed[parentIdx].angle, childCount[parentIdx])
			childCount[parentIdx]++
		}
	}
}

// fanAngle offsets the k-th sibling around a base direction: 0, +step, -step,
// +2·step, -2·step, ... so branches that diverge spread out evenly.
func fanAngle(base float64, k int) float64 {
	const step = 0.5
	if k == 0 {
		return base
	}
	mag := float64((k+1)/2) * step
	if k%2 == 0 {
		mag = -mag
	}
	return base + mag
}

// rayAngleFor returns the stable direction of the ray for the given first-hop
// neighbor, assigning it the widest free gap the first time it appears.
func (r *round) rayAngleFor(firstHop int64) float64 {
	if a, ok := r.rayAngle[firstHop]; ok {
		return a
	}
	a := r.nextRayAngle()
	r.rayAngle[firstHop] = a
	return a
}

// nextRayAngle returns the middle of the widest gap between existing rays, so a
// new ray appears where there is the most free space.
func (r *round) nextRayAngle() float64 {
	if len(r.rayAngle) == 0 {
		return -math.Pi / 2 // first ray goes straight up (12 o'clock)
	}
	angs := make([]float64, 0, len(r.rayAngle))
	for _, a := range r.rayAngle {
		angs = append(angs, a)
	}
	sort.Float64s(angs)
	// Start with the wrap-around gap between the last and the first angle.
	bestStart, bestGap := angs[len(angs)-1], (angs[0]+2*math.Pi)-angs[len(angs)-1]
	for i := 1; i < len(angs); i++ {
		if g := angs[i] - angs[i-1]; g > bestGap {
			bestStart, bestGap = angs[i-1], g
		}
	}
	return bestStart + bestGap/2
}

// pinCloud fixes a revealed node at the given position relative to the center
// (in zoom-1 units), so the player can arrange the map by hand.
func (r *round) pinCloud(i int, x, y float64) {
	if i < 0 || i >= len(r.revealed) {
		return
	}
	r.revealed[i].pinned = true
	r.revealed[i].px, r.revealed[i].py = x, y
}

// applyDelta changes the remaining time and ends the round when it runs out.
func (r *round) applyDelta(d time.Duration) {
	r.remaining += d
	if r.remaining <= 0 {
		r.remaining = 0
		r.state = roundLost
	}
}

// setFeedback stores the result, reason and time change of the last guess for
// the feedback line under the input.
func (r *round) setFeedback(res guessResult, reason string, delta time.Duration) {
	r.lastResult = res
	r.lastReason = reason
	r.lastDelta = delta
	r.flash = feedbackFlashTime
}
