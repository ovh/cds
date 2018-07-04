importScripts('./eventsource.js');

var sse;
var uuid;
var clients = [];
var filters = [];
var id = 0;
var headAuthKey;
var headAuthValue;
var urlSubscribe;
var urlUnsubscribe;
onmessage = function (e) {
    if (!sse && e.data.sseURL) {
        headAuthKey = e.data.headAuthKey;
        headAuthValue = e.data.headAuthValue;
        urlSubscribe = e.data.urlSubscribe;
        urlUnsubscribe = e.data.urlUnsubscribe;
        sse = connectSSE(e.data.sseURL, headAuthKey, headAuthValue);
        sse.onmessage = function(evt) {
            // if ack get UUID
            if (evt.data.indexOf('ACK: ') === 0) {
                uuid = evt.data.substr(5);

                // save worker client
                id++;
                var client = e.ports[0];
                clients[id] = client;

                e.port.postMessage({ 'uuid': uuid, 'identifier': id});
                return;
            }
            // send event to tabs
            clients.forEach(c => {
                c.port.postMessage(evt);
            });
            return;
        };
    }

    if (e.data.del_filter) {
        // browse clients filters to count the use of this filter
        var count = 0;
        filters.forEach((v, i) => {
            if (filterEqual(e.data.del_filter, v)) {
                count++;
            }
        });
        if (count === 1) {
            unsubscribeEvent(urlUnsubscribe, headAuthKey, headAuthValue, e.data.del_filter);
        }
        delete filters[e.data.id];
    }

    if (e.data.add_filter) {
        filters[e.data.id] = e.data.add_filter;
        subscribeEvent(urlSubscribe, headAuthKey, headAuthValue, e.data.add_filter);
    }
};

function filterEqual(f1, f2) {
    if (f1.project_key === f2.project_key && f1.workflow_name === f2.workflow_name) {
        return true;
    }
    return false;
}