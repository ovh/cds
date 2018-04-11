importScripts('./eventsource.js');

var sse;
onmessage = function (e) {
    if (!sse && e.data.head && e.data.sseURL) {
        sse = new EventSourcePolyfill(e.data.sseURL, {headers: e.data.head });

        sse.onmessage = function(e) {
            postMessage(e.data);
        };
    }
};