const MessageType = Object.freeze({
  Data: "DATA",
  Selection: "SELECTION",
});

const ws = new WebSocket(WEBSOCKET_URL);
const context = document.getElementById("contents");
const marker = new Mark(context);

function scrollToCursor() {
  const mark = document.querySelector("mark");
  if (!mark) return;

  mark.scrollIntoView({ behavior: "smooth", block: "center", inline: "nearest" });
}

let followingCursor = false;
function toggleFollowCursor(elem) {
  followingCursor = !followingCursor;
  if (followingCursor) {
      elem.classList.add('active');
      elem.classList.add('accent');
    scrollToCursor();
  } else {
      elem.classList.remove('active');
      elem.classList.remove('accent');
  }
}

// Sets the contents of the page's code blocks to WHAT. Note that this
// will delete any existing code in code block.
function setCodeContents(what) {
    let res = hljs.highlightAuto(what);
    context.className = "hljs " + res.language;
    context.innerHTML = res.value;
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
    let start = parseInt(msg[0]);
      let length = parseInt(msg[1]) || 1;

      // Handle case where we are at end of file.
      if (start >= context.innerText.length) {
	  start = context.innerText.length - 1;
      }
      // We can't highlight newlines so we highlight the previous
      // non-newline character.
      while (context.innerText[start] == "\n") {
	  start -= 1
      }

    marker.markRanges([{
      start: start,
      length: length,
    }]);

    if (followingCursor) scrollToCursor();
  } else {
    console.log("bad message from client");
  }
}

// Default on
// Set followingCursor and element class appropriately
toggleFollowCursor(document.getElementById('follow-cursor'));
