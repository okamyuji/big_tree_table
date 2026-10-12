# typed: false

Rails.application.routes.draw do
  get "up" => "rails/health#show", as: :rails_health_check

  namespace :api do
    namespace :v1 do
      resources :orders, only: [:index] do
        collection do
          get :tree
        end
      end
    end
  end
end
