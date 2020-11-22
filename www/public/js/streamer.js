const MessageType = Object.freeze({
    Data: "DATA",
    Cursor: "CURSOR",
    Selection: "SELECTION",
});

const ws = new WebSocket(WEBSOCKET_URL);
const context = document.getElementById("contents");
const marker = new Mark(context);


// Sets the contents of the page's code blocks to WHAT. Note that this
// will delete any existing code in code block.
function setCodeContents(what) {
    const code = document.getElementById("contents");
    code.innerText = what;
    hljs.initHighlighting.called = false;
    hljs.initHighlighting();
}

ws.onmessage = function (event) {
    // { type, content }
    let msg;
    try {
	msg = JSON.parse(event.data);
    } catch (err) {
	console.error('bad message from client', err);
	return;
    }

    if (msg.type === MessageType.Data) {
	setCodeContents(msg.content);
    } else if (msg.type === MessageType.Cursor) {
	// change cursor contents
	msg = msg.content.split(" ");
	if (msg.length != 2) {
	    console.log("bad message from client");
	    return;
	}

	const line = parseInt(msg[0]);
	const col = parseInt(msg[1]);
	cursor = document.getElementById("cursor");
	cursor.innerText = "\n".repeat(line) + " ".repeat(col) + "█";
    } else if (msg.type == MessageType.Selection) {
	console.log("selection");
	console.log(msg.content);
	marker.unmark();
	if (msg.content == "") {
	    // Nothing to do as there is no selection.
	    return;
	}
	msg = msg.content.split(" ");
	if (msg.length != 2) {
	    console.log("bad message from client")
	    return;
	}
	const start = parseInt(msg[0])
	const length = parseInt(msg[1])

	marker.markRanges([{
	    start: start,
	    length: length,
	}]);
    } else {
	console.log("bad message from client");
    }
}
