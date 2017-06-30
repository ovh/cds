importScripts('../common.js');

onmessage = function (e) {
    loadWarnings(e.data.user, e.data.session, e.data.api);
};

function loadWarnings (user, session, api) {
    loop(10, function () {
        var response = httpCall('/mon/warning', api, user, session);
        if (response.xhr.status >= 400) {
            return true;
        }
        if (response.xhr.status === 200) {
            postMessage(response.xhr.responseText);
        }
        return false;
    });
}
