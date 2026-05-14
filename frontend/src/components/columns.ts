export interface Column {
  key: string;
  label: string;
  width: string;
}

export const COLUMNS: Column[] = [
  { key: "order_number", label: "注文番号", width: "110px" },
  { key: "order_type", label: "種別", width: "80px" },
  { key: "order_date", label: "注文日", width: "110px" },
  { key: "customer_name", label: "顧客名", width: "150px" },
  { key: "product_name", label: "商品名", width: "180px" },
  { key: "quantity", label: "数量", width: "80px" },
  { key: "unit_price", label: "単価", width: "100px" },
  { key: "total_amount", label: "金額", width: "120px" },
  { key: "status", label: "ステータス", width: "110px" },
  { key: "delivery_date", label: "納期", width: "110px" },
];
