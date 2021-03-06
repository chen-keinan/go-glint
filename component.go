package glint

import (
	"context"

	"github.com/mitchellh/go-glint/internal/layout"
)

// Components are the individual items that are rendered within a document.
type Component interface {
	// Body returns the body of this component. This can be another custom
	// component or a standard component such as Text.
	//
	// The context parameter is used to send parameters across multiple
	// components. It should not be used for timeouts; Body should aim to
	// not block ever since this can block the render loop.
	//
	// Components are highly encouraged to support finalization (see
	// ComponentFinalizer). Components can finalize early by wrapping
	// their body in a Finalize built-in component. Finalization allows
	// the renderer to highly optimize output.
	Body(context.Context) Component
}

// ComponentFinalizer allows components to be notified they are going to
// be finalized. A finalized component may never be re-rendered again. The
// next call to Body should be considered the final call.
//
// In a Document, if the component list has a set of finalized components
// at the front, the renderer will draw it once and only re-draw non-finalized
// components. For example, consider a document that is a set of text components
// followed by a progress bar. If the text components are static, then they
// will be written to the output once and only the progress bar will redraw.
//
// Currently, Body may be called multiple times after Finalize. Implementers
// should return the same result after being finalized.
type ComponentFinalizer interface {
	Component

	// Finalize notifies the component that it will be finalized. This may
	// be called multiple times.
	Finalize()
}

// componentLayout can be implemented to set custom layout settings
// for the component. This can only be implemented by internal components
// since we use an internal library.
//
// End users should use the "Layout" component to set layout options.
type componentLayout interface {
	Component

	// Layout should return the layout settings for this component.
	Layout() *layout.Builder
}

// terminalComponent is an embeddable struct for internal usage that
// satisfies Component. This is used since terminal components are handled
// as special cases.
type terminalComponent struct{}

func (terminalComponent) Body(context.Context) Component { return nil }
