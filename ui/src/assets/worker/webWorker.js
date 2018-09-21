importScripts('./eventsource.js');
importScripts('./common.js');

var sse;
var headAuthKey;
var headAuthValue;
var pingUrl;
var offline = false;
onmessage = function (e) {
    if (!sse && e.data.sseURL) {
        pingUrl = e.data.pingURL;
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

setInterval(() => {
    if (pingUrl) {
        try {
            var xhr = new XMLHttpRequest();
            xhr.open('GET', pingUrl , false, null, null);
            xhr.setRequestHeader(headAuthKey, headAuthValue);
            xhr.send(null);
            if (xhr.status >= 400) {
                if (!offline) {
                    console.log('Closing SSE');
                    sse.close();
                    offline = true;
                }
            } else {
                if (offline) {
                    initSSE(true);
                    offline = false;
                }
            }
        } catch (e) {
            console.error(e);
            if (!offline) {
                console.log('Closing SSE');
                sse.close();
                offline = true;
            }
        }

    }
}, 5000);
