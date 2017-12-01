var apiTestEnv = 'foo.bar'; // url for test cf environments/environment.ts

function loop (initWaitingTime, func) {
    var retry = 0;
    var fibonacciStep = 0;
    var timeToWait = initWaitingTime;

    // First call  then run in an interval
    func();
    setInterval(function () {
        retry++;
        if (!(retry % timeToWait)) {
            retry = 0;
            var hasToRetry = func();
            if (hasToRetry) {
                fibonacciStep++;
                timeToWait = fibonacci(fibonacciStep);
            } else {
                fibonacciStep = 0;
            }
        }
    }, 1000);

}

function httpCall (path, host, user, session, additionnalHeaders) {
    if (host !== apiTestEnv) {
        var xhr = new XMLHttpRequest();
        xhr.open('GET', host + path, false, null, null);
        if (session) {
            xhr.setRequestHeader("Session-Token", session);
        } else if (user) {
            xhr.setRequestHeader("Authorization", "Basic " + user.token);
        }
        if (additionnalHeaders) {
            Object.keys(additionnalHeaders).forEach(function (k) {
                xhr.setRequestHeader(k, additionnalHeaders[k]);
            });
        }

        xhr.send(null);
        if (xhr.status === 401 || xhr.status === 403 || xhr.status === 404) {
            close();
        }
        return xhr;
    }
}

function fibonacci (retry) {
    if (retry <= 1) {
        return retry;
    }
    return fibonacci(retry - 1) + fibonacci(retry - 2);
}
