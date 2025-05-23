((nil . ((eval . (progn
		  (load-file "lisp/codetabs.el")
		  (advice-add 'org-html-src-block :around #'codetabs-src-block-advice)))))
 (org-mode . ((eval . (progn
			(setq-default org-html-htmlize-output-type 'css))))))
