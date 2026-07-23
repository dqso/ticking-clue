package core

import (
	"image/color"

	eimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Shared UI colors for menus and dialogs.
var (
	uiButtonIdleColor    = color.NRGBA{R: 0x3a, G: 0x3a, B: 0x50, A: 0xff}
	uiButtonHoverColor   = color.NRGBA{R: 0x4d, G: 0x4d, B: 0x6a, A: 0xff}
	uiButtonPressedColor = color.NRGBA{R: 0x2b, G: 0x2b, B: 0x3d, A: 0xff}
	uiTextColor          = color.NRGBA{R: 0xf0, G: 0xf0, B: 0xf0, A: 0xff}
	uiPanelColor         = color.NRGBA{R: 0x20, G: 0x20, B: 0x2e, A: 0xff}
	uiOverlayColor       = color.NRGBA{A: 0xa0}
)

// facePtr wraps a text face into a pointer to the interface,
// because ebitenui options expect *text.Face.
func facePtr(size float64) *text.Face {
	f := text.Face(newFace(size))
	return &f
}

// newButtonImage builds a simple flat-color button style.
func newButtonImage() *widget.ButtonImage {
	return &widget.ButtonImage{
		Idle:    eimage.NewNineSliceColor(uiButtonIdleColor),
		Hover:   eimage.NewNineSliceColor(uiButtonHoverColor),
		Pressed: eimage.NewNineSliceColor(uiButtonPressedColor),
	}
}

// newMenuButton creates a standard menu button with a click handler.
func newMenuButton(label string, onClick func()) *widget.Button {
	return widget.NewButton(
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(220, 48),
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			}),
		),
		widget.ButtonOpts.Image(newButtonImage()),
		widget.ButtonOpts.Text(label, facePtr(20), &widget.ButtonTextColor{
			Idle: uiTextColor,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(10)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			onClick()
		}),
	)
}

// newCenteredRoot creates a root container that centers its content on the screen.
func newCenteredRoot(content *widget.Container) *widget.Container {
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	content.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}
	root.AddChild(content)
	return root
}
