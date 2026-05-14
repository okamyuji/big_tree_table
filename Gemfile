source "https://rubygems.org"

ruby file: ".ruby-version"

gem "rails", "~> 8.1.3"
gem "propshaft"
gem "mysql2", "~> 0.5"
gem "puma", ">= 5.0"
gem "importmap-rails"
gem "turbo-rails"
gem "stimulus-rails"
gem "tzinfo-data", platforms: %i[windows jruby]
gem "solid_cache"
gem "solid_queue"
gem "bootsnap", require: false
gem "thruster", require: false

# Allow the existing React frontend (different origin) to call the JSON API.
gem "rack-cors"

group :development, :test do
  gem "debug", platforms: %i[mri windows], require: "debug/prelude"
  gem "bundler-audit", require: false
  gem "brakeman", require: false

  # Static type checker.
  gem "sorbet-static-and-runtime"
  gem "tapioca", require: false

  # Lint / format.
  gem "rubocop-rails-omakase", require: false
  gem "rubocop-minitest", require: false
  gem "rubocop-performance", require: false

  # N+1 detector — always-on in development/test, OFF in production
  # (see config/initializers/bullet.rb for rationale).
  gem "bullet"
end

group :development do
  gem "web-console"
end

group :test do
  gem "capybara"
  gem "selenium-webdriver"

  # Coverage reporting; the suite enforces a >= 80% line coverage floor.
  gem "simplecov", require: false
  gem "minitest-reporters"
end
