// Copyright 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package container

// options.go defines container options.

import (
	"errors"
	"fmt"
	"image"

	"github.com/mum4k/termdash/align"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/internal/area"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/widgetapi"
)

// applyOptions applies the options to the container and validates them.
func applyOptions(c *Container, opts ...Option) error {
	for _, opt := range opts {
		if err := opt.set(c); err != nil {
			return err
		}
	}
	return nil
}

// ensure all the container identifiers are either empty or unique.
func validateIds(c *Container, seen map[string]bool) error {
	if c.opts.id == "" {
		return nil
	} else if seen[c.opts.id] {
		return fmt.Errorf("duplicate container ID %q", c.opts.id)
	}
	seen[c.opts.id] = true

	return nil
}

// ensure all the container only have one split modifier.
func validateSplits(c *Container) error {
	if c.opts.splitFixed > DefaultSplitFixed && c.opts.splitPercent != DefaultSplitPercent {
		return fmt.Errorf(
			"only one of splitFixed `%v` and splitPercent `%v` is allowed to be set per container",
			c.opts.splitFixed,
			c.opts.splitPercent,
		)
	}

	return nil
}

// validateOptions validates options set in the container tree.
func validateOptions(c *Container) error {
	var errStr string
	seenID := map[string]bool{}
	preOrder(c, &errStr, func(c *Container) error {
		if err := validateIds(c, seenID); err != nil {
			return err
		}
		if err := validateSplits(c); err != nil {
			return err
		}

		return nil
	})
	if errStr != "" {
		return errors.New(errStr)
	}

	return nil
}

// Option is used to provide options to a container.
type Option interface {
	// set sets the provided option.
	set(*Container) error
}

// options stores the options provided to the container.
type options struct {
	// id is the identifier provided by the user.
	id string

	// inherited are options that are inherited by child containers.
	inherited inherited

	// split identifies how is this container split.
	split        splitType
	splitPercent int
	splitFixed   int

	// widget is the widget in the container.
	// A container can have either two sub containers (left and right) or a
	// widget. But not both.
	widget widgetapi.Widget

	// Alignment of the widget if present.
	hAlign align.Horizontal
	vAlign align.Vertical

	// border is the border around the container.
	border            linestyle.LineStyle
	borderTitle       string
	borderTitleHAlign align.Horizontal

	// padding is a space reserved between the outer edge of the container and
	// its content (the widget or other sub-containers).
	padding padding

	// margin is a space reserved on the outside of the container.
	margin margin
}

// margin stores the configured margin for the container.
// For each margin direction, only one of the percentage or cells is set.
type margin struct {
	topCells    int
	topPerc     int
	rightCells  int
	rightPerc   int
	bottomCells int
	bottomPerc  int
	leftCells   int
	leftPerc    int
}

// apply applies the configured margin to the area.
func (p *margin) apply(ar image.Rectangle) (image.Rectangle, error) {
	switch {
	case p.topCells != 0 || p.rightCells != 0 || p.bottomCells != 0 || p.leftCells != 0:
		return area.Shrink(ar, p.topCells, p.rightCells, p.bottomCells, p.leftCells)
	case p.topPerc != 0 || p.rightPerc != 0 || p.bottomPerc != 0 || p.leftPerc != 0:
		return area.ShrinkPercent(ar, p.topPerc, p.rightPerc, p.bottomPerc, p.leftPerc)
	}
	return ar, nil
}

// padding stores the configured padding for the container.
// For each padding direction, only one of the percentage or cells is set.
type padding struct {
	topCells    int
	topPerc     int
	rightCells  int
	rightPerc   int
	bottomCells int
	bottomPerc  int
	leftCells   int
	leftPerc    int
}

// apply applies the configured padding to the area.
func (p *padding) apply(ar image.Rectangle) (image.Rectangle, error) {
	switch {
	case p.topCells != 0 || p.rightCells != 0 || p.bottomCells != 0 || p.leftCells != 0:
		return area.Shrink(ar, p.topCells, p.rightCells, p.bottomCells, p.leftCells)
	case p.topPerc != 0 || p.rightPerc != 0 || p.bottomPerc != 0 || p.leftPerc != 0:
		return area.ShrinkPercent(ar, p.topPerc, p.rightPerc, p.bottomPerc, p.leftPerc)
	}
	return ar, nil
}

// inherited contains options that are inherited by child containers.
type inherited struct {
	// borderColor is the color used for the border.
	borderColor cell.Color
	// focusedColor is the color used for the border when focused.
	focusedColor cell.Color
}

