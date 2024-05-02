var socket = new WebSocket('ws://' + location.host + '/ws');
var req_queue = [];
var resp_queue = [];
var disabledFileExtensions = [];

var reqCookies = new Object();
var respCookies = new Object();

document.addEventListener('DOMContentLoaded', function() {
    var openTab = window.location.hash.slice(1);
    switch (openTab) {
        case 'requests':
            openRequestsTab();
            break;
        case 'responses':
            openResponsesTab();
            break;
        case 'history':
            openHistoryTab();
            break;
        default:
            openRequestsTab();
    }
    // Clear the input fields
    Array.prototype.slice.call(document.getElementsByTagName("input")).forEach(element => {
        element.value = '';
    });
    Array.prototype.slice.call(document.getElementsByTagName("textarea")).forEach(element => {
        element.value = '';
    });
    // Create a new WebSocket instance
    socket = new WebSocket('ws://' + location.host + '/ws');

    // WebSocket event listeners
    socket.addEventListener('open', function(event) {
        console.log('WebSocket connection established');
        // Pull the current queue from the WebSocket server
        socket.send(JSON.stringify({ action: 'ping' }));
        socket.send(JSON.stringify({ action: 'get_settings' }));
        socket.send(JSON.stringify({ action: 'get_req_queue' }));
        socket.send(JSON.stringify({ action: 'get_resp_queue' }));
        if (window.location.hash.slice(1) == 'history') {
            socket.send(JSON.stringify({ action: 'get_history' }));
        }
    });

    socket.addEventListener('message', parseWSMessage);

    socket.addEventListener('close', function(event) {
        console.log('WebSocket connection closed');
        // Reconnect to the WebSocket server after a delay
        setTimeout(Reconnect, 1000);
    });

    socket.addEventListener('error', function(event) {
        console.error('WebSocket error:', event);
    });

    

});

function parseWSMessage(event) {
    console.debug('Received message:', event.data);
    // Handle incoming messages from the WebSocket server
    const message = JSON.parse(event.data);
    switch (message.msgtype) {
        case 'req_queue':
            // Update the queue with the new data
            req_queue = message.queue;
            if (document.getElementById("uuid").value == '' && req_queue.length > 0) {
                loadNextInQueue();
            }
            break;
        case 'resp_queue':
            // Update the queue with the new data
            resp_queue = message.queue;
            if (document.getElementById("resp-uuid").value == '' && resp_queue.length > 0) {
                loadNextRespInQueue();
            }
            break;
        case 'history':
            updateHistory(message.queue);
        case 'error':
            // Display an error message to the user
            console.error(message.msg);
            if (message.msg === 'UUID does not match') {        
                socket.send(JSON.stringify({ action: 'get_resp_queue' }));
                socket.send(JSON.stringify({ action: 'get_req_queue' }));
            }
            break;
        case 'ping':
            console.log('Pong');
            break;
        case "newRequest":
            // Add the new request to the queue
            req_queue.push(message.queue[0]);
            if (document.getElementById("uuid").value == '' && req_queue.length > 0) {
                loadNextInQueue();
            }
            break;
        case "newResponse":
            // Add the new response to the queue
            resp_queue.push(message.queue[0]);
            if (document.getElementById("resp-uuid").value == '' && resp_queue.length > 0) {
                loadNextRespInQueue();
            }
            break;
        case "handled":
            // Remove the request from the queue
            req_queue = req_queue.filter(item => item.uuid !== message.uuid);
            resp_queue = resp_queue.filter(item => item.uuid !== message.uuid);
            if (document.getElementById("uuid").value == message.uuid) {
                loadNextInQueue();
                socket.send(JSON.stringify({ action: 'get_resp_queue' }));
            } else {
                if (document.getElementById("resp-uuid").value == message.uuid) {
                    loadNextRespInQueue();
                }
            }
            break;
        case "settings":
            loadSettings(message)
            break;
        case "success":
            console.log("Success");
            break;
        default:
            console.error('Unknown message type:', message.msgtype);
    }
};

