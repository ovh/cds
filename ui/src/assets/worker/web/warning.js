importScripts('../common.js');

onmessage = function (e) {
    loadWarnings(e.data.user, e.data.session, e.data.api);
};

function loadWarnings (user, session, api) {
    loop(10, function () {
        var xhr = httpCall('/mon/warning', api, user, session);
        if (xhr.status >= 400) {
            return true;
        }
        if (xhr.status === 200) {
            postMessage(xhr.responseText);
        }
        return false;
    });
}
