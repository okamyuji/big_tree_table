# typed: false

class CreateOrders < ActiveRecord::Migration[8.1]
  def change
    create_table :orders do |t|
      t.string  :order_number,   null: false, limit: 64
      t.string  :order_type,     null: false, limit: 32
      t.date    :order_date,     null: false
      t.string  :customer_name,  null: false, limit: 128
      t.string  :customer_code,  null: false, limit: 64
      t.string  :product_name,   null: false, limit: 128
      t.string  :product_code,   null: false, limit: 64
      t.integer :quantity,       null: false
      t.decimal :unit_price,     null: false, precision: 12, scale: 2
      t.decimal :total_amount,   null: false, precision: 14, scale: 2
      t.string  :status,         null: false, limit: 32
      t.date    :delivery_date,  null: true

      t.timestamps null: false
    end

    add_index :orders, :order_number,  name: "idx_order_number"
    add_index :orders, :order_type,    name: "idx_order_type"
    add_index :orders, :order_date,    name: "idx_order_date"
    add_index :orders, :customer_name, name: "idx_customer_name"
    add_index :orders, :product_name,  name: "idx_product_name"
    add_index :orders, :status,        name: "idx_status"
  end
end
