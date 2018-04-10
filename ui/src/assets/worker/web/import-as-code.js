importScripts('../common.js');

var key = '';
var uuid = '';


onmessage = function (e) {
    key = e.data.key;
    uuid = e.data.uuid;
    getOperation(e.data.user, e.data.session, e.data.api);
};

function getOperation (user, session, api) {
    loop(5, function () {
        var url = '/import/' + key + '/'+ uuid;

        var xhr = httpCall(url, api, user, session);
        if (xhr.status >= 400) {
            return true;
        }
        if (xhr.status === 200 && xhr.responseText !== null) {
            postMessage(xhr.responseText);
        }
        return false;
    });
}
