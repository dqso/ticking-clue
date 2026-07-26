package core

import (
	"image/color"
	"math"

	"github.com/ebitenui/ebitenui"
	eimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// CreditsScene shows what the game is about and the credits:
// author, fonts and content licenses.
type CreditsScene struct {
	ui *ebitenui.UI
	// words is the group shared with the main menu, it keeps flying here.
	words *flyingWords
	// back is set by the Back button and handled on the next Update.
	back bool
	// content, scroll and slider are kept to hide the scrolling
	// when the whole content fits into the view.
	content *widget.Container
	scroll  *widget.ScrollContainer
	slider  *widget.Slider
	// texts are every text line, kept so their wrap width can follow the
	// window size (see reflow).
	texts []*widget.Text
}

// creditsLinks maps a [link=id] from the credits markup to its URL.
// The id (not the URL) is kept in the text so the BBCode arg parser
// never has to deal with the colons and slashes of a URL.
var creditsLinks = map[string]string{
	"dqso":       "https://github.com/dqso",
	"ebitengine": "https://github.com/hajimehoshi/ebiten",
	"wiktionary": "https://en.wiktionary.org",
}

func newCreditsScene(words *flyingWords) *CreditsScene {
	s := &CreditsScene{words: words}

	content := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(12),
			widget.RowLayoutOpts.Padding(&widget.Insets{Left: 24, Right: 24, Top: 8, Bottom: 8}),
		)),
	)

	// line adds a left-aligned text line. The text may contain
	// [link=id]...[/link] BBCode; clicking it opens creditsLinks[id]. Long
	// lines wrap on their own: reflow keeps MaxWidth in sync with the window.
	line := func(txt string, size float64) {
		t := widget.NewText(
			widget.TextOpts.Text(txt, facePtr(size), uiTextColor),
			widget.TextOpts.Position(widget.TextPositionStart, widget.TextPositionStart),
			widget.TextOpts.ProcessBBCode(true),
			widget.TextOpts.LinkClickedHandler(func(a *widget.LinkEventArgs) {
				if url, ok := creditsLinks[a.Id]; ok {
					openURL(url)
				}
			}),
			widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionStart,
			})),
		)
		s.texts = append(s.texts, t)
		content.AddChild(t)
	}

	// Short pitch answering the "What is it?" panel title, then the controls.
	// The pitch flows as one paragraph and is wrapped by reflow; the other
	// blocks keep their hand-made breaks for structure and wrap only if narrow.
	line("A word is hidden. Around it grows a graph of related words - "+
		"synonyms, antonyms and more. The time left on the clock is your "+
		"resource: it drains as you play, good guesses top it up and hints "+
		"spend it. Type your guesses: a close word reveals more of the graph "+
		"and buys you time, a wrong one costs it. Name the hidden word before "+
		"the clock counts down. Buy hints to close in - but beware, some of "+
		"them only make time run faster.", 18)

	line("Controls", 26)
	line("Type a word and press Enter to guess, Backspace to erase.\n"+
		"Drag the sheet to pan, drag a word to move it, use [+]/[-] to zoom.\n"+
		"Click a hint cell to buy it. Press Esc to pause.", 18)

	line("Originally created by Denis Proleev ([link=dqso]github.com/dqso[/link]) "+
		"for GMTK Game Jam 2026, theme: count down", 18)

	line("Game engine", 26)
	line("Ebitengine [link=ebitengine]github.com/hajimehoshi/ebiten[/link], "+
		"licensed under the Apache License 2.0", 18)

	line("Fonts", 26)
	line("Fira Sans - © The Mozilla Foundation and Telefonica S.A., "+
		"licensed under the SIL Open Font License 1.1", 18)

	line("Content", 26)
	line("Word data is based on Wiktionary [link=wiktionary]en.wiktionary.org[/link], "+
		"dual-licensed under CC BY-SA 4.0 and GFDL", 18)

	// The content is scrollable: it may not fit on small screens.
	scroll := widget.NewScrollContainer(
		widget.ScrollContainerOpts.Content(content),
		widget.ScrollContainerOpts.StretchContentWidth(),
		widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{
			Idle: eimage.NewNineSliceColor(uiPanelColor),
			Mask: eimage.NewNineSliceColor(uiPanelColor),
		}),
	)

	// pageSize converts the visible part of the content into slider
	// units (the slider range is 0..1000).
	pageSize := func() int {
		h := content.GetWidget().Rect.Dy()
		if h == 0 {
			return 0
		}
		return int(math.Round(float64(scroll.ViewRect().Dy()) / float64(h) * 1000 / 3))
	}

	slider := widget.NewSlider(
		widget.SliderOpts.Orientation(widget.DirectionVertical),
		widget.SliderOpts.MinMax(0, 1000),
		widget.SliderOpts.PageSizeFunc(pageSize),
		widget.SliderOpts.ChangedHandler(func(args *widget.SliderChangedEventArgs) {
			scroll.ScrollTop = float64(args.Slider.Current) / 1000
		}),
		widget.SliderOpts.Images(
			&widget.SliderTrackImage{
				Idle:  eimage.NewNineSliceColor(uiButtonPressedColor),
				Hover: eimage.NewNineSliceColor(uiButtonPressedColor),
			},
			newButtonImage(),
		),
	)

	// The mouse wheel over the scroll area moves the slider too.
	scroll.GetWidget().ScrolledEvent.AddHandler(func(args any) {
		if a, ok := args.(*widget.WidgetScrolledEventArgs); ok {
			slider.Current -= int(math.Round(a.Y * float64(pageSize())))
		}
	})

	scrollArea := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Stretch([]bool{true, false}, []bool{true}),
			widget.GridLayoutOpts.Spacing(8, 0),
		)),
	)
	scrollArea.AddChild(scroll)
	scrollArea.AddChild(slider)

	// panel holds the fixed title, the scroll area and the Back button.
	panel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(eimage.NewNineSliceColor(uiPanelColor)),
		widget.ContainerOpts.WidgetOpts(
			// Hovering the panel sets input.UIHovered, so clicks on it
			// do not spawn new flying words (see flyingWords.handleClick).
			widget.WidgetOpts.TrackHover(true),
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				StretchVertical:    true,
			}),
		),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			// Only the middle row (the scroll area) takes the free height.
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{false, true, false}),
			widget.GridLayoutOpts.Padding(widget.NewInsetsSimple(16)),
			widget.GridLayoutOpts.Spacing(0, 16),
		)),
	)

	title := widget.NewText(
		widget.TextOpts.Text("What is it?", facePtr(36), uiTextColor),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.GridLayoutData{
			HorizontalPosition: widget.GridLayoutPositionCenter,
		})),
	)
	panel.AddChild(title)
	panel.AddChild(scrollArea)

	backBtn := newDialogButton("Back", func() {
		s.back = true
	})
	backBtn.GetWidget().LayoutData = widget.GridLayoutData{
		HorizontalPosition: widget.GridLayoutPositionCenter,
	}
	panel.AddChild(backBtn)

	// The panel is centered and stretched vertically with a margin.
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(40)),
		)),
	)
	root.AddChild(panel)

	// The link color is set through the theme, since the per-widget
	// LinkColor option is not applied in ebitenui 0.7.3.
	theme := &widget.Theme{
		TextTheme: &widget.TextParams{
			LinkColor: &widget.TextLinkColor{
				Idle:  uiLinkColor,
				Hover: uiLinkHoverColor,
			},
		},
	}

	s.content, s.scroll, s.slider = content, scroll, slider
	s.ui = &ebitenui.UI{Container: root, PrimaryTheme: theme}
	return s
}

