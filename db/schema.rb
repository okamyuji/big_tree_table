# This file is auto-generated from the current state of the database. Instead
# of editing this file, please use the migrations feature of Active Record to
# incrementally modify your database, and then regenerate this schema definition.
#
# This file is the source Rails uses to define your schema when running `bin/rails
# db:schema:load`. When creating a new database, `bin/rails db:schema:load` tends to
# be faster and is potentially less error prone than running all of your
# migrations from scratch. Old migrations may fail to apply correctly if those
# migrations use external dependencies or application code.
#
# It's strongly recommended that you check this file into your version control system.

ActiveRecord::Schema[8.1].define(version: 2026_05_14_125233) do
  create_table "orders", charset: "utf8mb4", collation: "utf8mb4_0900_ai_ci", force: :cascade do |t|
    t.datetime "created_at", null: false
    t.string "customer_code", limit: 64, null: false
    t.string "customer_name", limit: 128, null: false
    t.date "delivery_date"
    t.string "notes", default: "", null: false
    t.date "order_date", null: false
    t.string "order_number", limit: 64, null: false
    t.string "order_type", limit: 32, null: false
    t.string "product_code", limit: 64, null: false
    t.string "product_name", limit: 128, null: false
    t.integer "quantity", null: false
    t.string "status", limit: 32, null: false
    t.decimal "total_amount", precision: 14, scale: 2, null: false
    t.decimal "unit_price", precision: 12, scale: 2, null: false
    t.datetime "updated_at", null: false
    t.index ["customer_name"], name: "idx_customer_name"
    t.index ["order_date"], name: "idx_order_date"
    t.index ["order_number"], name: "idx_order_number"
    t.index ["order_type"], name: "idx_order_type"
    t.index ["product_name"], name: "idx_product_name"
    t.index ["status"], name: "idx_status"
  end
end
