importScripts('./eventsource.js');
importScripts('./common.js');

var sse;
var urlSubscribe;
var urlUnsubscribe;
var uuid = '';
var headAuthKey;
var headAuthValue;
var currentFilter;
onmessage = function (e) {
    if (!sse && e.data.sseURL) {
        urlSubscribe = e.data.urlSubscribe;
        urlUnsubscribe = e.data.urlUnsubscribe;
        headAuthKey = e.data.headAuthKey;
        headAuthValue = e.data.headAuthValue;
        sse = connectSSE(e.data.sseURL, e.data.headAuthKey, e.data.headAuthValue, uuid);
        sse.onmessage = function (evt) {
            if (evt.data.indexOf('ACK: ') === 0) {
                uuid = evt.data.substr(5).trim();
                postMessage({uuid: uuid});
                if (currentFilter) {
                    currentFilter.uuid = uuid;
                    subscribeEvent(urlSubscribe, headAuthKey, headAuthValue, currentFilter);
                }
                return;
            }
            let myEvent = JSON.parse(evt.data);
            postMessage(myEvent);
        };
        sse.onerror = function (err) {
            console.log('SSE Error: ', err);
        }
    }

    if(e.data.add_filter) {
        e.data.add_filter['overwrite'] = true;
        currentFilter = e.data.add_filter;
        subscribeEvent(urlSubscribe, headAuthKey, headAuthValue, e.data.add_filter);
    }
};