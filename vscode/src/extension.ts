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
let wsURL: string | null = null;
let wsSubscriptions: { dispose(): any }[] = [];

const closeWS = () => ws?.close();

const displayNoActiveSession = () => vscode.window.showErrorMessage("No active mirroring session.");

const openWSURL = () =>
  wsURL === null
    ? displayNoActiveSession()
    : vscode.env.openExternal(vscode.Uri.parse(wsURL));

// Create new session
const createWS = () => {
  closeWS();

  ws = new WebSocket('wss://mirror.chmod4.com/create');

  const send = (type: MessageType, content: string) => ws?.send(JSON.stringify({ type, content }));
  const updateSelection = () => {
    if (ws === null) { return; }
    const selectionRange = vscode.window.activeTextEditor?.selection?.with();
    if (selectionRange === undefined) { return; }

    const text = vscode.window.activeTextEditor?.document?.getText() || '';
    const textLC = lineColumn(text, { origin: 0 });

    let start = textLC.toIndex(selectionRange.start.line, selectionRange.start.character);
    let end = textLC.toIndex(selectionRange.end.line, selectionRange.end.character);
    if (start < 0) {start = text.length;}
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
      wsURL = msg.content;
      openWSURL();
    } else if (msg.type === MessageType.resend) {
      // Send latest code changes (for new clients)
      updateDataAndSelection();
    }
  };

  ws.onopen = () => {
    vscode.window.showInformationMessage("Mirroring session started.");
    wsSubscriptions.push(...[
      vscode.window.onDidChangeActiveTextEditor(updateDataAndSelection),
      vscode.workspace.onDidChangeTextDocument(updateDataAndSelection),
      vscode.window.onDidChangeTextEditorSelection(updateSelection)
    ]);
  };
  ws.onerror = (event) => vscode.window.showErrorMessage(event.error);
  ws.onclose = () => {
    ws = null;
    wsURL = null;
    wsSubscriptions.forEach((sub) => sub.dispose());
    wsSubscriptions = [];
    vscode.window.showInformationMessage("Mirroring session closed.");
  };
};

export function activate(context: vscode.ExtensionContext) {
  // Create new session
  const createSessionDisposable = vscode.commands.registerCommand('cm.createSession', createWS);
  // Show existing session URL
  const viewSessionDisposable = vscode.commands.registerCommand('cm.viewSession', openWSURL);
  // Close existing session
  const closeSessionDisposable = vscode.commands.registerCommand('cm.closeSession', () =>
    ws === null
      ? displayNoActiveSession()
      : closeWS()
  );

  context.subscriptions.push(createSessionDisposable);
  context.subscriptions.push(viewSessionDisposable);
  context.subscriptions.push(closeSessionDisposable);
}

// this method is called when your extension is deactivated
export function deactivate() {
  closeWS();
}
