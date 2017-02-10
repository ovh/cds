importScripts('../common.js');

var workerStarted = false;
var ports = [];
var call = {};

onconnect = function (e) {
    if (!e.ports || e.ports.length === 0) {
        return;
    }
    // Register worker + give it an ID
    ports.push(e.ports[0]);
    var id = ports.length;

    // Return ID to caller
    e.ports[0].postMessage({ 'worker_id': id });

    // Receive msg
    e.ports[0].onmessage = function (e) {
        switch (e.data.action) {
            case 'subscribe':
                addCall(e);
                break;
            case 'unsubscribe':
                removePort(e);
                break;
        }
        // Run worker
        if (!workerStarted) {
            workerStarted = true;
            loadApplicationStatus(e.data.user, e.data.api);
        }
    };
};

/**
 * Load all need workflow status
 * @param user
 * @param api
 */
function loadApplicationStatus (user, api) {
    if (user && api) {
        callAPI(user, api);
        setInterval(function () {
            callAPI(user, api);
        }, 2000);
    }
}

function callAPI(user, api) {
    // Browse all call needed
    if (Object.keys(call).length === 0) {
        console.log('Nothing to do');
    }
    for(var k in call ){
        var c = call[k];
        var url = '/project/' + c.key + '/application/' + c.appName +
            '?applicationStatus=true&branchName=' + c.branch + '&version=' + c.version;
        httpCallSharedWorker(url, api, user, c, k, postCall);
    }
}

function postCall(k, response) {
}

/**
 * Add a port for a call
 * @param e
 */
function addCall(e) {
    // call ID
    var key = 'filter-' + e.data.key + '-' + e.data.appName + '-' + e.data.branch + '-' + e.data.version;

    // If call don't exist, create it
    if (!call[key]) {
        call[key]= {
            key: e.data.key,
            appName: e.data.appName,
            branch: e.data.branch,
            version: e.data.version,
            ports: []
        };
    }

    // check subscription
    var found = false;
    call[key].ports.forEach( function(id) {
        if (id === e.data.id) {
            found = true;
        }
    });

    if (!found) {
        // Add port for the call
        call[key].ports.push(e.data.id);
    }
}

/**
 * Remove a port from all call
 * @param e
 */
function removePort(e) {
    // Browse all call
    for (var k in call) {
        var c = call[k];
        // For each call, browse all ports
        for (var indexPort = 0; indexPort < c.ports.length; indexPort++) {
            // If id match, delete it
            if (c.ports[indexPort] === e.data.id) {
                c.ports.splice(indexPort, 1);
                indexPort--;
            }
        }
        if (c.ports.length === 0) {
            delete call[k];
        }
    }
}


