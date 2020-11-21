;; Streams the contents of the current buffer to a remote server. This
;; is slow because it seems to need to run syncronously.
;; (require 'websocket)

;; (setq sharing-websocket nil)

(setq sharing-url nil)

(defun init-sharing (url)
  (interactive "P\nsSharing url:")
  (setq sharing-url url)
  (add-hook 'post-command-hook 'do-sharing nil 'local))

(defun do-sharing ()
  (when sharing-url (post-buffer-contents sharing-url)))

(defun post-buffer-contents (url)
  (let ((url-request-method "POST")
	(url-request-extra-headers
	 '(("Content-Type" . "text")))
	(url-request-data (buffer-string)))
    (url-retrieve url '(lambda (status) nil))))

(init-sharing "http://localhost:8080/")
