cd _site && JEKYLL_ENV=production bundle exec jekyll build && cd ..
git checkout master
rm -rf images
mv -f _site/* .
git checkout working