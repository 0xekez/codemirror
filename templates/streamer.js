// Set's the contents of the page's code blocks to WHAT. Note that this
// will delete any existing code in code block.
function setCodeContents(what) {
    code = document.getElementById("contents")
    code.innerHTML = what
    hljs.initHighlighting.called = false
    hljs.initHighlighting()
}

const ws = new WebSocket("ws://localhost:3001")

ws.onmessage = function (event) {
    if (event.data.startsWith("DATA")) {
	// Remove the first four characters
	setCodeContents(event.data.slice(4))
    } else if (event.data.startsWith("POINT")) {
	// change cursor contents
	let info = event.data.slice(5)
	let l = info.split(" ")
	if (l.length != 2) {
	    console.log("bad message from client")
	    return;
	}
	let line = parseInt(l[0])
	let col = parseInt(l[1])
	cursor = document.getElementById("cursor")
	// FIXME: this fails when line or col minus one is zero.
	cursor.innerHTML = "\n".repeat(line - 1) + " ".repeat(col - 1) + "█"
	console.log("point")
    } else {
	console.log("bad message from client")
    }
}
