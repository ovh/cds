onmessage = function (e) {
    loadLastUpdates(e.data.user, e.data.session, e.data.api);
};

var lastUpdate;

function loadLastUpdates (user, session, api) {
    if (user && api) {
        setInterval(function () {
            var response = httpCall('/mon/lastupdates', api, user, session);
            postMessage(response);
        }, 1000);
    }
}


function httpCall (path, host, user, session) {
    if (host !== 'foo.bar') {
        var xhr = new XMLHttpRequest();
        xhr.open('GET', host + path, false, null, null);
        if (session) {
            xhr.setRequestHeader("Session-Token", session);
        } else if (user) {
            xhr.setRequestHeader("Authorization", "Basic " + user.token);
        }

        if (lastUpdate) {
            xhr.setRequestHeader("If-Modified-Since", lastUpdate);
        }

        xhr.send(null);
        if (xhr.status === 200) {
            lastUpdate = xhr.getResponseHeader("ETag");
            return xhr.responseText;
        }
        return null;
    }
}
