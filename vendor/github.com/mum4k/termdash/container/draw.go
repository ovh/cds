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

// draw.go contains logic to draw containers and the contained widgets.

import (
	"errors"
	"fmt"
	"image"

	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/internal/area"
	"github.com/mum4k/termdash/internal/canvas"
	"github.com/mum4k/termdash/internal/draw"
	"github.com/mum4k/termdash/widgetapi"
)

// drawTree draws this container and all of its sub containers.
func drawTree(c *Container) error {
	var errStr string

	root := rootCont(c)
	size := root.term.Size()
	ar, err := root.opts.margin.apply(image.Rect(0, 0, size.X, size.Y))
	if err != nil {
		return err
	}
	root.area = ar

	preOrder(root, &errStr, visitFunc(func(c *Container) error {
		first, second, err := c.split()
		if err != nil {
			return err
		}
		if c.first != nil {
			ar, err := c.first.opts.margin.apply(first)
			if err != nil {
				return err
			}
			c.first.area = ar
		}

		if c.second != nil {
			ar, err := c.second.opts.margin.apply(second)
			if err != nil {
				return err
			}
			c.second.area = ar
		}
		return drawCont(c)
	}))
	if errStr != "" {
		return errors.New(errStr)
	}
	return nil
}

// drawBorder draws the border around the container if requested.
func drawBorder(c *Container) error {
	if !c.hasBorder() {
		return nil
	}

	cvs, err := canvas.New(c.area)
	if err != nil {
		return err
	}

	ar, err := area.FromSize(cvs.Size())
	if err != nil {
		return err
	}

	var cOpts []cell.Option
	if c.focusTracker.isActive(c) {
		cOpts = append(cOpts, cell.FgColor(c.opts.inherited.focusedColor))
	} else {
		cOpts = append(cOpts, cell.FgColor(c.opts.inherited.borderColor))
	}

	if err := draw.Border(cvs, ar,
		draw.BorderLineStyle(c.opts.border),
		draw.BorderTitle(c.opts.borderTitle, draw.OverrunModeThreeDot, cOpts...),
		draw.BorderTitleAlign(c.opts.borderTitleHAlign),
		draw.BorderCellOpts(cOpts...),
	); err != nil {
		return err
	}
	return cvs.Apply(c.term)
}

// drawWidget requests the widget to draw on the canvas.
func drawWidget(c *Container) error {
	widgetArea, err := c.widgetArea()
	if err != nil {
		return err
	}
	if widgetArea == image.ZR {
		return nil
	}

	if !c.hasWidget() {
		return nil
	}

	needSize := image.Point{1, 1}
	wOpts := c.opts.widget.Options()
	if wOpts.MinimumSize.X > 0 && wOpts.MinimumSize.Y > 0 {
		needSize = wOpts.MinimumSize
	}

	if widgetArea.Dx() < needSize.X || widgetArea.Dy() < needSize.Y {
		return drawResize(c, c.usable())
	}

	cvs, err := canvas.New(widgetArea)
	if err != nil {
		return err
	}

	meta := &widgetapi.Meta{
		Focused: c.focusTracker.isActive(c),
	}

	if err := c.opts.widget.Draw(cvs, meta); err != nil {
		return err
	}
	return cvs.Apply(c.term)
}

// drawResize draws an unicode character indicating that the size is too small to draw this container.
// Does nothing if the size is smaller than one cell, leaving no space for the character.
func drawResize(c *Container, area image.Rectangle) error {
	if area.Dx() < 1 || area.Dy() < 1 {
		return nil
	}

	cvs, err := canvas.New(area)
	if err != nil {
		return err
	}
	if err := draw.ResizeNeeded(cvs); err != nil {
		return err
	}
	return cvs.Apply(c.term)
}

// drawCont draws the container and its widget.
func drawCont(c *Container) error {
	if us := c.usable(); us.Dx() <= 0 || us.Dy() <= 0 {
		return drawResize(c, c.area)
	}

	if err := drawBorder(c); err != nil {
		return fmt.Errorf("unable to draw container border: %v", err)
	}

	if err := drawWidget(c); err != nil {
		return fmt.Errorf("unable to draw widget %T: %v", c.opts.widget, err)
	}
	return nil
}
