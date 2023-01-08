JEKYLL_ENV=production bundle exec jekyll build
purgecss --css _site/style.css --content _site/**/*.html --output _site
git checkout master
rsync -a -v _site/ ./
git checkout working