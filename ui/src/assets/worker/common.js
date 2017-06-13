var apiTestEnv = 'foo.bar'; // url for test cf environments/environment.ts

function httpCall (path, host, user, session) {
    if (host !== apiTestEnv) {
        var xhr = new XMLHttpRequest();
        xhr.open('GET', host + path, false, null, null);
        if (session) {
            xhr.setRequestHeader("Session-Token", session);
        } else if (user) {
            xhr.setRequestHeader("Authorization", "Basic " + user.token);
        }

        xhr.send(null);
        if (xhr.status === 200) {
            return xhr.responseText;
        }
        if (xhr.status === 401) {
            close();
        }
        return null;
    }
}
