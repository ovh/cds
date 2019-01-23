importScripts('../common.js');

var path = '';

onmessage = function (e) {
    path = e.data.path;
    getOperation(e.data.user, e.data.session, e.data.api);
};

function getOperation (user, session, api) {
    loop(5, function () {
        var xhr = httpCall(path, api, user, session);
        if (xhr.status >= 400) {
            return true;
        }
        if (xhr.status === 200 && xhr.responseText !== null) {
            postMessage(xhr.responseText);
        }
        return false;
    });
}
