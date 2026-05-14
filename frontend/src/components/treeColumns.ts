export interface TreeColumn {
  key: string;
  label: string;
  width: string;
  sortable?: boolean;
  sortKey?: string;
}

export const TREE_COLUMNS: TreeColumn[] = [
  {
    key: "label",
    label: "階層 / 注文番号",
    width: "260px",
    sortable: true,
    sortKey: "order_number",
  },
  { key: "kind", label: "行種別", width: "90px" },
  { key: "order_date", label: "注文日", width: "110px", sortable: true, sortKey: "order_date" },
  {
    key: "customer_name",
    label: "顧客名",
    width: "150px",
    sortable: true,
    sortKey: "customer_name",
  },
  { key: "product_name", label: "商品名", width: "180px", sortable: true, sortKey: "product_name" },
  { key: "quantity", label: "数量", width: "90px", sortable: true, sortKey: "quantity" },
  { key: "unit_price", label: "単価", width: "100px", sortable: true, sortKey: "unit_price" },
  { key: "total_amount", label: "金額", width: "120px", sortable: true, sortKey: "total_amount" },
  { key: "status", label: "ステータス", width: "130px", sortable: true, sortKey: "status" },
  { key: "delivery_date", label: "納期", width: "110px", sortable: true, sortKey: "delivery_date" },
];
