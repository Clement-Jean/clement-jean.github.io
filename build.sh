JEKYLL_ENV=production bundle exec jekyll build
git checkout master
rsync -a -v _site/ ./
git checkout working
