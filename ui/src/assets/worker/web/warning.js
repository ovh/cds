importScripts('../common.js');

onmessage = function (e) {
    loadWarnings(e.data.user, e.data.api);
};

function loadWarnings (user, api) {
    if (user && api) {
        setInterval(function () {
            postMessage(httpCall('/mon/warning', api, user));
        }, 2000);
    }
}
