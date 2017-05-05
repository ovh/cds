+++
draft = false
title = "Installation"
description = ""

[menu.main]
parent = "start"
identifier = "installation"
weight = 1

+++



The following steps are here to help you initialize your new website. If you don’t know Hugo at all, we strongly suggest you to train by following this [great documentation for beginners](https://gohugo.io/overview/quickstart/).


## Create Your Documentation

Hugo provides a `new` command to create a new website.

	$ hugo new site <new_website>

## Install The Theme

Install the **Hugo-theme-docdock** theme by following this 

Switch into the themes directory and download the theme b

	$ cd themes
	$ git clone https://github.com/vjeantet/hugo-theme-docdock.git docdock

Alternatively, you can [{{%icon download%}} download the theme as .zip](https://github.com/vjeantet/hugo-theme-docdock/archive/master.zip) file and extract it in the themes directory

## Basic Configuration

[Follow instructions here]({{%relref "configuration.md"%}})