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
# initialiser default of "localhost:3000".
Rails.application.config.hosts << "backend" if Rails.application.config.respond_to?(:hosts)