function Reconnect() {
    socket = new WebSocket('ws://' + location.host + '/ws');
    socket.addEventListener('open', function(event) {
        console.log('WebSocket connection established');
        // Pull the current queue from the WebSocket server
        socket.send(JSON.stringify({ action: 'ping' }));
        socket.send(JSON.stringify({ action: 'get_settings' }));
        socket.send(JSON.stringify({ action: 'get_req_queue' }));
        socket.send(JSON.stringify({ action: 'get_resp_queue' }));
    });
    socket.addEventListener('message', parseWSMessage);
    socket.addEventListener('close', function(event) {
        console.log('WebSocket connection closed');
        // Reconnect to the WebSocket server after a delay
        setTimeout(Reconnect, 3000);
    });
    socket.addEventListener('error', function(event) {
        console.error('WebSocket error:', event);
    });
}


function handlePass() {
    // Send a pass message to the WebSocket server
    var item = new Object();
    item.path = document.getElementById("path").value;
    item.body = document.getElementById("body").value;
    item.method = document.getElementById("method").value;
    // var headers = new Object();
    // for (var line of document.getElementById("headers").value.split('\n')) {
    //     var parts = line.split(':');
    //     if (parts.length < 2) {
    //         continue;
    //     }
    //     var header = parts.shift();
    //     var value = parts.join(':');
    //     // Remove leading and trailing whitespace
    //     header = header.trim();
    //     value = value.trim();
    //     headers[header] = new Array(value);
    // }
    var headerTable = document.getElementById("reqHeaderTable");
    item.headers = parseHeaderTable(headerTable);
    // item.headers = headers
    item.uuid = document.getElementById("uuid").value;
    item.host = document.getElementById("host").value;
    item.query = encodeURIComponent(document.getElementById("query").value);
    // item.cookies = JSON.parse(document.getElementById("cookies").value);
    item.cookies = parseCookiesTable(document.getElementById("reqCookiesTable"));
    
    socket.send(JSON.stringify({ action: 'pass_req' , uuid: document.getElementById("uuid").value, queue: [item] }));
    socket.send(JSON.stringify({ action: 'get_req_queue' }));
    socket.send(JSON.stringify({ action: 'get_resp_queue' }));
}

function handleDrop() {
    socket.send(JSON.stringify({ action: 'drop', uuid: document.getElementById("uuid").value }));
    socket.send(JSON.stringify({ action: 'get_req_queue' }));
    socket.send(JSON.stringify({ action: 'get_resp_queue' }));
}
    
function handleRespPass() {
    var item = new Object();
    item.status = parseInt(document.getElementById("resp-statusCode").value);
    item.body = document.getElementById("resp-body").value;
    // var headers = new Object();
    // for (var line of document.getElementById("resp-headers").value.split('\n')) {
    //     var parts = line.split(':');
    //     if (parts.length < 2) {
    //         continue;
    //     }
    //     var header = parts.shift();
    //     var value = parts.join(':');
    //     // Remove leading and trailing whitespace
    //     header = header.trim();
    //     value = value.trim();
    //     headers[header] = new Array(value);
    // }
    item.headers = parseHeaderTable(document.getElementById("respHeaderTable"));
    // item.cookies = JSON.parse(document.getElementById("resp-cookies").value);
    item.cookies = parseCookiesTable(document.getElementById("respCookiesTable"));
    item.uuid = document.getElementById("resp-uuid").value;
    socket.send(JSON.stringify({ action: 'pass_resp' , uuid: document.getElementById("resp-uuid").value, queue: [item] }));
    socket.send(JSON.stringify({ action: 'get_req_queue' }));
    socket.send(JSON.stringify({ action: 'get_resp_queue' }));
}

