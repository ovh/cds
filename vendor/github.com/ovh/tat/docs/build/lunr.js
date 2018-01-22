
var hugolunr = require('lunr-hugo');
var h = new hugolunr();
h.setInput('content/**');
h.setOutput('static/json/search.json');
h.index();
