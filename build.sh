JEKYLL_ENV=production bundle exec jekyll build
git checkout master
purgecss --css _site/style.css --content _site/**/*.html --output _site
rsync -a -v _site/ ./
git checkout working
