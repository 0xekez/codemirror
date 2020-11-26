import sublime, sublime_plugin
from . import websocket
import ssl
import json
import threading

class EventDump(sublime_plugin.EventListener):
	is_mirroring = False
	socket = None
	create_uri = "wss://mirror.noahsaso.com/create"
	mirror_uri = ""
	listener_thread = None

	def send_view(ws, view):
		EventDump.socket.send(json.dumps(
			{
				"type": "DATA",
				"content": view.substr(sublime.Region(0, view.size())),
			}))

	def send_selection(ws, view):
		selection = view.sel()[0]
		EventDump.socket.send(json.dumps(
			{
				"type": "SELECTION",
				"content": "{} {}".format(selection.begin(), selection.size()),
			}))

	def send_all(ws, view):
		EventDump.send_view(ws, view)
		EventDump.send_selection(ws, view)

	def handle_server_message(ws, message):
		message = json.loads(message)
		if message["type"] == "URL":
			EventDump.mirror_uri = message["content"]
			print("started mirroring on {}".format(EventDump.mirror_uri))
		elif message["type"] == "RESEND":
			EventDump.send_view(EventDump.socket, sublime.active_window().active_view())
		else:
			print("bad message type from server: {}", message["TYPE"])

	def _start_mirroring():
		EventDump.socket = websocket.WebSocketApp("wss://mirror.noahsaso.com/create",
                              on_message = EventDump.handle_server_message,
                              on_error = lambda ws, e: EventDump.stop_mirroring(),
                              on_close = lambda ws: EventDump.stop_mirroring())
		EventDump.is_mirroring = True
		EventDump.socket.run_forever(sslopt={"cert_reqs": ssl.CERT_NONE})

	def start_mirroring():
		t = threading.Thread(target=EventDump._start_mirroring)
		t.start()
		EventDump.listener_thread = t

	def stop_mirroring():
		if EventDump.socket != None:
			EventDump.socket = None
			EventDump.is_mirroring = False

	def on_modified(self, view):
		if EventDump.is_mirroring:
			EventDump.send_view(EventDump.socket, view)

	def on_selection_modified(self, view):
		if EventDump.is_mirroring:
			EventDump.send_selection(EventDump.socket, view)

	def on_activated(self, view):
		if EventDump.is_mirroring:
			EventDump.send_all(EventDump.socket, view)

class StartMirroringCommand(sublime_plugin.TextCommand):
	def run(self, edit):
		EventDump.start_mirroring()

class StopMirroringCommand(sublime_plugin.TextCommand):
	def run(self, edit):
		EventDump.socket.close()
		EventDump.listener_thread.join()
		EventDump.listener_thread = None
		print("stopped mirroring")