function loadNextInQueue() {
    // Load the next item in the queue
    if (req_queue.length > 0) {
        const item = req_queue.shift();
        console.log('Loading item:', item);
        document.getElementById("path").value = item.path;
        document.getElementById("body").value = item.body;
        document.getElementById("method").value = item.method;
        // document.getElementById("headers").value = parseHeaders(item.headers);
        createHeaderTable(document.getElementById("reqHeaderTable"), item.headers);
        document.getElementById("uuid").value = item.uuid;
        document.getElementById("host").value = item.host;
        document.getElementById("query").value = decodeURIComponent(item.query);
        document.getElementById("requestsButton").classList.add("notification");
        createCookieTable(document.getElementById("reqCookiesTable"), item.cookies);
    } else {
        document.getElementById('path').value = 'Queue is empty';
        document.getElementById('body').value = '';
        document.getElementById('method').value = '';
        // document.getElementById('headers').value = '';
        clearTable(document.getElementById("reqHeaderTable"));
        document.getElementById('uuid').value = '';
        document.getElementById("host").value = '';
        document.getElementById("query").value = '';
        // document.getElementById("cookies").value = '';
        clearTable(document.getElementById("reqCookiesTable"));
        document.getElementById("requestsButton").classList.remove("notification");
    }
}

function loadNextRespInQueue() {
    if (resp_queue.length > 0) {
        const item = resp_queue.shift();
        console.log('Loading item:', item);
        console.log(resp_queue)
        document.getElementById("resp-statusCode").value = item.status
        document.getElementById("resp-body").value = item.body;
        // document.getElementById("resp-headers").value = parseHeaders(item.headers);
        createHeaderTable(document.getElementById("respHeaderTable"), item.headers);
        document.getElementById("resp-uuid").value = item.uuid;
        document.getElementById("responsesButton").classList.add("notification");
        // document.getElementById("resp-cookies").value = parseCookies(item.cookies);
        createCookieTable(document.getElementById("respCookiesTable"), item.cookies);
    } else {
        document.getElementById('resp-statusCode').value = '';
        document.getElementById('resp-body').value = '';
        // document.getElementById('resp-headers').value = '';
        clearTable(document.getElementById("respHeaderTable"));
        document.getElementById('resp-uuid').value = '';
        // document.getElementById('resp-cookies').value = '';
        clearTable(document.getElementById("respCookiesTable"));
        document.getElementById("responsesButton").classList.remove("notification");
    }
}

function handleSettings() {
    var settings = new Object();
    settings.enabled = document.getElementById("enabled").checked
    settings.ignoredTypes = [];
    settings.ignoredHosts = [];
    settings.disabledFileExtensions = document.getElementsByClassName("disabledFileExtension");
    settings.disabledHosts = document.getElementsByClassName("disabledHost");
    settings.proxyPort = parseInt(document.getElementById("proxyPort").value);
    settings.catchResponse = document.getElementById("catchResponse").checked;
    for (var i = 0; i < settings.disabledFileExtensions.length; i++) {
        if (settings.disabledFileExtensions[i].value == '') {
            continue;
        }
        if (!settings.ignoredTypes.includes(settings.disabledFileExtensions[i].value)) {
            settings.ignoredTypes.push(settings.disabledFileExtensions[i].value);
        }
    }
    settings.whitelist = document.getElementById("whitelist").checked;
    for(var i = 0; i < settings.disabledHosts.length; i++) {
        if (settings.disabledHosts[i].value == '') {
            continue;
        }
        if (!settings.ignoredHosts.includes(settings.disabledHosts[i].value)) {
            settings.ignoredHosts.push(settings.disabledHosts[i].value);
        }
    }
    settings.useRegex = document.getElementById("useRegex").checked;
    socket.send(JSON.stringify({ action: 'set_settings', settings : settings }));
    closeSettingsModal()
}

