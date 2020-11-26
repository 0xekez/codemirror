import * as vscode from 'vscode';
import WebSocket from 'ws';
import lineColumn from 'line-column';

enum MessageType {
  data = 'DATA',
  url = 'URL',
  resend = 'RESEND',
  selection = 'SELECTION',
};

interface Message {
  type: MessageType;
  content: string;
};

let ws: WebSocket | null = null;
let wsUrl: string | null = null;

const closeWs = () => ws?.close();

const displayNoActiveSession = () => vscode.window.showErrorMessage("No active sharing session");

const displayWsUrl = () =>
  wsUrl === null
    ? displayNoActiveSession()
    : vscode.window
      .showInformationMessage(wsUrl, 'Copy to Clipboard')
      .then(() => vscode.env.clipboard.writeText(wsUrl || ''));

// Create new session
const create = (context: vscode.ExtensionContext) => {
  ws = new WebSocket('wss://mirror.noahsaso.com/create');

  const send = (type: MessageType, content: string) => ws?.send(JSON.stringify({ type, content }));
  const updateSelection = () => {
    if (ws === null) { return; }
    const selectionRange = vscode.window.activeTextEditor?.selection?.with();
    if (selectionRange === undefined) { return; }

    const text = vscode.window.activeTextEditor?.document?.getText() || '';
    const textLC = lineColumn(text, { origin: 0 });
    const start = textLC.toIndex(selectionRange.start.line, selectionRange.start.character);
    let end = textLC.toIndex(selectionRange.end.line, selectionRange.end.character);
    if (end < 0) {end = text.length;}

    send(MessageType.selection, `${start} ${end - start}`);
  };
  const updateDataAndSelection = () => {
    if (ws === null) { return; }
    const text = vscode.window.activeTextEditor?.document?.getText();
    if (text === undefined) { return; }
    send(MessageType.data, text);
    updateSelection();
  };

  ws.onmessage = (event) => {
    const msg = JSON.parse(event.data.toString('utf8')) as Message;
    // If sent URL back, display to user.
    if (msg.type === MessageType.url) {
      wsUrl = msg.content;
      displayWsUrl();
    } else if (msg.type === MessageType.resend) {
      // Send latest code changes (for new clients)
      updateDataAndSelection();
    }
  };

  ws.onopen = () => vscode.window.showInformationMessage("Sharing session started.");
  ws.onerror = (event) => vscode.window.showErrorMessage(event.error);
  ws.onclose = () => {
    ws = null;
    wsUrl = null;
    vscode.window.showInformationMessage("Sharing session closed.");
  };

  context.subscriptions.push(...[
    vscode.window.onDidChangeActiveTextEditor(updateDataAndSelection),
    vscode.workspace.onDidChangeTextDocument(updateDataAndSelection),
    vscode.window.onDidChangeTextEditorSelection(updateSelection)
  ]);
};

export function activate(context: vscode.ExtensionContext) {
  // Create new session
  const createSessionDisposable = vscode.commands.registerCommand('cm.createSession', () => {
    if (ws) {
      vscode.window
        .showInformationMessage(
          'There is already an active sharing session. Would you like to close it and start a new one?',
          'No, Keep Existing',
          'Yes, Close Existing'
        )
        .then(selected => {
          if (selected?.startsWith('Yes')) {
            ws?.close();
            create(context);
          }
        });
    } else {
      create(context);
    }
  });
  // Show existing session URL
  const showSessionUrlDisposable = vscode.commands.registerCommand('cm.showSessionUrl', displayWsUrl);
  // Close existing session
  const closeSessionDisposable = vscode.commands.registerCommand('cm.closeSession', () => 
    ws === null
      ? displayNoActiveSession()
      : closeWs()
  );

  context.subscriptions.push(createSessionDisposable);
  context.subscriptions.push(showSessionUrlDisposable);
  context.subscriptions.push(closeSessionDisposable);
}

// this method is called when your extension is deactivated
export function deactivate() {
  closeWs();
}
