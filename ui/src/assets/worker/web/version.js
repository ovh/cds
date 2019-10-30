importScripts('../common.js');

onmessage = function (e) {
    var url;
    if (e.data.mode === 'local') {
        url = '/assets/version.json';
    } else {
        if (e.data.base && e.data.base !== '') {
            //url = e.data.base;
        }
        url = '/mon/version';
    }
    loadVersion(url);
};

function loadVersion (url) {
    loop(60, function () {
        var xhr = httpCall(url  + '?ts=' + (new Date()).getTime(), '../../../');
        if (xhr.status >= 400) {
            return true;
        }
        if (xhr.status === 200) {
            postMessage(xhr.responseText);
        }
        return false;
    });
}
