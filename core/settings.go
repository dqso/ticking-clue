package core

import pb "github.com/dqso/ticking-clue/proto/gen"

// Level is a CEFR level index: 0=A1 .. 5=C2.
type Level int

const (
	LevelA1 Level = iota
	LevelA2
	LevelB1
	LevelB2
	LevelC1
	LevelC2
	// levelCount is how many CEFR levels there are.
	levelCount = 6
)

// levelBits maps each Level to its Attributes flag, ordered A1..C2.
var levelBits = [levelCount]pb.Attributes{
	pb.Attributes_LEVEL_A1,
	pb.Attributes_LEVEL_A2,
	pb.Attributes_LEVEL_B1,
	pb.Attributes_LEVEL_B2,
	pb.Attributes_LEVEL_C1,
	pb.Attributes_LEVEL_C2,
}

// levelLabels are the button captions for each Level, ordered A1..C2.
var levelLabels = [levelCount]string{"A1", "A2", "B1", "B2", "C1", "C2"}

// Settings holds the in-session player options. It is not persisted yet: it
// lives on *Game for the lifetime of the process.
type Settings struct {
	// Levels[i] reports whether CEFR level i is enabled for word selection.
	Levels [levelCount]bool
}

// newSettings returns the default settings with every level enabled.
func newSettings() *Settings {
	s := &Settings{}
	for i := range s.Levels {
		s.Levels[i] = true
	}
	return s
}

// enabledLevels returns the levels the player picked. When nothing is selected
// it falls back to A1 only, so word selection always has at least one level.
func (s *Settings) enabledLevels() [levelCount]bool {
	for _, on := range s.Levels {
		if on {
			return s.Levels
		}
	}
	return [levelCount]bool{LevelA1: true}
}