function loadSettings(message) {
    document.getElementById("enabled").checked = message.settings.enabled;
    document.getElementById("disabledFileExtensions").innerHTML = '';
    if (message.settings.ignoredTypes != null) {
        for (var i = 0; i < message.settings.ignoredTypes.length; i++) {
            createDisabledExtension(message.settings.ignoredTypes[i]);
            disabledFileExtensions.push(message.settings.ignoredTypes[i]);
        }
    }
    document.getElementById("disabledHosts").innerHTML = '';
    if (message.settings.ignoredHosts != null) {
        for (var i = 0; i < message.settings.ignoredHosts.length; i++) {
            createDisabledHost(message.settings.ignoredHosts[i]);
        }
    }
    document.getElementById("proxyPort").value = message.settings.proxyPort;
    document.getElementById("catchResponse").checked = message.settings.catchResponse;
    document.getElementById("whitelist").checked = message.settings.whiteList;
    document.getElementById("useRegex").checked = message.settings.useRegex;
}

function openSettingsModal() {
    document.getElementById("settingsModal").style.display = "block";
}

function closeSettingsModal() {
    document.getElementById("settingsModal").style.display = "none";
}

function openRequestsTab() {
    document.getElementById("requestsTab").style.display = "block";
    document.getElementById("responsesTab").style.display = "none";
    document.getElementById("historyTab").style.display = "none";
    window.location.hash = 'requests';
}

function openResponsesTab() {
    document.getElementById("requestsTab").style.display = "none";
    document.getElementById("responsesTab").style.display = "block";
    document.getElementById("historyTab").style.display = "none";
    window.location.hash = 'responses';
}

function openHistoryTab() {
    document.getElementById("requestsTab").style.display = "none";
    document.getElementById("responsesTab").style.display = "none";
    document.getElementById("historyTab").style.display = "block";
    window.location.hash = 'history';
    if (socket.readyState == WebSocket.OPEN) {
        socket.send(JSON.stringify({ action: 'get_history' }));
    }
}

function createDisabledExtension(extension) {
    var div = document.createElement("div");
    var input = document.createElement("input");
    input.type = "text";
    input.className = "disabledInput disabledFileExtension";
    input.value = extension;
    div.appendChild(input);
    var button = document.createElement("button");
    button.innerHTML = "Remove";
    button.className = "removeButton";
    button.onclick = function(e) {
        e.preventDefault();
        div.remove();
    }
    div.appendChild(button);
    document.getElementById("disabledFileExtensions").appendChild(div);
}

function createDisabledHost(host) {
    var div = document.createElement("div");
    var input = document.createElement("input");
    input.type = "text";
    input.className = "disabledInput disabledHost";
    input.value = host;
    div.appendChild(input);
    var button = document.createElement("button");
    button.innerHTML = "Remove";
    button.className = "removeButton";
    button.onclick = function(e) {
        e.preventDefault();
        div.remove();
    }
    div.appendChild(button);
    document.getElementById("disabledHosts").appendChild(div);
}

function updateHistory(history) {
    var table = document.getElementById("historyTable");
    clearTable(table);
    history.sort((a, b) => (a.timestamp > b.timestamp) ? -1 : 1);
    for (var i = 1; i < history.length + 1 && i < 15; i++) {
        var row = table.insertRow(i);
        var cell = row.insertCell(0);
        cell.innerHTML = new Date(history[i-1].timestamp).toLocaleString();
        cell= row.insertCell(1);
        cell.innerHTML = history[i-1].method;
        cell = row.insertCell(2);
        cell.innerHTML = history[i-1].path + history[i-1].query;
        cell = row.insertCell(3);
        cell.innerHTML = history[i-1].host
        cell = row.insertCell(4);
        cell.innerHTML = parseHeaders(history[i-1].headers);
        cell = row.insertCell(5);
        cell.innerHTML = history[i-1].body;
        cell = row.insertCell(6);
        cell.innerHTML = history[i-1].status;
        cell = row.insertCell(7);
        cell.innerHTML = history[i-1].statusMessage;
        cell = row.insertCell(8);
        cell.innerHTML = parseHeaders(history[i-1].respHeaders);
        cell = row.insertCell(9);
        textBox = document.createElement("textarea");
        textBox.rows = 10;
        textBox.value = history[i-1].respBody;
        cell.innerHTML = '';
        cell.appendChild(textBox);


    }
}

