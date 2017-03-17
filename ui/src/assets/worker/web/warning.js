importScripts('../common.js');

onmessage = function (e) {
    loadWarnings(e.data.user, e.data.session, e.data.api);
};

function loadWarnings (user, session, api) {
    if (user && api) {
        setInterval(function () {
            postMessage(httpCall('/mon/warning', api, user, session));
        }, 10000);
    }
}
