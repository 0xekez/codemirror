;; Streams the contents of the current buffer over a websocket
;; connection. The process for creating a connection and then
;; streaming buffer contents over it is as follows:
;;
;;   1. Call `init-websocket-connection' with the url of the websocket
;;      you'd like to connect to.
;;   2. Call `init-websocket-streaming' to start streaming buffer
;;      contents over the connection.
(require 'websocket)

;; The websocket that we are sending buffer and point information
;; over.
(setq sharing-websocket nil)

(defun jsonify-data-msg (contents)
  (let ((myHash (make-hash-table :test 'equal)))
    (puthash "type" "DATA" myHash)
    (puthash "content" (buffer-string) myHash)
    (json-serialize myHash)))

(defun jsonify-selection-msg ()
  (let ((myHash (make-hash-table :test 'equal))
	(startPos (min (mark) (point)))
	(linesBetween (abs (- (line-number-at-pos (mark)) (line-number-at-pos (point))))))
    (puthash "type" "SELECTION" myHash)
    (puthash "content" (if mark-active
			   (format "%s %s"
				   (- startPos (line-number-at-pos startPos))
				   (- (abs (- (mark) (point))) linesBetween))
			 "") myHash)
    (json-serialize myHash)))

(defun jsonify-point-msg ()
  (let ((myHash (make-hash-table :test 'equal)))
    (puthash "type" "SELECTION" myHash)
    (puthash "content" (format "%s %s" (- (point) (line-number-at-pos (point))) 1) myHash)
    (json-serialize myHash)))

;; Sends the contents of the current buffer over WS.
(defun send-buffer-contents (ws)
  (websocket-send-text ws (jsonify-data-msg (buffer-string))))

;; Get's the point location formatted in a way that lines up with the
;; formatting requirements for our frontend.
(defun get-point-loc ()
  (jsonify-point-msg))

;; Sends the location of the cursor over WS.
(defun send-point (ws)
  (websocket-send-text ws
		       (if (and (mark) mark-active)
			   (jsonify-selection-msg)
			 (jsonify-point-msg))))

;; Sends the buffer contents and the point information over the
;; sharing-websocket connection if it has been initialized.
(defun do-websocket ()
  (when sharing-websocket
    (send-buffer-contents sharing-websocket)
    (send-point sharing-websocket)))

(defun handle-url (url)
  (with-temp-buffer-window "connection initialized" nil (lambda (a b) ())
    (princ "Your new mirroring session is ready. Others can view your session\n")
    (princ "at the following url:\n\n")
    (princ url)))

(defun handle-resend (contents)
  (do-websocket))

(defun handle-server-message (msg)
  (let ((json (json-parse-string msg)))
    (let ((type (gethash "type" json))
	  (contents (gethash "content" json)))
      (cond
       ((string= type "URL") (handle-url contents))
       ((string= type "RESEND") (handle-resend contents))
       (t (message "unrecognized message from mirroring server"))))))

;; Initializes our sharing-websocket connection with a given URL.
(defun init-websocket-connection (url)
  (setq sharing-websocket
        (websocket-open
	 url
         :on-message (lambda (_websocket frame)
                       (handle-server-message (websocket-frame-text frame)))
         :on-close (lambda (_websocket) (message "connection closed")))))

;; Initializes streaming over the current websocket streaming
;; connection.
(defun init-websocket-streaming ()
    (when sharing-websocket
      (add-hook 'post-command-hook 'do-websocket)))

(defun start-mirroring ()
  (interactive)
  (init-websocket-connection "ws://localhost:8080/create")
  (init-websocket-streaming)
  (message "started a mirroring session"))


(defun stop-mirroring ()
  (interactive)
  (websocket-close sharing-websocket)
  (remove-hook 'post-command-hook 'do-websocket))
