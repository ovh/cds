// Copyright 2017 Zack Guo <zack.y.guo@gmail.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

// +build ignore

package main

import "github.com/gizak/termui"

func main() {
	err := termui.Init()
	if err != nil {
		panic(err)
	}
	defer termui.Close()

	//termui.UseTheme("helloworld")

	strs := []string{
		"[0] github.com/gizak/termui",
		"[1] [你好，世界](fg-blue)",
		"[2] [こんにちは世界](fg-red)",
		"[3] [color output](fg-white,bg-green)",
		"[4] output.go",
		"[5] random_out.go",
		"[6] dashboard.go",
		"[7] nsf/termbox-go"}

	ls := termui.NewList()
	ls.Items = strs
	ls.ItemFgColor = termui.ColorYellow
	ls.BorderLabel = "List"
	ls.Height = 7
	ls.Width = 25
	ls.Y = 0

	termui.Render(ls)
	termui.Handle("/sys/kbd/q", func(termui.Event) {
		termui.StopLoop()
	})
	termui.Loop()

}
