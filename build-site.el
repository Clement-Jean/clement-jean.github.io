;; Set the package installation directory so that packages aren't stored in the
;; ~/.emacs.d/elpa path.
(require 'package)
(setq package-user-dir (expand-file-name "./.packages"))
(setq package-archives '(("melpa" . "https://melpa.org/packages/")
                         ("elpa" . "https://elpa.gnu.org/packages/")))

;; Initialize the package system
(package-initialize)
(unless package-archive-contents
  (package-refresh-contents))

;; Install dependencies
(package-install 'htmlize)

;; Load codetabs.el
(load-file "_org/lisp/codetabs.el")
(advice-add 'org-html-src-block :around #'codetabs-src-block-advice)

;; Load the publishing system
(require 'ox-publish)

;; Define the publishing project
  (setq org-publish-project-alist
      '(("clement-jean.github.io"
         :base-directory "~/Git/clement-jean.github.io/_org"
         :base-extension "org"
         :publishing-directory "~/Git/clement-jean.github.io/_posts/"
         :recursive t
         :publishing-function org-html-publish-to-html
         :headline-levels 4
         :html-extension "html"
         :body-only t)))

;; Generate the site output
(org-publish "clement-jean.github.io")

(message "Build complete!")
