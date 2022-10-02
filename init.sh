bundle init
bundle config set --local path "vendor/bundle"
bundle add jekyll --version "~>4.2"
bundle add webrick --version "~> 1.7"
bundle add jekyll-feed --version "~> 0.16.0"
bundle add jekyll-sitemap --version "~> 1.4"
bundle add jdvp-codetabs-commonmark --version "~> 0.1.2"
bundle install
bundle update