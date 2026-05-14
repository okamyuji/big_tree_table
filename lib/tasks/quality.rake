# typed: false
# frozen_string_literal: true

# Aggregate "quality" rake task — same checks as bin/quality, exposed inside
# rake so editors / CI runners that prefer rake task discovery can call it
# without shelling out to bin/quality.
namespace :quality do
  desc "rubocop format + lint"
  task :rubocop do
    sh "bundle exec rubocop --no-color"
  end

  desc "Sorbet type check"
  task :sorbet do
    sh "bundle exec srb tc"
  end

  desc "brakeman security scan"
  task :brakeman do
    sh "bundle exec brakeman --no-pager --quiet --exit-on-warn --exit-on-error"
  end

  desc "bundler-audit (CVE scan against Gemfile.lock)"
  task :audit do
    sh "bundle exec bundler-audit check --update"
  end

  desc "Rails test suite (Minitest + SimpleCov 80% floor)"
  task :test do
    sh "bin/rails test"
  end

  desc "Production asset precompile smoke build"
  task :build do
    sh "bin/rails assets:precompile"
  end
end

desc "Run every quality gate (rubocop → sorbet → brakeman → audit → tests → build)"
task quality: %w[
  quality:rubocop
  quality:sorbet
  quality:brakeman
  quality:audit
  quality:test
  quality:build
]
