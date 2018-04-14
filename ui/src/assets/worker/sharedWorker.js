importScripts('./eventsource.js');

var sse;
var uuid;
var clients = [];
var filters = [];
var id = 0;

onmessage = function (e) {
    if (!sse && e.data.head && e.data.sseURL) {
        sse = connectSSE(e.data.sseURL, e.data.head);
        sse.onmessage = function(evt) {
            // if ack get UUID
            if (evt.data.indexOf('ACK: ') === 0) {
                uuid = evt.data.substr(5);
                e.port.postMessage({ 'uuid': uuid});
                return;
            }
            // send event to tabs
            clients.forEach(c => {
                c.port.postMessage(evt);
            });
            return;
        };
        // save worker client
        id++;
        var client = e.ports[0];
        clients[id] = client;
        e.port.postMessage({ 'identifier': id});
    }

    if (e.data.add_filter) {
        filters[e.data.id] = e.data.add_filter;
        //Call api
        // TODO
    }
    if (e.data.del_filter) {
        // browse clients filters to count the use of this filter

        delete filters[e.data.id];
        //Call api
        // TODO
    }
};