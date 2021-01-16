(require 'websocket)

;; The websocket that we are sending buffer and point information
;; over.
(setq sharing-websocket nil)

;; Converts an emacs index into the buffer into a browser index into
;; the buffer.
(defun to-browser-index (loc)
  (- loc 1))

(defun jsonify-data-msg (contents)
  (let ((myHash (make-hash-table :test 'equal)))
    (puthash "type" "DATA" myHash)
    (puthash "content" (buffer-string) myHash)
    (json-serialize myHash)))

(defun jsonify-selection-msg ()
  (let ((myHash (make-hash-table :test 'equal))
	(startPos (min (mark) (point))))
    (puthash "type" "SELECTION" myHash)
    (puthash "content" (if mark-active
			   (format "%s %s"
				   (to-browser-index startPos)
				   (abs (-
					 (to-browser-index (mark))
					 (to-browser-index (point)))))
			 "") myHash)
    (json-serialize myHash)))

(defun jsonify-point-msg ()
  (let ((myHash (make-hash-table :test 'equal)))
    (puthash "type" "SELECTION" myHash)
    (puthash "content" (format "%s %s" (to-browser-index (point)) 1) myHash)
    (json-serialize myHash)))

(setq last-buffer-string nil)

(defun need-buffer-update ()
  (let ((current (buffer-string)))
    (if (equal current last-buffer-string)
	nil
      (setq last-buffer-string current))))

;; Sends the contents of the current buffer over WS.
(defun send-buffer-contents (ws)
  (when (need-buffer-update)
    (websocket-send-text ws (jsonify-data-msg (buffer-string)))))

;; Get's the point location formatted in a way that lines up with the
;; formatting requirements for our frontend.
(defun get-point-loc ()
  (jsonify-point-msg))

(defun get-point-state ()
  (list (point) mark-active (mark)))

(setq last-point-state (get-point-state))

(defun point-needs-update ()
  (let ((point-state (get-point-state)))
    (if (equal point-state last-point-state)
	nil
      (setq last-point-state point-state))))

;; Sends the location of the cursor over WS.
(defun send-point (ws)
  (when (point-needs-update)
    (websocket-send-text ws
			 (if (and (mark) mark-active)
			     (jsonify-selection-msg)
			   (jsonify-point-msg)))))

;; Sends the buffer contents and the point information over the
;; sharing-websocket connection if it has been initialized.
(defun do-websocket ()
  (when (websocket-openp sharing-websocket)
    (send-buffer-contents sharing-websocket)
    (send-point sharing-websocket)))

(defun handle-url (url)
  (browse-url url))
  ;; (with-temp-buffer-window "connection initialized" nil (lambda (a b) ())
  ;;   (princ "Your new mirroring session is ready. Others can view your session\n")
  ;;   (princ "at the following url:\n\n")
  ;;   (princ url)))

(defun handle-resend (contents)
  ;; Reset the cached buffer information
  (setq last-point-state nil)
  (setq last-buffer-string nil)
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
  (setq last-buffer-string nil)
  (setq last-point-state nil)
  (init-websocket-connection "wss://mirror.chmod4.com/create")
  (init-websocket-streaming)
  (message "started a mirroring session"))


(defun stop-mirroring ()
  (interactive)
  (websocket-close sharing-websocket)
  (remove-hook 'post-command-hook 'do-websocket))
