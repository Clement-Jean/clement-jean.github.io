bundle init
bundle add jekyll --version "~>4.2"
bundle config set --local path "vendor/bundle"
bundle install
bundle add webrick jekyll-sitemap jekyll-feed jdvp-codetabs-commonmark
bundle install
bundle update