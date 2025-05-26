((nil . ((eval . (progn
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
		   (load-file "lisp/codetabs.el")
		   (add-hook 'htmlize-before-hook 'codetabs-htmlize-preprocess-fix)
		   (advice-add 'org-html-src-block :around #'codetabs-src-block-advice)))))
 (org-mode . ((eval . (progn
			(setq-default org-html-htmlize-output-type 'css))))))
