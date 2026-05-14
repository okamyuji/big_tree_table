# typed: false

# Bullet — N+1 query / unused-eager-load / counter-cache detector.
#
# We enable Bullet in development and test so any new code path that introduces
# an N+1 fails loudly during development AND during the test run (Bullet is
# wired into ActiveSupport::TestCase via Bullet.profile in test_helper.rb).
#
# In production we deliberately leave Bullet OFF:
#
#   - Bullet wraps every ActiveRecord query and adds non-trivial CPU/memory
#     overhead. On a 1M+ row table that pressure becomes measurable.
#   - Bullet's UI integrations (alerts, footer, JS console) are also unwanted
#     in production responses.
#
# If an N+1 is suspected in production, investigate via APM / structured logs
# instead — e.g. enable `ActiveRecord::Base.logger.level = :debug` selectively,
# turn on `rack-mini-profiler` behind an admin flag, or capture a slow-query
# trace from your APM (Datadog, New Relic, Skylight, etc.). Do not flip Bullet
# on globally in production to chase the problem.
if Rails.env.development? || Rails.env.test?
  Rails.application.config.after_initialize do
    Bullet.enable        = true
    Bullet.bullet_logger = true        # log/bullet.log
    Bullet.rails_logger  = true        # surface in development.log

    # In dev: alert visibly so the developer sees it immediately.
    if Rails.env.development?
      Bullet.alert       = false       # JS alert — too noisy for an API server
      Bullet.console     = true        # browser devtools console
      Bullet.add_footer  = false       # API responses, no HTML to inject into
    end

    # In test: turn N+1 into a hard failure so CI catches regressions.
    if Rails.env.test?
      Bullet.raise = true
    end

    # Detection knobs — keep all three on; we want comprehensive coverage.
    Bullet.n_plus_one_query_enable     = true
    Bullet.unused_eager_loading_enable = true
    Bullet.counter_cache_enable        = true
  end
end
