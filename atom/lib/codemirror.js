'use babel';

import CodemirrorView from './codemirror-view';
import { CompositeDisposable } from 'atom';
import WebSocket from 'ws';
import lineColumn from 'line-column';

const MessageType = Object.freeze({
  data: 'DATA',
  url: 'URL',
  resend: 'RESEND',
  selection: 'SELECTION',
});

export default {

  ws: null,
  wsURL: null,
  wsSubscriptions: [],
  subscriptions: null,

  activate(state) {
    this.subscriptions = new CompositeDisposable();
    this.subscriptions.add(atom.commands.add('atom-workspace', {
      // Create new session
      'codemirror:create-session': () => this.create(),
      // Show existing session URL
      'codemirror:show-session-url': () => this.displayWsURL(),
      // Close existing session
      'codemirror:close-session': () =>
        this.ws === null
          ? this.displayNoActiveSession()
          : this.closeWs(),
    }));
  },

  deactivate() {
    this.closeWs();
    this.subscriptions.dispose();
  },

  serialize() {
    return {};
  },

  displayNoActiveSession() {
    atom.notifications.addError("No active sharing session");
  },

  closeWs() {
    this.ws && this.ws.close();
  },

  displayWsURL() {
    this.wsURL === null
      ? this.displayNoActiveSession()
      : atom.notifications.addInfo(this.wsURL, {
        buttons: [{
          text: 'Copy to Clipboard',
          onDidClick: () => atom.clipboard.write(this.wsURL),
        }]
      });
  },

  create() {
    this.ws = new WebSocket('wss://mirror.noahsaso.com/create');

    const send = (type, content) => this.ws && this.ws.send(JSON.stringify({ type, content }));
    const updateSelection = () => {
      if (this.ws === null) { return; }
      let selectionRange = atom.workspace.getActiveTextEditor();
      selectionRange = selectionRange && selectionRange.getSelectedBufferRange();
      if (selectionRange === undefined) { return; }

      let text = atom.workspace.getActiveTextEditor();
      text = text && text.getText();
      if (text === undefined) { return; }

      console.log(selectionRange);

      const textLC = lineColumn(text, { origin: 0 });
      const start = textLC.toIndex(selectionRange.start.row, selectionRange.start.column);
      let end = textLC.toIndex(selectionRange.end.row, selectionRange.end.column);
      if (end < 0) { end = text.length; }

      send(MessageType.selection, `${start} ${end - start}`);
    };
    const updateDataAndSelection = () => {
      if (this.ws === null) { return; }
      let text = atom.workspace.getActiveTextEditor();
      text = text && text.getText();
      if (text === undefined) { return; }
      send(MessageType.data, text);
      updateSelection();
    };

    this.ws.onmessage = (event) => {
      const msg = JSON.parse(event.data.toString('utf8'));
      // If sent URL back, display to user.
      if (msg.type === MessageType.url) {
        this.wsURL = msg.content;
        this.displayWsURL();
      } else if (msg.type === MessageType.resend) {
        // Send latest code changes (for new clients)
        updateDataAndSelection();
      }
    };

    this.ws.onopen = () => {
      atom.notifications.addInfo("Sharing session started.");
      this.wsSubscriptions.push(...[
        atom.workspace.onDidChangeActiveTextEditor(updateDataAndSelection),
        atom.workspace.observeTextEditors((editor) => {
          if (this.ws === null) { return; }

          updateDataAndSelection();
          this.wsSubscriptions.push(...[
            editor.onDidChange(updateDataAndSelection),
            editor.onDidChangeSelectionRange(updateSelection)
          ]);
        })
      ]);
    };
    this.ws.onerror = (event) => atom.notifications.addError(event.error);
    this.ws.onclose = () => {
      this.ws = null;
      this.wsURL = null;
      this.wsSubscriptions.forEach((sub) => sub.dispose());
      this.wsSubscriptions = [];
      atom.notifications.addInfo("Sharing session closed.");
    };
  }

};
