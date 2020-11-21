;; Streams the contents of the current buffer over a websocket connection.
(require 'websocket)

(setq sharing-websocket nil)
(setq websocket-server nil)

hello my name is zeke

(defun send-buffer-contents (ws)
  (websocket-send-text ws (format "DATA%s" (buffer-string))))

(defun do-websocket ()
  (when sharing-websocket
    (send-buffer-contents sharing-websocket)
    (send-point sharing-websocket)))

(defun send-point (ws)
  (websocket-send-text ws (get-point-loc)))

(defun init-websocket-server  ()
  (setq  websocket-server
	(websocket-server
	 3001
	 :host 'local
	 ;; When a connection is opened set the sharing websocket to
	 ;; this new one and send the current buffer contents over.
	 :on-open (lambda (ws) (setq sharing-websocket ws) (do-websocket))
	 :on-close (lambda (ws)
		     (message "closing websocket connection")
		     (setq  sharing-websocket nil))))
  (add-hook 'post-command-hook 'do-websocket nil 'local))

(init-websocket-server)

;; (websocket-server-close websocket-server)

(defun get-point-loc ()
  (format "POINT%d %d" (line-number-at-pos) (current-column)))
