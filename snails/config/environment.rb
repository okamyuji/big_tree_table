# App environment — the environment-dependent wiring, run once at boot.
#
# Settings (values) are in config/application.rb; this file decides what to
# wire for the current mode: the logger settings, the job queue adapter, the
# mailer provider, and the boot settings below. It is app-owned — sync never
# rewrites it.
#
# Reopening `Snails::Main` here replaces the default entry behavior: boot
# requires this file last, so the last definition wins.
#
#   module Snails
#     module Main
#       def self.dispatch(req, res)
#         # ... the app's own pipeline
#       end
#     end
#   end

# Boot settings — read them at call time; never capture one into a load-time
# constant.
#
# SNAILS_ENV (development|test|staging|production) is read on first use.
# The process zone: stored UTC instants are presented in this zone, and the
# app reads the zone back as `Snails.app.time_zone`. An unknown name raises
# here, so a typo fails at boot instead of leaving the process in UTC.
Snails.time_zone = "UTC"

# The PostgreSQL adapter binds values by quoting them in the SQL text and
# doubling single quotes only, which is safe only while
# standard_conforming_strings is on. Force it for every session through libpq's
# `options`, whatever the server or database default is.
pg_url = ENV["PG_URL"].to_s
# libpq keeps only one `options`, so appending ours would silently drop the caller's.
if pg_url.include?("options") && !pg_url.include?("standard_conforming_strings")
  abort("PG_URL already sets libpq options; add -c standard_conforming_strings=on to them")
end
unless pg_url.empty? || pg_url.include?("standard_conforming_strings")
  ENV["PG_URL"] = if pg_url.start_with?("postgres://", "postgresql://")
    "#{pg_url}#{pg_url.include?("?") ? "&" : "?"}options=-c%20standard_conforming_strings%3Don"
  else
    "#{pg_url} options='-c standard_conforming_strings=on'"
  end
end

# Signing key — `Snails.app.secret_key_base` is the standard reader. Its
# source is the deployment's SECRET_KEY_BASE, and a missing value raises.
# If this app signs tokens, read it here so a missing key fails at boot:
#
#   Snails.app.secret_key_base

# Logging — every call writes a local line (stderr by default, tagged with
# the request id while a request runs) and, once a transport is set, a
# structured record that the job stack delivers to the destination in
# batches. Without a level the mode decides: development writes debug,
# every other mode writes info and above. There is no level env variable —
# set the slots here.
# Snails.logger.level            = :info
# The benchmark sets SNAILS_LOG_LEVEL=warn so request lines are not written,
# as RAILS_LOG_LEVEL=warn does for Rails.
Snails.logger.level = :warn if ENV["SNAILS_LOG_LEVEL"] == "warn"
# Snails.logger.output           = $stdout
# Snails.logger.formatter        = MyFormatter.new
# Snails.logger.record_formatter = MyRecordFormatter.new
# Snails.logger.progname         = "myapp"
#
# Remote delivery stays off until a transport is set; the core ships an
# OTLP/HTTP one. Its endpoint is the receiver's base address — the transport
# appends the logs path itself. Endpoint and token are app-owned and come
# from the environment — never write their values into this file:
#
#   Snails.logger.host_name = "api-1"
#   Snails.logger.transport = Snails::Logger::OTLPTransport.new(
#     endpoint: ENV.fetch("OTLP_ENDPOINT"),
#     token: ENV.fetch("OTLP_TOKEN")
#   )

# Job queue — in-process by default; any class with the adapter interface
# can be selected here (`snails-core/active_job/base.rb`).
ActiveJob.queue_adapter = ActiveJob::ProcessQueue

# Mail delivery — register a provider here, or CaptureMethod to collect
# without sending. A delivery with nothing registered raises.
# In test mode the harness captures deliveries into tmp/mail-capture.jsonl
# instead of sending.
# ActionMailer::Base.delivery_method = MyProvider
ActionMailer::Base.delivery_method = ActionMailer::DeliveryLog if Snails.env.test?
