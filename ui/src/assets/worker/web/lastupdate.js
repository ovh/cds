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
        var response = httpCall('/mon/lastupdates', api, user, session, header);
        if (response.xhr.status >= 400) {
            return true;
        }
        lastUpdate = response.xhr.getResponseHeader("ETag");
        if (response.xhr.status === 200) {
            postMessage(response.xhr.responseText);
        }
        return false;
    });
}
