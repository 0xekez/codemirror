// Set's the contents of the page's code blocks to WHAT. Note that this
// will delete any existing code in code block.
function setCodeContents(what) {
  code = document.getElementById("contents")
  code.innerHTML = what
  hljs.initHighlighting.called = false
  hljs.initHighlighting()
}

const MessageType = Object.freeze({
  Data: "DATA",
  Cursor: "CURSOR",
});

const ws = new WebSocket(WEBSOCKET_URL);

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
    const msg = msg.content.split(" ")
    if (msg.length != 2) {
      console.log("bad message from client")
      return;
    }
    let line = parseInt(msg[0])
    let col = parseInt(msg[1])
    cursor = document.getElementById("cursor")
    // FIXME: this fails when line or col minus one is zero.
    cursor.innerHTML = "\n".repeat(line - 1) + " ".repeat(col - 1) + "█"
    console.log("point")
  } else {
    console.log("bad message from client")
  }
}
