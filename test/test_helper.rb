# typed: false

ENV["RAILS_ENV"] ||= "test"

# SimpleCov must boot before any application code is loaded so it can hook
# into the require chain. The 80% line floor enforces the project gate.
require "simplecov"
SimpleCov.start "rails" do
  enable_coverage :branch
  add_filter "/test/"
  add_filter "/config/"
  add_filter "/db/"
  add_filter "/bin/"
  add_filter "/vendor/"
  # The boot wiring (application/boot/environment) is plumbing — exclude it
  # from the coverage floor so it doesn't dilute the score we actually care
  # about (app/ logic).
  add_filter "/config/application.rb"
  add_filter "/config/environment.rb"
  add_filter "/config/boot.rb"
  add_filter "/config/puma.rb"

  minimum_coverage 80
end

require_relative "../config/environment"
require "rails/test_help"

require "minitest/reporters"
Minitest::Reporters.use!(
  Minitest::Reporters::ProgressReporter.new(color: true),
  ENV,
  Minitest.backtrace_filter
)

module ActiveSupport
  class TestCase
    # Parallel workers conflict with SimpleCov's single-process coverage merge,
    # so run serially. Suite is small enough that this isn't a wall-clock issue.
    parallelize(workers: 1)

    fixtures :all

    # Wrap every test in Bullet.profile so any N+1 / unused-eager-load /
    # missing-counter-cache regression fails the test (config/initializers/bullet.rb
    # sets Bullet.raise = true in the test env).
    setup do
      Bullet.start_request if defined?(Bullet) && Bullet.enable?
    end

    teardown do
      if defined?(Bullet) && Bullet.enable?
        Bullet.perform_out_of_channel_notifications if Bullet.notification?
        Bullet.end_request
      end
    end
  end
end
