Reflects code being edited in browser to web url. The url can be
shared and visitors will see whatever is being edited in real time
with cursor position and selection highliting. Intended to be used as
a replacement for squinting at code over a Zoom screenshare.

## Implementation

`www/server.go` contains a web server implementation that handles the
creation of sessions. Say for now that the server is running at
`foo.com`.

To create a session, open a websocket connection to
`ws://foo.com/create`. Once the connection has been extablished, the
server will send a json object over the new connection with a URL that
other's can visit to view the session.

There are currently implementations for both Emacs and VSCode in the
`emacs` and `vscode` folders respectively. These both support all of
the features but are likely to be quite buggy.

## Messages from the server

All communcations are json objets with a `type` and `contents`
field. The server is capiable of sending two such messages, one is a
message with type `URL` and contents a url that others can visit to
watch the session. The second message type is `RESEND` which indicates
that the server would like you to resend your editor state. `RESEND`
messages have no content, but the field still exists with an empty
string as its value.

## Messages to the server

Editors can send two message types to the server. They are detailed
below:

```js
{
  type: "DATA"
  // The contents of the buffer that is currently being edited.
  contents: "..."
}
```

```js
{
  type: "SELECTION"
  // The index of the cursor in the file and how long the current
  // selection is. Zero for no selection and only cursor. <start
  // index> does not include newline characters.
  contents: "<start index> <length>"
}
```