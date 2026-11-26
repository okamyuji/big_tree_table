# Benchmark environment: production settings against the shared PostgreSQL in
# ../bench, served over plain HTTP on loopback. The rate-limit and auth-failure
# counters live in process memory, as they do in the Go and snails versions.
require_relative "production"

Rails.application.configure do
  config.assume_ssl = false
  config.force_ssl  = false
  config.cache_store = :memory_store
  config.active_job.queue_adapter = :async
  config.solid_queue.connects_to = nil
end
