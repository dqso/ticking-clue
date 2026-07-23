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

	// centered adds a text line centered inside the column. The text may
	// contain [link=id]...[/link] BBCode; clicking it opens creditsLinks[id].
	centered := func(txt string, size float64) {
		content.AddChild(widget.NewText(
			widget.TextOpts.Text(txt, facePtr(size), uiTextColor),
			widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionStart),
			widget.TextOpts.ProcessBBCode(true),
			widget.TextOpts.LinkClickedHandler(func(a *widget.LinkEventArgs) {
				if url, ok := creditsLinks[a.Id]; ok {
					openURL(url)
				}
			}),
			widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			})),
		))
	}

	// TODO: write the game description when the gameplay is settled.
	centered("TODO: game description", 18)
	centered("TODO: controls", 18)
	centered("originally created by Denis Proleev ([link=dqso]github.com/dqso[/link])\n"+
		"for GMTK Game Jam 2026, theme: count down", 16)

	centered("Game engine", 24)
	centered("[link=ebitengine]Ebitengine[/link] (github.com/hajimehoshi/ebiten),\n"+
		"licensed under the Apache License 2.0", 16)

	centered("Fonts", 24)
	centered("Fira Sans — © The Mozilla Foundation and Telefonica S.A.,\n"+
		"licensed under the SIL Open Font License 1.1", 16)

	centered("Content", 24)
	centered("Word data is based on [link=wiktionary]Wiktionary[/link] (en.wiktionary.org),\n"+
		"dual-licensed under CC BY-SA 4.0 and GFDL", 16)

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