// newOptions returns a new options instance with the default values.
// Parent are the inherited options from the parent container or nil if these
// options are for a container with no parent (the root).
func newOptions(parent *options) *options {
	opts := &options{
		inherited: inherited{
			focusedColor: cell.ColorYellow,
		},
		hAlign:       align.HorizontalCenter,
		vAlign:       align.VerticalMiddle,
		splitPercent: DefaultSplitPercent,
		splitFixed:   DefaultSplitFixed,
	}
	if parent != nil {
		opts.inherited = parent.inherited
	}
	return opts
}

// option implements Option.
type option func(*Container) error

// set implements Option.set.
func (o option) set(c *Container) error {
	return o(c)
}

// SplitOption is used when splitting containers.
type SplitOption interface {
	// setSplit sets the provided split option.
	setSplit(*options) error
}

// splitOption implements SplitOption.
type splitOption func(*options) error

// setSplit implements SplitOption.setSplit.
func (so splitOption) setSplit(opts *options) error {
	return so(opts)
}

// DefaultSplitPercent is the default value for the SplitPercent option.
const DefaultSplitPercent = 50

// DefaultSplitFixed is the default value for the SplitFixed option.
const DefaultSplitFixed = -1

// SplitPercent sets the relative size of the split as percentage of the available space.
// When using SplitVertical, the provided size is applied to the new left
// container, the new right container gets the reminder of the size.
// When using SplitHorizontal, the provided size is applied to the new top
// container, the new bottom container gets the reminder of the size.
// The provided value must be a positive number in the range 0 < p < 100.
// If not provided, defaults to DefaultSplitPercent.
func SplitPercent(p int) SplitOption {
	return splitOption(func(opts *options) error {
		if min, max := 0, 100; p <= min || p >= max {
			return fmt.Errorf("invalid split percentage %d, must be in range %d < p < %d", p, min, max)
		}
		opts.splitPercent = p
		return nil
	})
}

// SplitFixed sets the size of the first container to be a fixed value
// and makes the second container take up the remaining space.
// When using SplitVertical, the provided size is applied to the new left
// container, the new right container gets the reminder of the size.
// When using SplitHorizontal, the provided size is applied to the new top
// container, the new bottom container gets the reminder of the size.
// The provided value must be a positive number in the range 0 <= cells.
// If SplitFixed() is not specified, it defaults to SplitPercent() and its given value.
// Only one of SplitFixed() and SplitPercent() can be specified per container.
func SplitFixed(cells int) SplitOption {
	return splitOption(func(opts *options) error {
		if cells < 0 {
			return fmt.Errorf("invalid fixed value %d, must be in range %d <= cells", cells, 0)
		}
		opts.splitFixed = cells
		return nil
	})
}

// SplitVertical splits the container along the vertical axis into two sub
// containers. The use of this option removes any widget placed at this
// container, containers with sub containers cannot contain widgets.
func SplitVertical(l LeftOption, r RightOption, opts ...SplitOption) Option {
	return option(func(c *Container) error {
		c.opts.split = splitTypeVertical
		c.opts.widget = nil
		for _, opt := range opts {
			if err := opt.setSplit(c.opts); err != nil {
				return err
			}
		}

		if err := c.createFirst(l.lOpts()); err != nil {
			return err
		}
		return c.createSecond(r.rOpts())
	})
}

// SplitHorizontal splits the container along the horizontal axis into two sub
// containers. The use of this option removes any widget placed at this
// container, containers with sub containers cannot contain widgets.
func SplitHorizontal(t TopOption, b BottomOption, opts ...SplitOption) Option {
	return option(func(c *Container) error {
		c.opts.split = splitTypeHorizontal
		c.opts.widget = nil
		for _, opt := range opts {
			if err := opt.setSplit(c.opts); err != nil {
				return err
			}
		}

		if err := c.createFirst(t.tOpts()); err != nil {
			return err
		}

		return c.createSecond(b.bOpts())
	})
}

// ID sets an identifier for this container.
// This ID can be later used to perform dynamic layout changes by passing new
// options to this container. When provided, it must be a non-empty string that
// is unique among all the containers.
func ID(id string) Option {
	return option(func(c *Container) error {
		if id == "" {
			return errors.New("the ID cannot be an empty string")
		}
		c.opts.id = id
		return nil
	})
}