// reflow keeps every line's wrap width matched to the window, so the text
// re-wraps on resize. The width is derived from the screen, not from the
// content: measuring the content would loop, because the panel sizes itself to
// the widest line, so an unwrapped line would grow the panel and never wrap.
// A cap keeps the lines readable on wide screens.
func (s *CreditsScene) reflow(screenW int) {
	// chrome is the fixed horizontal space around the text (root, panel and
	// content paddings plus the scrollbar column).
	const chrome, maxW = 220, 820
	w := float64(screenW) - chrome
	if w > maxW {
		w = maxW
	}
	if w <= 0 {
		return
	}
	for _, t := range s.texts {
		t.MaxWidth = w
	}
}

// updateScrolling hides the slider and resets the scroll position when
// the whole content fits into the view, so there is nothing to scroll.
func (s *CreditsScene) updateScrolling() {
	fits := s.content.GetWidget().Rect.Dy() <= s.scroll.ViewRect().Dy()
	if fits {
		s.slider.GetWidget().SetVisibility(widget.Visibility_Hide)
		s.slider.Current = 0
		s.scroll.ScrollTop = 0
	} else {
		s.slider.GetWidget().SetVisibility(widget.Visibility_Show)
	}
}

func (s *CreditsScene) Update(g *Game) error {
	s.reflow(g.screenWidth)
	s.updateScrolling()
	s.ui.Update()
	s.words.handleClick()
	s.words.update(float64(g.screenWidth), float64(g.screenHeight))
	// Esc acts like the Back button.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.back = true
	}
	if s.back {
		s.back = false
		g.Pop()
	}
	return nil
}

func (s *CreditsScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 0x18, G: 0x18, B: 0x24, A: 0xff})
	s.words.draw(screen)
	s.ui.Draw(screen)
}
