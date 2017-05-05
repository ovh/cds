+++
draft = false
title = "Home"
description = ""
date = "2017-04-24T18:36:24+02:00"


creatordisplayname = "Valere JEANTET"
creatoremail = "valere.jeantet@gmail.com"
lastmodifierdisplayname = "Valere JEANTET"
lastmodifieremail = "valere.jeantet@gmail.com"

+++

<span id="sidebar-toggle-span">
<a href="#" id="sidebar-toggle" data-sidebar-toggle=""><i class="fa fa-bars"></i></a>
</span>


# Hugo docDock theme documentation
[Hugo-theme-docdock {{%icon fa-github%}}](https://github.com/vjeantet/hugo-theme-docdock) is a theme for Hugo, a fast and modern static website engine written in Go. Where Hugo is often used for blogs, this theme is fully designed for documentation.

This theme is a partial porting of the [Learn theme of matcornic {{%icon fa-github%}}](https://github.com/matcornic/hugo-theme-learn), a modern flat-file CMS written in PHP.

This current documentation has been statically generated with Hugo with a simple command : `hugo -t docdock` -- source code is [available here at GitHub {{%icon fa-github%}}](https://github.com/vjeantet/hugo-theme-docDock-doc)



{{% panel theme="success" header="Automated deployments" footer="Netlify builds, deploys, and hosts  frontends." %}}
The current documentation is automatically published thanks to [Netlify](https://www.netlify.com/).
Read more about [Automated deployments with Wercker on gohugo.io](https://gohugo.io/tutorials/automated-deployments/)
{{% /panel %}}

## The Dodock theme
This theme support a page tree structure to display and organize pages.

{{%panel%}}**content organization** : All contents are pages which belong to other pages. [read more about this]({{%relref "organisation.md"%}}) {{%/panel%}}

## Features
Here are the main features :

* [Search]({{%relref "search.md" %}})
* **Unlimited menu levels**
* [Generate RevealJS presentation]({{%relref "page-slide.md"%}}) from markdown (embededed or fullscreen page)
* [Attachments files]({{%relref "shortcode/attachments.md" %}})
* [List child pages]({{%relref "shortcode/children.md" %}})
* [Excerpt]({{%relref "shortcode/excerpt.md"%}}) ! Include segment of content from one page in another
* Automatic next/prev buttons to navigate through menu entries
* [Mermaid diagram]({{%relref "shortcode/mermaid.md" %}}) (flowchart, sequence, gantt)
* [Icons]({{%relref "shortcode/icon.md" %}}), [Buttons]({{%relref "shortcode/button.md" %}}), [Alerts]({{%relref "shortcode/alert.md" %}}), [Panels]({{%relref "shortcode/panel.md" %}}), [Tip/Note/Info/Warning boxes]({{%relref "shortcode/notice.md" %}})
* [Image resizing, shadow...]({{%relref "shortcode/image.md" %}})
* Tags

![](https://raw.githubusercontent.com/vjeantet/hugo-theme-docdock/master/images/tn.png?width=33pc&classes=border,shadow)

## Contribute to this documentation
Feel free to update this content, just send a Edit a page and pullrequest it, your modification will be deployed automatically when merged.

Use the "Edit this page" link you will find on top right of each page.