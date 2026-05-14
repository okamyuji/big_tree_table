# typed: false

class AddNotesToOrders < ActiveRecord::Migration[8.1]
  def change
    add_column :orders, :notes, :string, limit: 255, null: false, default: ""
  end
end
