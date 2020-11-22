import { start } from 'repl';
import * as vscode from 'vscode';
import * as WebSocket from 'ws';

enum MessageType {
  data = 'DATA',
  cursor = 'CURSOR',
  url = 'URL',
  resend = 'RESEND',
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
const create = () => {
  ws = new WebSocket('ws://localhost:8080/create');

  const send = (type: MessageType, content: string) => ws?.send(JSON.stringify({ type, content }));
  const updateData = () => {
    if (ws === null) { return; }
    const text = vscode.window.activeTextEditor?.document?.getText();
    if (text === undefined) { return; }
    send(MessageType.data, text);
  };
  const updateCursor = () => {
    if (ws === null) { return; }
    const pos = vscode.window.activeTextEditor?.selection.active;
    if (pos === undefined) { return; }
    send(MessageType.cursor, `${pos.line} ${pos.character}`);
  };

  ws.onmessage = (event) => {
    const msg = JSON.parse(event.data.toString('utf8')) as Message;
    // If sent URL back, display to user.
    if (msg.type === MessageType.url) {
      wsUrl = msg.content;
      displayWsUrl();
    } else if (msg.type === MessageType.resend) {
      // Send latest code changes (for new clients)
      updateData();
      updateCursor();
    }
  };

  ws.onopen = () => vscode.window.showInformationMessage("Sharing session started.");
  ws.onerror = (event) => vscode.window.showErrorMessage(event.error);
  ws.onclose = () => {
    ws = null;
    wsUrl = null;
    vscode.window.showInformationMessage("Sharing session closed.");
  };

  vscode.window.onDidChangeActiveTextEditor(updateData);
  vscode.workspace.onDidChangeTextDocument(updateData);
  vscode.window.onDidChangeActiveTextEditor(updateCursor);
  vscode.window.onDidChangeTextEditorSelection(updateCursor);
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
            create();
          }
        });
    } else {
      create();
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
