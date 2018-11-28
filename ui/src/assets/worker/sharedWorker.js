importScripts('./common.js');
importScripts('./eventsource.js');

let sse;
let sseURL;
let pingUrl;
let headerKey;
let headerValue;
const connections = [];
let offline = false;
onconnect = function(e) {
    let port = e.ports[0];
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
            connections.forEach( p => {
                p.postMessage(jsonEvent);
            });
            return;
        };
        sse.onerror = function (err) {
          console.log('Error on SSE: ', err);
        };
    }
}

// Send state of the connexion every 5 seconds
setInterval(() => {
    if (sse && sse.readyState > 1) {
        sse.close();
        sse = undefined;
    }
    connections.forEach( p => {
        p.postMessage({ healthCheck: sse.readyState });
    });
}, 5000);

// Check if token is still valid
setInterval(() => {
    if (pingUrl) {
        try {
            let xhr = new XMLHttpRequest();
            xhr.open('GET', pingUrl , false, null, null);
            xhr.setRequestHeader(headerKey, headerValue);
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
}, 60000);
