importScripts('./eventsource.js');

var sse;
onmessage = function (e) {
    if (!sse && e.data.head && e.data.sseURL) {
        sse = connectSSE(e.data.sseURL, e.data.head);

        sse.onmessage = function(e) {
            postMessage(e.data);
        };
    }
};