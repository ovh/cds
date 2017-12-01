+++
draft = false
title = "children"
description = ""

[menu.main]
parent = "shortcodes"
identifier = "children"
+++

This shortcode will list the child pages of a page.

## Usage

| Parameter | Default | Description |
|:--|:--|:--|
| style | "li" | Choose the style used to display descendants. It could be any HTML tag name |
| nohidden | "false" | When true, child pages hidden from the menu will not display |


## Demo

<table>
	<thead>
	<tr>
		<th>{{</* children */>}}</th>
		<th>{{</* children style="h3" */>}}</th>
		<th>{{</* children style="div" nohidden="true" */>}}</th>
	</tr>
	</thead>
	<tbody>
		<tr>
			<td>{{< children />}}</td>
			<td>{{< children style="h3" />}}</td>
			<td>{{< children style="div" nohidden="true"/>}}</td>
		</tr>
	</tbody>
</table>




