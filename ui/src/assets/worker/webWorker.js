importScripts('./eventsource.js');
importScripts('./common.js');

var sse;
var headAuthKey;
var headAuthValue;
onmessage = function (e) {
    if (!sse && e.data.sseURL) {
        headAuthKey = e.data.headAuthKey;
        headAuthValue = e.data.headAuthValue;
        sse = connectSSE(e.data.sseURL, e.data.headAuthKey, e.data.headAuthValue);
        sse.onmessage = function (evt) {
            if (evt.data.indexOf('ACK: ') === 0) {
                return;
            }
            let myEvent = JSON.parse(evt.data);
            postMessage(myEvent);
        };
        sse.onerror = function (err) {
            console.log('SSE Error: ', err);
        }
    }
};