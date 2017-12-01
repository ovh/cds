+++
draft = false
title = "Home page"
description = ""
date = "2017-04-28T18:36:24+02:00"
creatordisplayname = "Valere JEANTET"
creatoremail = "valere.jeantet@gmail.com"
lastmodifierdisplayname = "Valere JEANTET"
lastmodifieremail = "valere.jeantet@gmail.com"
tags = ["tag1","tag2"]

[menu.main]
parent = "page"
identifier = "home"
weight = 1

+++

To tell Hugo-theme-docdock to consider a page as homepage's content, just create a content file named `_index.md` in content folder.

{{%panel theme="danger" header="**Homepage consideration**"%}}Do not set [menu.main] in the frontmatter of your _index.md file{{%/panel%}}
