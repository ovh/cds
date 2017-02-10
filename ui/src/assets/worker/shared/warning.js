importScripts('../common.js');

var workerStarted = false;
var ports = [];
var call = {};

onconnect = function (e) {
    if (!e.ports || e.ports.length === 0) {
        return;
    }

    ports.push(e.ports[0]);

    e.ports[0].onmessage = function (e) {
        if (!workerStarted) {
            workerStarted = true;
            loadWarnings(e.data.user, e.data.api);
        }
    };
};

function loadWarnings (user, api) {
    if (user && api) {
        setInterval(function () {
            var warnings = httpCall('/mon/warning', api, user);
            for (var i = 0; i < ports.length; i++) {
                ports[i].postMessage(warnings);
            }
        }, 2000);
    }
}
