importScripts('../common.js');

onmessage = function (e) {
    loadLastUpdates(e.data.user, e.data.session, e.data.api);
};

var lastUpdate;

function loadLastUpdates (user, session, api) {
    loop(1, function () {
        var header = {};
        if (lastUpdate) {
            header = {"If-Modified-Since": lastUpdate};
        }
        var xhr = httpCall('/mon/lastupdates', api, user, session, header);
        if (xhr.status >= 400) {
            return true;
        }
        lastUpdate = xhr.getResponseHeader("ETag");
        if (xhr.status === 200) {
            postMessage(xhr.responseText);
        }
        return false;
    });
}
