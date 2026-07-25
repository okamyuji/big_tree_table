# Allow the existing React frontend (Vite dev server, CRA, etc.) to call the JSON API
# without modification. Origins are read from BIG_TREE_TABLE_CORS_ORIGINS (comma-separated)
# and fall back to common local frontend ports.
Rails.application.config.middleware.insert_before 0, Rack::Cors do
  origins = ENV.fetch(
    "BIG_TREE_TABLE_CORS_ORIGINS",
    "http://localhost:3000,http://localhost:5173,http://127.0.0.1:3000,http://127.0.0.1:5173"
  ).split(",").map(&:strip).reject(&:empty?)

  allow do
    origins(*origins)

    resource "/api/*",
      headers: :any,
      methods: %i[get options head],
      expose: %w[X-Total-Count]
  end
end

# Permit Rails to receive Vite-proxied requests on dev hosts beyond the
# initialiser default of "localhost:3000". The Vite dev server reaches the API
# as http://backend:3000 inside the compose network, so "backend" arrives as the
# Host header.
#
# development only. Rails seeds config.hosts with localhost and friends in
# development, so adding one entry there just widens an existing allow list.
# In production config.hosts starts empty, and adding a single entry turns
# HostAuthorization on with "backend" as the only permitted Host — every other
# Host, including the real deployment domain, then gets a 403.
if Rails.env.development? && Rails.application.config.respond_to?(:hosts)
  Rails.application.config.hosts << "backend"
end
