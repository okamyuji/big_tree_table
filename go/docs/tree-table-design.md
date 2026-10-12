# BigTreeTable Design

## Goal

BigTable と並列の TreeTable 版として、折りたたみ可能な階層行をバックエンドのレスポンス構造から扱う。既存の100万行 `orders` テーブルと `/api/orders` は残し、TreeTable 用に `/api/order-tree` を追加する。

## Data Contract

TreeTable は平坦な `Order[]` ではなく、次の階層データを受け取る。

```json
{
  "data": [
    {
      "id": "customer:CST-001",
      "kind": "customer",
      "depth": 0,
      "label": "田中製作所",
      "summary": { "order_count": 4, "quantity": 120, "total_amount": 510000, "statuses": ["受注確認"] },
      "children": []
    }
  ],
  "total": 1000000,
  "page": 1,
  "per_page": 25,
  "total_pages": 40000
}
```

階層は `customer -> product -> order` とする。親行は集計、葉行は注文実体を持つ。

## Table Design

物理DBは既存の `orders` fact table を維持する。理由は以下。

- 既存記事の主題である100万行テーブル、ソート、フィルター、ページネーションの検証を壊さない。
- TreeTable用の階層は `customer_code`、`product_code`、`id` から決定できる。
- バックエンドでレスポンス構造を変更すれば、フロントはTreeTable専用のデータ契約で実装できる。

将来、顧客・商品マスタを個別管理する必要が出た場合は `customers`、`products`、`orders`、`order_lines` に正規化する。ただし今回の説明用アプリでは、物理スキーマ変更よりAPI構造変更のほうが目的に対して小さく安全。

## Backend

- `GET /api/orders`: 既存互換の平坦なテーブルAPI。
- `GET /api/order-tree`: TreeTable用API。

`/api/order-tree` は既存のクエリ条件でページ分の注文を取得し、サーバー内で `customer -> product -> order` に変換して返す。ページング単位は注文行のまま維持する。つまり「現在ページに含まれる注文の階層」を返す。

## Frontend

1. `useTreeTableData` が `/api/order-tree` から階層レスポンスを取得する。
2. `expandedNodeIds: Set<string>` を保持する。
3. `flattenVisibleTree(data, expandedNodeIds)` が展開状態を反映した一次元配列を作る。
4. 既存 `VirtualScroller` に visible node count を渡す。
5. `TreeTableRow` が `depth`、`kind`、`summary`、`order` を使って表示する。

## Interaction

- 展開可能行はボタンで開閉する。
- `aria-expanded` は展開可能行だけに付与する。
- `aria-level` は `depth + 1` とする。
- 「すべて展開」「すべて折りたたみ」を提供する。
- 全行の高さは `40px` 固定にして仮想スクロールの計算を安定させる。

## Self Review

- Problem: ページング単位が注文行のままだと、同じ顧客が複数ページに分かれる。
  Resolution: UIに「現在ページの受注を階層表示」と明記する。既存API性能デモとの整合性を優先する。

- Problem: フロントだけで階層化するとAPI契約がTreeTableに見えない。
  Resolution: `/api/order-tree` を追加し、バックエンドが階層レスポンスを返す。

- Problem: 物理スキーマまで正規化すると記事用アプリとして差分が大きすぎる。
  Resolution: 今回は `orders` fact table を維持し、必要になった場合の正規化方針を設計書に残す。

- Problem: 深い階層でインデントが崩れる。
  Resolution: 行は任意階層を受け入れるが、表示インデントは上限付きにする。

- Problem: ソート対象が親行と葉行で意味を持ちにくい。
  Resolution: ソートは注文データに対してサーバー側で実行し、その結果ページを階層化する。

## Decision

TreeTableに合うデータ構造はバックエンドレスポンスとして実装する。物理テーブルは現時点では維持し、必要な場合の正規化設計を将来方針として明記する。
