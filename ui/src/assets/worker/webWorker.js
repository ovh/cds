importScripts('./eventsource.js');
importScripts('./common.js');

var sse;
var urlSubscribe;
var urlUnsubscribe;
var uuid = '';
var headAuthKey;
var headAuthValue;
onmessage = function (e) {
    if (!sse && e.data.sseURL) {
        urlSubscribe = e.data.urlSubscribe;
        urlUnsubscribe = e.data.urlUnsubscribe;
        headAuthKey = e.data.headAuthKey;
        headAuthValue = e.data.headAuthValue;
        sse = connectSSE(e.data.sseURL + '?uuid=' + uuid, e.data.headAuthKey, e.data.headAuthValue, uuid);
        sse.onmessage = function(evt) {
            if (evt.data.indexOf('ACK: ') === 0) {
                uuid = evt.data.substr(5).trim();
                sse.url = e.data.sseURL + '?uuid=' + uuid;
                postMessage({uuid: uuid});
                return;
            }
            postMessage(JSON.parse(evt.data));
        };
    }

    if(e.data.add_filter) {
        e.data.add_filter['overwrite'] = true;
        subscribeEvent(urlSubscribe, headAuthKey, headAuthValue, e.data.add_filter);
    }
};