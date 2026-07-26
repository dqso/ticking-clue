package core

import (
	"image/color"

	"github.com/ebitenui/ebitenui"
	eimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type OptionsScene struct {
	ui *ebitenui.UI
	// words is the group shared with the main menu, it keeps flying here.
	words *flyingWords
	// settings is the shared player options this scene edits.
	settings *Settings
	// levelButtons are the six CEFR toggle buttons, so the scene can re-check
	// A1 when the player clears the whole selection.
	levelButtons [levelCount]*widget.Button
	// back is set by the Back button and handled on the next Update.
	back bool
}

func newOptionsScene(words *flyingWords, settings *Settings) *OptionsScene {
	s := &OptionsScene{words: words, settings: settings}

	content := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(12),
		)),
	)

	content.AddChild(widget.NewText(
		widget.TextOpts.Text("Options", facePtr(36), uiTextColor),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
	))

	content.AddChild(widget.NewText(
		widget.TextOpts.Text("Choose your level:", facePtr(24), uiTextColor),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
	))

	content.AddChild(s.levelGrid())

	// A small gap between the level buttons and the Back button.
	content.AddChild(widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(0, 16)),
	))

	content.AddChild(newMenuButton("Back", func() {
		s.back = true
	}))

	s.ui = &ebitenui.UI{Container: newCenteredRoot(content)}
	return s
}

// levelGrid builds the two rows of three CEFR toggle buttons (A1 A2 B1 / B2 C1
// C2). Each button is a checkbox: click to enable a level, click again to
// disable it. They sit above the Back button.
func (s *OptionsScene) levelGrid() *widget.Container {
	grid := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			widget.GridLayoutOpts.Spacing(8, 8),
		)),
	)
	for i := range levelLabels {
		lvl := Level(i)
		btn := newLevelButton(levelLabels[i], s.settings.Levels[i], func(on bool) {
			s.setLevel(lvl, on)
		})
		s.levelButtons[i] = btn
		grid.AddChild(btn)
	}
	return grid
}

// setLevel records a level toggle. When the player clears the last enabled
// level, A1 is re-selected so word selection always has at least one level.
func (s *OptionsScene) setLevel(lvl Level, on bool) {
	s.settings.Levels[lvl] = on
	if on {
		return
	}
	for _, enabled := range s.settings.Levels {
		if enabled {
			return
		}
	}
	// Nothing left selected: snap A1 back on (its button follows via SetState).
	s.settings.Levels[LevelA1] = true
	s.levelButtons[LevelA1].SetState(widget.WidgetChecked)
}

// newLevelButton builds one CEFR toggle button, starting checked when the level
// is enabled. onChange fires with the new state each time it is toggled.
func newLevelButton(label string, checked bool, onChange func(on bool)) *widget.Button {
	b := widget.NewButton(
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(72, 44)),
		// A toggle button draws its Pressed image while checked, so a green
		// Pressed marks an enabled level.
		widget.ButtonOpts.Image(&widget.ButtonImage{
			Idle:    eimage.NewNineSliceColor(uiButtonIdleColor),
			Hover:   eimage.NewNineSliceColor(uiButtonHoverColor),
			Pressed: eimage.NewNineSliceColor(uiButtonCheckedColor),
		}),
		widget.ButtonOpts.Text(label, facePtr(20), &widget.ButtonTextColor{Idle: uiTextColor}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(8)),
		widget.ButtonOpts.ToggleMode(),
		widget.ButtonOpts.StateChangedHandler(func(args *widget.ButtonChangedEventArgs) {
			onChange(args.State == widget.WidgetChecked)
		}),
	)
	if checked {
		b.SetState(widget.WidgetChecked)
	}
	return b
}

func (s *OptionsScene) Update(g *Game) error {
	s.ui.Update()
	s.words.handleClick()
	s.words.update(float64(g.screenWidth), float64(g.screenHeight))
	// Esc acts like the Back button.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.back = true
	}
	if s.back {
		s.back = false
		// Persist the level selection when leaving the options screen.
		s.settings.save()
		g.Pop()
	}
	return nil
}

func (s *OptionsScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 0x18, G: 0x18, B: 0x24, A: 0xff})
	s.words.draw(screen)
	s.ui.Draw(screen)
}
