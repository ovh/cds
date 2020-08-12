var lunrIndex, pagesIndex;
var resultDetails = []; 

function endsWith(str, suffix) {
    return str.indexOf(suffix, str.length - suffix.length) !== -1;
}

// Initialize lunrjs using our generated index file
function initLunr() {
    // First retrieve the index file
    $.getJSON($('#indexJSON').attr('href'))
        .done(function(index) {
            pagesIndex =   index;
            // Set up lunrjs by declaring the fields we use
            // Also provide their boost level for the ranking
            //lunrIndex = new lunr.Index
            lunrIndex = lunr(function () {
                this.ref("uri");
                this.field('title', {
                    boost: 15
                });
                this.field('tags', {
                    boost: 2000
                });
                this.field("content", {
                    boost: 2
                });
                // Feed lunr with each file and let lunr actually index them
                pagesIndex.forEach(function(page) {
                    this.add(page);
                }, this);
                this.pipeline.remove(this.stemmer)
            });

        })
        .fail(function(jqxhr, textStatus, error) {
            var err = textStatus + ", " + error;
            console.error("Error getting index.json file:", err);
        });
}

/**
 * Trigger a search in lunr and transform the result
 */
function search(query) {
    // Find the item in our index corresponding to the lunr one to have more info
    return lunrIndex.search(query).map(function(result) {
        return pagesIndex.filter(function(page) {
            return page.uri === result.ref;
        })[0];
    });
}

// Let's get started
initLunr();
$( document ).ready(function() {
    var searchList = new autoComplete({
        /* selector for the search box element */
        selector: $("#search-query").get(0),
        /* source is the callback to perform the search */
        source: function(term, response) {
            response(search(term));
        },
        /* renderItem displays individual search results */
        renderItem: function(item, term) {
            var numContextWords = 2;
            var text = item.content.match(
                "(?:\\s?(?:[\\w]+)\\s?){0,"+numContextWords+"}" +
                    term+"(?:\\s?(?:[\\w]+)\\s?){0,"+numContextWords+"}");
            item.context = text;
            
            var pathItem = item.uri;
            if (endsWith(pathItem,"/")) {
                pathItem = pathItem.substring(0, pathItem.length-1);
            };

            return '<div class="autocomplete-suggestion" ' +
                'data-term="' + term + '" ' +
                'data-title="' + item.title + '" ' +
                'data-uri="'+ item.uri + '" ' +
                'data-context="' + item.context + '">' +
                '» ' + item.title + " - " + pathItem +
                '<div class="context">' +
                (item.context || '') +'</div>' +
                '</div>';
        },
        /* onSelect callback fires when a search suggestion is chosen */
        onSelect: function(e, term, item) {
            location.href = item.getAttribute('data-uri');
        }
    });
});
