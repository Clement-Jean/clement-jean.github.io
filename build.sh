cd _site && JEKYLL_ENV=production bundle exec jekyll build && cd ..
git checkout master
mv _site/* .
git checkout working