function parseHeaders(headers) {
    var res = "";
    for (var key in headers) {
        res += key + ": " + headers[key] + "\n";
    }
    return res
}

function ignoreCurrentHost() {
    var host = document.getElementById("host").value;
    if (host == '') {
        return;
    }
    createDisabledHost(host);
    handleSettings();
    handlePass();
}

function ignoreCurrentExtension() {
    var path = document.getElementById("path").value;
    if (path == '') {
        return;
    }
    var extension = path.split('.').pop();
    createDisabledExtension(extension);
    handleSettings();
    handlePass();
}


function createHeaderTable(table, headers) {
    clearTable(table);
    for (var key in headers) {
        addHeader(table, [key, headers[key]]);
    }
}

function clearTable(table) {
    while (table.rows.length > 1) {
        table.deleteRow(1);
    }
}

function addHeader(table, header) {
    var row = table.insertRow(table.rows.length);
    var cell = row.insertCell(0);
    var input = document.createElement("input");
    input.type = "text";
    input.value = header[0] || '';
    cell.appendChild(input);
    cell = row.insertCell(1);
    input = document.createElement("textarea");
    input.type = "text";
    input.value = header[1] || '';
    cell.appendChild(input);
    cell = row.insertCell(2);
    var button = document.createElement("button");
    button.innerHTML = "Remove";
    button.onclick = function(e) {
        e.preventDefault();
        this.parentElement.parentElement.remove();
    }
    cell.appendChild(button);
}

function parseHeaderTable(table) {
    var headers = new Object();
    for (var i = 1; i < table.rows.length; i++) {
        var key = table.rows[i].cells[0].children[0].value;
        var value = table.rows[i].cells[1].children[0].value;
        headers[key] = new Array(value);
    }
    return headers;
}

function createCookieTable(table, cookies) {
    clearTable(table);
    for(var cookie in cookies) {
        addCookie(table, cookies[cookie]);
    }

}

function addCookie(table, cookie) {
    // Name, Value, Path, Domain, Expires
    var row = table.insertRow(table.rows.length);
    var cell = row.insertCell(0);
    var input = document.createElement("input");
    input.type = "text";
    input.value = cookie.Name || '';
    cell.appendChild(input);
    cell = row.insertCell(1);
    input = document.createElement("textarea");
    input.type = "text";
    input.value = cookie.Value || '';
    cell.appendChild(input);
    cell = row.insertCell(2);
    input = document.createElement("input");
    input.type = "text";
    input.value = cookie.Path || '';
    cell.appendChild(input);
    cell = row.insertCell(3);
    input = document.createElement("input");
    input.type = "text";
    input.value = cookie.Domain || '';
    cell.appendChild(input);
    cell = row.insertCell(4);
    input = document.createElement("input");
    input.type = "text";
    input.value = cookie.Expires || '';
    cell.appendChild(input);
    cell = row.insertCell(5);
    var button = document.createElement("button");
    button.innerHTML = "Remove";
    button.onclick = function(e) {
        e.preventDefault();
        this.parentElement.parentElement.remove();
    }
    cell.appendChild(button);
}

function parseCookiesTable(table) {
    var cookies = new Array();
    for (var i = 1; i < table.rows.length; i++) {
        var Cookie = new Object();
        var name = table.rows[i].cells[0].children[0].value;
        var value = table.rows[i].cells[1].children[0].value;
        var path = table.rows[i].cells[2].children[0].value;
        var domain = table.rows[i].cells[3].children[0].value;
        var expires = table.rows[i].cells[4].children[0].value;
        Cookie.Name = name;
        Cookie.Value = value;
        Cookie.Path = path;
        Cookie.Domain = domain;
        Cookie.Expires = expires;
        cookies.push(Cookie);
    }
    return cookies;
}
