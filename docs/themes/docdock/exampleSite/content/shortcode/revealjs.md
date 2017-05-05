+++
draft = false
title = "revealjs"
slug = "revealjs"
description = "present content as a reveal.js slide"

[menu.main]
parent = "shortcodes"
identifier = "revealjs"
weight = 1
+++

This shortcode will format the enclosed markdow to render it with [reveal.js](http://lab.hakim.se/reveal-js/) at runtime (client-side)

Read more on [revealjs github repo](https://github.com/hakimel/reveal.js/#markdown).

## Usage

`revealjs` can use the following named parameters :

* theme
* transition
* controls
* progress
* history
* center


{{%warning title="Important" %}}Even if the enclosed content is a mardown, use `<` shortcode notation instead of the `%` notation {{%/warning %}}

### Content formating and slide delimiters

[read more on this here]({{% relref "page-slide.md"%}})

## Demo

{{<revealjs theme="moon" progress="true">}}

# In the morning

___


## Getting up

- Turn off alarm
- Get out of bed

___

## Breakfast

- Eat eggs
- Drink coffee

---

# In the evening

___

## Dinner

- Eat spaghetti
- Drink wine

___

## Going to sleep

- Get in bed
- Count sheep

{{</revealjs>}}

## Source :

* [{{%icon "sunglasses" %}} click here to view raw content](https://github.com/vjeantet/hugo-docdock-doc/blob/master/content/shortcode/revealjs.md)


