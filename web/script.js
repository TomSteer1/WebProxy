var socket = new WebSocket('ws://' + location.host + '/ws');
var queue = [];

document.addEventListener('DOMContentLoaded', function() {
    // Create a new WebSocket instance
    socket = new WebSocket('ws://' + location.host + '/ws');

    // WebSocket event listeners
    socket.addEventListener('open', function(event) {
        console.log('WebSocket connection established');
        // Pull the current queue from the WebSocket server
        socket.send(JSON.stringify({ action: 'ping' }));
        socket.send(JSON.stringify({ action: 'get_queue' }));
    });

    socket.addEventListener('message', function(event) {
        console.log('Received message:', event.data);
        // Handle incoming messages from the WebSocket server
        const message = JSON.parse(event.data);
        switch (message.msgtype) {
            case 'queue':
                // Update the queue with the new data
                queue = message.queue;
                if (document.getElementById("url").value === 'Queue is empty') {
                    loadNextInQueue();
                }
                break;
            case 'error':
                // Display an error message to the user
                alert(message.error);
                break;
            case 'ping':
                console.log('Pong');
                break;
            case "newRequest":
                // Add the new request to the queue
                queue.push(message.queue[0]);
                if (queue.length === 1) {
                    loadNextInQueue();
                }
                break;
            default:
                console.error('Unknown message type:', message.msgtype);
        }
    });

    socket.addEventListener('close', function(event) {
        console.log('WebSocket connection closed');
        // Reconnect to the WebSocket server after a delay
        setTimeout(function() {
            socket = new WebSocket('ws://' + location.host + '/ws');
        }, 1000);
    });

    socket.addEventListener('error', function(event) {
        console.error('WebSocket error:', event);
    });
});



function handlePass() {
    console.log('Passing request');
    // Send a pass message to the WebSocket server
    socket.send(JSON.stringify({ action: 'pop' }));
    loadNextInQueue();
}

function loadNextInQueue() {
    // Load the next item in the queue
    if (queue.length > 0) {
        const item = queue.shift();
        console.log('Loading item:', item);
        document.getElementById("url").value = item.url;
        document.getElementById("body").value = item.body;
        document.getElementById("method").value = item.method;
        document.getElementById("headers").value = JSON.stringify(item.headers);
    } else {
        document.getElementById('url').value = 'Queue is empty';
        document.getElementById('body').value = '';
        document.getElementById('method').value = '';
        document.getElementById('headers').value = '';
    }
}