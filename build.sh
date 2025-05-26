set -o pipefail

JEKYLL_ENV=production bundle exec jekyll build
/Applications/Emacs.app/Contents/MacOS/Emacs -nw -Q --script build-site.el
git checkout master
purgecss --css _site/css/*.css --content _site/**/*.html --output _site/css
rsync --exclude src -a -v _site/ ./