// Clear clears this container.
// If the container contains a widget, the widget is removed.
// If the container had any sub containers or splits, they are removed.
func Clear() Option {
	return option(func(c *Container) error {
		c.opts.widget = nil
		c.first = nil
		c.second = nil
		return nil
	})
}

// PlaceWidget places the provided widget into the container.
// The use of this option removes any sub containers. Containers with sub
// containers cannot have widgets.
func PlaceWidget(w widgetapi.Widget) Option {
	return option(func(c *Container) error {
		c.opts.widget = w
		c.first = nil
		c.second = nil
		return nil
	})
}

// MarginTop sets reserved space outside of the container at its top.
// The provided number is the absolute margin in cells and must be zero or a
// positive integer. Only one of MarginTop or MarginTopPercent can be specified.
func MarginTop(cells int) Option {
	return option(func(c *Container) error {
		if min := 0; cells < min {
			return fmt.Errorf("invalid MarginTop(%d), must be in range %d <= value", cells, min)
		}
		if c.opts.margin.topPerc > 0 {
			return fmt.Errorf("cannot specify both MarginTop(%d) and MarginTopPercent(%d)", cells, c.opts.margin.topPerc)
		}
		c.opts.margin.topCells = cells
		return nil
	})
}

// MarginRight sets reserved space outside of the container at its right.
// The provided number is the absolute margin in cells and must be zero or a
// positive integer. Only one of MarginRight or MarginRightPercent can be specified.
func MarginRight(cells int) Option {
	return option(func(c *Container) error {
		if min := 0; cells < min {
			return fmt.Errorf("invalid MarginRight(%d), must be in range %d <= value", cells, min)
		}
		if c.opts.margin.rightPerc > 0 {
			return fmt.Errorf("cannot specify both MarginRight(%d) and MarginRightPercent(%d)", cells, c.opts.margin.rightPerc)
		}
		c.opts.margin.rightCells = cells
		return nil
	})
}

// MarginBottom sets reserved space outside of the container at its bottom.
// The provided number is the absolute margin in cells and must be zero or a
// positive integer. Only one of MarginBottom or MarginBottomPercent can be specified.
func MarginBottom(cells int) Option {
	return option(func(c *Container) error {
		if min := 0; cells < min {
			return fmt.Errorf("invalid MarginBottom(%d), must be in range %d <= value", cells, min)
		}
		if c.opts.margin.bottomPerc > 0 {
			return fmt.Errorf("cannot specify both MarginBottom(%d) and MarginBottomPercent(%d)", cells, c.opts.margin.bottomPerc)
		}
		c.opts.margin.bottomCells = cells
		return nil
	})
}

// MarginLeft sets reserved space outside of the container at its left.
// The provided number is the absolute margin in cells and must be zero or a
// positive integer. Only one of MarginLeft or MarginLeftPercent can be specified.
func MarginLeft(cells int) Option {
	return option(func(c *Container) error {
		if min := 0; cells < min {
			return fmt.Errorf("invalid MarginLeft(%d), must be in range %d <= value", cells, min)
		}
		if c.opts.margin.leftPerc > 0 {
			return fmt.Errorf("cannot specify both MarginLeft(%d) and MarginLeftPercent(%d)", cells, c.opts.margin.leftPerc)
		}
		c.opts.margin.leftCells = cells
		return nil
	})
}

// MarginTopPercent sets reserved space outside of the container at its top.
// The provided number is a relative margin defined as percentage of the container's height.
// Only one of MarginTop or MarginTopPercent can be specified.
// The value must be in range 0 <= value <= 100.
func MarginTopPercent(perc int) Option {
	return option(func(c *Container) error {
		if min, max := 0, 100; perc < min || perc > max {
			return fmt.Errorf("invalid MarginTopPercent(%d), must be in range %d <= value <= %d", perc, min, max)
		}
		if c.opts.margin.topCells > 0 {
			return fmt.Errorf("cannot specify both MarginTopPercent(%d) and MarginTop(%d)", perc, c.opts.margin.topCells)
		}
		c.opts.margin.topPerc = perc
		return nil
	})
}

// MarginRightPercent sets reserved space outside of the container at its right.
// The provided number is a relative margin defined as percentage of the container's height.
// Only one of MarginRight or MarginRightPercent can be specified.
// The value must be in range 0 <= value <= 100.
func MarginRightPercent(perc int) Option {
	return option(func(c *Container) error {
		if min, max := 0, 100; perc < min || perc > max {
			return fmt.Errorf("invalid MarginRightPercent(%d), must be in range %d <= value <= %d", perc, min, max)
		}
		if c.opts.margin.rightCells > 0 {
			return fmt.Errorf("cannot specify both MarginRightPercent(%d) and MarginRight(%d)", perc, c.opts.margin.rightCells)
		}
		c.opts.margin.rightPerc = perc
		return nil
	})
}

