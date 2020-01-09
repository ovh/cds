importScripts('../common.js');

var path = '';

onmessage = function (e) {
    path = e.data.path;
    getOperation();
};

function getOperation() {
    loop(5, function () {
        var xhr = httpCallAPI(path);
        if (xhr.status >= 400) {
            return true;
        }
        if (xhr.status === 200 && xhr.responseText !== null) {
            postMessage(xhr.responseText);
        }
        return false;
    });
}
