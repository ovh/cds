var body;

function newElement(tag, className, id){
    var el = document.createElement(tag);
    if (className) el.className = className;
    if (id) el.id = id;
    return el;
}

function GetLatestReleaseInfo() {
    $.getJSON("https://api.github.com/repos/ovh/cds/releases/latest").done(function(release) {
      for (var i = 0; i < release.assets.length; i++) {
        $("."+release.assets[i].name.replace('.', '-')).text(release.assets[i].name);
        $("."+release.assets[i].name.replace('.', '-')).fadeIn("slow");
        $(".download-"+release.assets[i].name.replace('.', '-')).attr("href", release.assets[i].browser_download_url);
      }
    });
  }

var getUrlParameter = function getUrlParameter(sPageURL) {
    var url = sPageURL.split('?');
    var obj = {};
    if (url.length == 2) {
        var sURLVariables = url[1].split('&'),
            sParameterName,
            i;
        for (i = 0; i < sURLVariables.length; i++) {
            sParameterName = sURLVariables[i].split('=');
            obj[sParameterName[0]] = sParameterName[1];
        }
        return obj;
    } else {
        return undefined;
    }
};

jQuery(document).ready(function () {
    var sidebarActiveli = $('#sidebar').find('li.active').get(0);
    sidebarActiveli && sidebarActiveli.scrollIntoView();
    GetLatestReleaseInfo();

    // Execute actions on images generated from Markdown pages
    var images = $("div#docsContent img").not(".inline");

    // Wrap image inside a featherlight (to get a full size view in a popup)
    // images.wrap(function () {
    //     var image = $(this);
    //     if (!image.parent("a").length) {
    //         return "<a href='" + image[0].src + "' data-featherlight='image'></a>";
    //     }
    // });

    // Change styles, depending on parameters set to the image
    images.each(function (index) {
        var image = $(this);
        var o = getUrlParameter(image[0].src);
        if (typeof o !== "undefined") {
            var h = o["height"];
            var w = o["width"];
            var c = o["classes"];
            image.css({
                width: function () {
                    if (typeof w !== "undefined") {
                        return w;
                    }
                },
                height: function () {
                    if (typeof h !== "undefined") {
                        return h;
                    }
                }
            });
            if (typeof c !== "undefined") {
                var classes = c.split(',');
                $.each(classes, function(i) {
                    image.addClass(classes[i]);
                });
            }
        }
    });

    // // Add link button for every
    // var text, clip = new Clipboard('.anchor');
    // $("h1~h2,h1~h3,h1~h4,h1~h5,h1~h6").append(function (index, html) {
    //     var element = $(this);
    //     var url = document.location.origin + document.location.pathname;
    //     var link = url + "#" + element[0].id;
    //     return " <span class='anchor' data-clipboard-text='" + link + "'>" +
    //         "<i class='fa fa-link fa-lg'></i>" +
    //         "</span>";
    // });

    // $(".anchor").on('mouseleave', function (e) {
    //     $(this).attr('aria-label', null).removeClass('tooltipped tooltipped-s tooltipped-w');
    // });

    // clip.on('success', function (e) {
    //     e.clearSelection();
    //     $(e.trigger).attr('aria-label', 'Link copied to clipboard!').addClass('tooltipped tooltipped-s');
    // });
    
});

(function(){
    var menuSelected = true;
    var moving = false;
    var CSS_BROWSER_HACK_DELAY = 25;

    $(document).ready(function(){
        $('.toc-container').each(function () {
            var toc = this;
            var content = this.innerHTML;
            var container = newElement('div', 'container');
            container.innerHTML = content;
            $(toc).empty();
            toc.appendChild(container);
            CollapseBox($(container));
        });

        setMenuSelected();

        setTimeout(function () {
            menuSelected = false;
        }, 500);
    });

    function CollapseBox(container){
        container.children('.item').each(function(){
            var item = this;
            var isContainer = item.tagName === 'DIV';

            var titleText = item.getAttribute('data-title');
            var title = newElement('div', 'title');
            title.innerHTML = titleText;

            var wrapper, content;

            if (isContainer) {
                wrapper = newElement('div', 'wrapper');
                content = newElement('div', 'content');
                content.innerHTML = item.innerHTML;
                wrapper.appendChild(content);
            }

            item.innerHTML = '';
            item.appendChild(title);

            if (wrapper) {
                item.appendChild(wrapper);
                $(wrapper).css({height: 0});
            }

            $(title).click(function(){
                if (!menuSelected) {
                    if (moving) return;
                    moving = true;
                }

                if (container[0].getAttribute('data-single')) {
                    var openSiblings = item.siblings().filter(function(sib){return sib.hasClass('on');});
                    openSiblings.forEach(function(sibling){
                        toggleItem(sibling);
                    });
                }

                setTimeout(function(){
                    if (!isContainer) {
                        moving = false;
                        return;
                    }
                    toggleItem(item);
                }, CSS_BROWSER_HACK_DELAY);
            });

            function toggleItem(thisItem){
                var thisWrapper = $(thisItem).find('.wrapper').eq(0);
                if (!thisWrapper) return;

                var contentHeight = thisWrapper.find('.content').eq(0).innerHeight() + 'px';

                if ($(thisItem).hasClass('on')) {
                    thisWrapper.css({height: contentHeight});
                    $(thisItem).removeClass('on');

                    setTimeout(function(){
                        thisWrapper.css({height: 0});
                        moving = false;
                    }, CSS_BROWSER_HACK_DELAY);
                } else {
                    $(item).addClass('on');
                    thisWrapper.css({height: contentHeight});

                    var duration = parseFloat(getComputedStyle(thisWrapper[0]).transitionDuration) * 1000;

                    setTimeout(function(){
                        thisWrapper.css({height: ''});
                        moving = false;
                    }, duration);
                }
            }

            if (content) {
                var innerContainers = $(content).children('.container');
                if (innerContainers.length > 0) {
                    innerContainers.each(function(){
                        CollapseBox($(this));
                    });
                }
            }
        });
    }

    function setMenuSelected() {
        var pathname = location.href.split('#')[0];
        var currentLinks = [];

        $('.toc-container a').each(function () {
            if (pathname === this.href) currentLinks.push(this);
        });

        currentLinks.forEach(function (menuSelectedLink) {
            $(menuSelectedLink).parents('.item').each(function(){
                $(this).addClass('on');
                $(this).find('.wrapper').eq(0).css({height: 'auto'});
                $(this).find('.content').eq(0).css({opacity: 1});
            });

            $(menuSelectedLink).addClass('menuSelected');
            menuSelectedLink.onclick = function(e){e.preventDefault();};
        });
    }
})();
