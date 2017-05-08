+++
draft = false
title = "Content Organisation"
description = ""

[menu.main]
parent = ""
identifier = "organisation"
weight = 20

+++

With **Hugo**, pages are the core of your site. Organize your site like any other Hugo project. **Magic occurs in the frontmatter of each content**.


# Menu
Hugo has a simple yet powerful [menu system](https://gohugo.io/extras/menus/) that permits content to be placed in menus with a good degree of control without a lot of work, in content pages.

With docdock, **Each content page composes the menu**, they shape the structure of your website.

To link pages to each other : 

* In the frontmatter of each content page ;
	* Set the `parent` identifier.
	* Set the `identifier` of you content


{{%alert info %}} **identifier** should be a unique label thought all your content {{%/alert%}}

{{%alert warning %}} when **parent** is empty, content will be placed at the root level (homepage's child) {{%/alert%}}

In this example "My Dad page" will be attached to root, level 1 menu.

	+++
	title = "My Dad page"

	[menu.main]
	identifier = "daddy"
	+++


In this example "My child page" will be attached to "daddy" page, and displayed as level 2 menu.

	+++
	title = "My child page"

	[menu.main]
	identifier = "child"
	parent="dady"
	+++

### Add icon to a menu entry

in the page frontmatter, add a `pre` param to insert any HTML code before the menu label:

example to display a github icon 

	+++
	[menu.main]
	parent = ""
	identifier = "repo"
	pre ="<i class='fa fa-github'></i> "
	+++

![dsf](/menu-entry-icon.png?height=40px&classes=shadow)

### Customize menu entry label

Add a `name` param next to `[menu.main]`

	+++
	[menu.main]
	parent = ""
	identifier = "repo"
	pre ="<i class='fa fa-github'></i> "
	name = "Github repo"
	+++

### Create a page redirector
Add a `url` param next to `[menu.main]`

	+++
	[menu.main]
	parent = "page"
	identifier = "page-images"
	weight = 23
	url = "/shortcode/image/"
	+++

{{%alert info%}}Look at the menu "Create Page/About images" which redirects to "Shortcodes/image{{%/alert%}}

### Order sibling menu/page entries

in the [menu.main] add `weight` param with a number to order.

	+++
	[menu.main]
	identifier = "child"
	parent="dady"
	weight = 4
	+++


### Hide a menu entry

Do not set identifier to hide a menu entry from... the menu.
You content stays attached to its parent page.

### Folder structure and file name

Content organization is not your folder structure.
Feel free to save your .md file the way your want, it may not necessary reflects your menu organisation. 

### Homepage

Find out how to [customize homepage]({{%relref "homepage.md"%}}) 



