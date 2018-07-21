importScripts('./common.js');
importScripts('./eventsource.js');

var sse;
var sseURL;
var pingUrl;
var headerKey;
var headerValue;
const connections = [];
var offline = false;
onconnect = function(e) {
    var port = e.ports[0];
    connections.push(port);
    port.onmessage = function (event) {
        pingUrl = event.data.pingURL;
        sseURL = event.data.sseURL;
        headerKey = event.data.headAuthKey;
        headerValue = event.data.headAuthValue;
        initSSE(false);
    };
};

function initSSE(force) {
    if ((!sse || force) && sseURL) {
        console.log('Start SSE');
        sse = connectSSE(sseURL, headerKey, headerValue);
        sse.onmessage = function(evt) {
            // if ack get UUID
            if (evt.data.indexOf('ACK: ') === 0) {
                return;
            }
            let jsonEvent = JSON.parse(evt.data);
            connections.forEach(p => {
                p.postMessage(jsonEvent);
            });
            return;
        };
    }
}

setInterval(() => {
    if (pingUrl) {
        try {
            var xhr = new XMLHttpRequest();
            xhr.open('GET', pingUrl , false, null, null);
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
            if (!offline) {
                console.log('Closing SSE');
                sse.close();
                offline = true;
            }
        }

    }
}, 5000);