// MarginBottomPercent sets reserved space outside of the container at its bottom.
// The provided number is a relative margin defined as percentage of the container's height.
// Only one of MarginBottom or MarginBottomPercent can be specified.
// The value must be in range 0 <= value <= 100.
func MarginBottomPercent(perc int) Option {
	return option(func(c *Container) error {
		if min, max := 0, 100; perc < min || perc > max {
			return fmt.Errorf("invalid MarginBottomPercent(%d), must be in range %d <= value <= %d", perc, min, max)
		}
		if c.opts.margin.bottomCells > 0 {
			return fmt.Errorf("cannot specify both MarginBottomPercent(%d) and MarginBottom(%d)", perc, c.opts.margin.bottomCells)
		}
		c.opts.margin.bottomPerc = perc
		return nil
	})
}

// MarginLeftPercent sets reserved space outside of the container at its left.
// The provided number is a relative margin defined as percentage of the container's height.
// Only one of MarginLeft or MarginLeftPercent can be specified.
// The value must be in range 0 <= value <= 100.
func MarginLeftPercent(perc int) Option {
	return option(func(c *Container) error {
		if min, max := 0, 100; perc < min || perc > max {
			return fmt.Errorf("invalid MarginLeftPercent(%d), must be in range %d <= value <= %d", perc, min, max)
		}
		if c.opts.margin.leftCells > 0 {
			return fmt.Errorf("cannot specify both MarginLeftPercent(%d) and MarginLeft(%d)", perc, c.opts.margin.leftCells)
		}
		c.opts.margin.leftPerc = perc
		return nil
	})
}

// PaddingTop sets reserved space between container and the top side of its widget.
// The widget's area size is decreased to accommodate the padding.
// The provided number is the absolute padding in cells and must be zero or a
// positive integer. Only one of PaddingTop or PaddingTopPercent can be specified.
func PaddingTop(cells int) Option {
	return option(func(c *Container) error {
		if min := 0; cells < min {
			return fmt.Errorf("invalid PaddingTop(%d), must be in range %d <= value", cells, min)
		}
		if c.opts.padding.topPerc > 0 {
			return fmt.Errorf("cannot specify both PaddingTop(%d) and PaddingTopPercent(%d)", cells, c.opts.padding.topPerc)
		}
		c.opts.padding.topCells = cells
		return nil
	})
}

// PaddingRight sets reserved space between container and the right side of its widget.
// The widget's area size is decreased to accommodate the padding.
// The provided number is the absolute padding in cells and must be zero or a
// positive integer. Only one of PaddingRight or PaddingRightPercent can be specified.
func PaddingRight(cells int) Option {
	return option(func(c *Container) error {
		if min := 0; cells < min {
			return fmt.Errorf("invalid PaddingRight(%d), must be in range %d <= value", cells, min)
		}
		if c.opts.padding.rightPerc > 0 {
			return fmt.Errorf("cannot specify both PaddingRight(%d) and PaddingRightPercent(%d)", cells, c.opts.padding.rightPerc)
		}
		c.opts.padding.rightCells = cells
		return nil
	})
}

// PaddingBottom sets reserved space between container and the bottom side of its widget.
// The widget's area size is decreased to accommodate the padding.
// The provided number is the absolute padding in cells and must be zero or a
// positive integer. Only one of PaddingBottom or PaddingBottomPercent can be specified.
func PaddingBottom(cells int) Option {
	return option(func(c *Container) error {
		if min := 0; cells < min {
			return fmt.Errorf("invalid PaddingBottom(%d), must be in range %d <= value", cells, min)
		}
		if c.opts.padding.bottomPerc > 0 {
			return fmt.Errorf("cannot specify both PaddingBottom(%d) and PaddingBottomPercent(%d)", cells, c.opts.padding.bottomPerc)
		}
		c.opts.padding.bottomCells = cells
		return nil
	})
}

