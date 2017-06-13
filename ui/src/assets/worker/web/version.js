importScripts('../common.js');

onmessage = function () {
    loadVersion();
};

function loadVersion () {
    setInterval(function () {
        var date = new Date();
        postMessage(httpCall('assets/version.json?ts=' + date.getTime(), '../../../'));
    }, 10000);
}
