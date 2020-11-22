const MessageType = Object.freeze({
  Data: "DATA",
  Selection: "SELECTION",
});

const ws = new WebSocket(WEBSOCKET_URL);
const context = document.getElementById("contents");
const marker = new Mark(context);

function scrollToCursor() {
    document.querySelector("mark").scrollIntoView();
}

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

  marker.unmark();

  if (msg.type === MessageType.Data) {
    setCodeContents(msg.content);
  } else if (msg.type == MessageType.Selection) {
    if (msg.content == "") {
      // Nothing to do as there is no selection.
      return;
    }
    msg = msg.content.split(" ");
    if (msg.length != 2) {
      console.log("bad message from client");
      return;
    }
    const start = parseInt(msg[0]);
    const length = parseInt(msg[1]) || 1;

    marker.markRanges([{
      start: start,
      length: length,
    }]);
  } else {
    console.log("bad message from client");
  }
}