// PaddingLeft sets reserved space between container and the left side of its widget.
// The widget's area size is decreased to accommodate the padding.
// The provided number is the absolute padding in cells and must be zero or a
// positive integer. Only one of PaddingLeft or PaddingLeftPercent can be specified.
func PaddingLeft(cells int) Option {
	return option(func(c *Container) error {
		if min := 0; cells < min {
			return fmt.Errorf("invalid PaddingLeft(%d), must be in range %d <= value", cells, min)
		}
		if c.opts.padding.leftPerc > 0 {
			return fmt.Errorf("cannot specify both PaddingLeft(%d) and PaddingLeftPercent(%d)", cells, c.opts.padding.leftPerc)
		}
		c.opts.padding.leftCells = cells
		return nil
	})
}

// PaddingTopPercent sets reserved space between container and the top side of
// its widget. The widget's area size is decreased to accommodate the padding.
// The provided number is a relative padding defined as percentage of the
// container's height. The value must be in range 0 <= value <= 100.
// Only one of PaddingTop or PaddingTopPercent can be specified.
func PaddingTopPercent(perc int) Option {
	return option(func(c *Container) error {
		if min, max := 0, 100; perc < min || perc > max {
			return fmt.Errorf("invalid PaddingTopPercent(%d), must be in range %d <= value <= %d", perc, min, max)
		}
		if c.opts.padding.topCells > 0 {
			return fmt.Errorf("cannot specify both PaddingTopPercent(%d) and PaddingTop(%d)", perc, c.opts.padding.topCells)
		}
		c.opts.padding.topPerc = perc
		return nil
	})
}

// PaddingRightPercent sets reserved space between container and the right side of
// its widget. The widget's area size is decreased to accommodate the padding.
// The provided number is a relative padding defined as percentage of the
// container's width. The value must be in range 0 <= value <= 100.
// Only one of PaddingRight or PaddingRightPercent can be specified.
func PaddingRightPercent(perc int) Option {
	return option(func(c *Container) error {
		if min, max := 0, 100; perc < min || perc > max {
			return fmt.Errorf("invalid PaddingRightPercent(%d), must be in range %d <= value <= %d", perc, min, max)
		}
		if c.opts.padding.rightCells > 0 {
			return fmt.Errorf("cannot specify both PaddingRightPercent(%d) and PaddingRight(%d)", perc, c.opts.padding.rightCells)
		}
		c.opts.padding.rightPerc = perc
		return nil
	})
}

// PaddingBottomPercent sets reserved space between container and the bottom side of
// its widget. The widget's area size is decreased to accommodate the padding.
// The provided number is a relative padding defined as percentage of the
// container's height. The value must be in range 0 <= value <= 100.
// Only one of PaddingBottom or PaddingBottomPercent can be specified.
func PaddingBottomPercent(perc int) Option {
	return option(func(c *Container) error {
		if min, max := 0, 100; perc < min || perc > max {
			return fmt.Errorf("invalid PaddingBottomPercent(%d), must be in range %d <= value <= %d", perc, min, max)
		}
		if c.opts.padding.bottomCells > 0 {
			return fmt.Errorf("cannot specify both PaddingBottomPercent(%d) and PaddingBottom(%d)", perc, c.opts.padding.bottomCells)
		}
		c.opts.padding.bottomPerc = perc
		return nil
	})
}

// PaddingLeftPercent sets reserved space between container and the left side of
// its widget. The widget's area size is decreased to accommodate the padding.
// The provided number is a relative padding defined as percentage of the
// container's width. The value must be in range 0 <= value <= 100.
// Only one of PaddingLeft or PaddingLeftPercent can be specified.
func PaddingLeftPercent(perc int) Option {
	return option(func(c *Container) error {
		if min, max := 0, 100; perc < min || perc > max {
			return fmt.Errorf("invalid PaddingLeftPercent(%d), must be in range %d <= value <= %d", perc, min, max)
		}
		if c.opts.padding.leftCells > 0 {
			return fmt.Errorf("cannot specify both PaddingLeftPercent(%d) and PaddingLeft(%d)", perc, c.opts.padding.leftCells)
		}
		c.opts.padding.leftPerc = perc
		return nil
	})
}

// AlignHorizontal sets the horizontal alignment for the widget placed in the
// container. Has no effect if the container contains no widget.
// Defaults to alignment in the center.
func AlignHorizontal(h align.Horizontal) Option {
	return option(func(c *Container) error {
		c.opts.hAlign = h
		return nil
	})
}

// AlignVertical sets the vertical alignment for the widget placed in the container.
// Has no effect if the container contains no widget.
// Defaults to alignment in the middle.
func AlignVertical(v align.Vertical) Option {
	return option(func(c *Container) error {
		c.opts.vAlign = v
		return nil
	})
}

