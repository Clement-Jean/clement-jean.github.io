JEKYLL_ENV=production bundle exec jekyll build
git checkout master
mv -f _site/* .
git checkout working