// Border configures the container to have a border of the specified style.
func Border(ls linestyle.LineStyle) Option {
	return option(func(c *Container) error {
		c.opts.border = ls
		return nil
	})
}

// BorderTitle sets a text title within the border.
func BorderTitle(title string) Option {
	return option(func(c *Container) error {
		c.opts.borderTitle = title
		return nil
	})
}

// BorderTitleAlignLeft aligns the border title on the left.
func BorderTitleAlignLeft() Option {
	return option(func(c *Container) error {
		c.opts.borderTitleHAlign = align.HorizontalLeft
		return nil
	})
}

// BorderTitleAlignCenter aligns the border title in the center.
func BorderTitleAlignCenter() Option {
	return option(func(c *Container) error {
		c.opts.borderTitleHAlign = align.HorizontalCenter
		return nil
	})
}

// BorderTitleAlignRight aligns the border title on the right.
func BorderTitleAlignRight() Option {
	return option(func(c *Container) error {
		c.opts.borderTitleHAlign = align.HorizontalRight
		return nil
	})
}

// BorderColor sets the color of the border around the container.
// This option is inherited to sub containers created by container splits.
func BorderColor(color cell.Color) Option {
	return option(func(c *Container) error {
		c.opts.inherited.borderColor = color
		return nil
	})
}

// FocusedColor sets the color of the border around the container when it has
// keyboard focus.
// This option is inherited to sub containers created by container splits.
func FocusedColor(color cell.Color) Option {
	return option(func(c *Container) error {
		c.opts.inherited.focusedColor = color
		return nil
	})
}

// splitType identifies how a container is split.
type splitType int

// String implements fmt.Stringer()
func (st splitType) String() string {
	if n, ok := splitTypeNames[st]; ok {
		return n
	}
	return "splitTypeUnknown"
}

// splitTypeNames maps splitType values to human readable names.
var splitTypeNames = map[splitType]string{
	splitTypeVertical:   "splitTypeVertical",
	splitTypeHorizontal: "splitTypeHorizontal",
}

const (
	splitTypeVertical splitType = iota
	splitTypeHorizontal
)

// LeftOption is used to provide options to the left sub container after a
// vertical split of the parent.
type LeftOption interface {
	// lOpts returns the options.
	lOpts() []Option
}

// leftOption implements LeftOption.
type leftOption func() []Option

// lOpts implements LeftOption.lOpts.
func (lo leftOption) lOpts() []Option {
	if lo == nil {
		return nil
	}
	return lo()
}

// Left applies options to the left sub container after a vertical split of the parent.
func Left(opts ...Option) LeftOption {
	return leftOption(func() []Option {
		return opts
	})
}

// RightOption is used to provide options to the right sub container after a
// vertical split of the parent.
type RightOption interface {
	// rOpts returns the options.
	rOpts() []Option
}

// rightOption implements RightOption.
type rightOption func() []Option

// rOpts implements RightOption.rOpts.
func (lo rightOption) rOpts() []Option {
	if lo == nil {
		return nil
	}
	return lo()
}

// Right applies options to the right sub container after a vertical split of the parent.
func Right(opts ...Option) RightOption {
	return rightOption(func() []Option {
		return opts
	})
}

// TopOption is used to provide options to the top sub container after a
// horizontal split of the parent.
type TopOption interface {
	// tOpts returns the options.
	tOpts() []Option
}

// topOption implements TopOption.
type topOption func() []Option

// tOpts implements TopOption.tOpts.
func (lo topOption) tOpts() []Option {
	if lo == nil {
		return nil
	}
	return lo()
}

// Top applies options to the top sub container after a horizontal split of the parent.
func Top(opts ...Option) TopOption {
	return topOption(func() []Option {
		return opts
	})
}

// BottomOption is used to provide options to the bottom sub container after a
// horizontal split of the parent.
type BottomOption interface {
	// bOpts returns the options.
	bOpts() []Option
}

// bottomOption implements BottomOption.
type bottomOption func() []Option

// bOpts implements BottomOption.bOpts.
func (lo bottomOption) bOpts() []Option {
	if lo == nil {
		return nil
	}
	return lo()
}

// Bottom applies options to the bottom sub container after a horizontal split of the parent.
func Bottom(opts ...Option) BottomOption {
	return bottomOption(func() []Option {
		return opts
	})
}
