package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"bigtable-backend/model"
)

func TestParseV1Params_MatchesRailsNormalization(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  model.QueryParams
	}{
		{"defaults", "", model.QueryParams{Page: 1, PerPage: 50}},
		{"explicit", "page=3&per_page=25&sort=order_date&order=DESC",
			model.QueryParams{Page: 3, PerPage: 25, Sort: "order_date", Order: "DESC"}},
		{"per_page zero falls back to 50", "per_page=0", model.QueryParams{Page: 1, PerPage: 50}},
		{"per_page negative falls back to 50", "per_page=-5", model.QueryParams{Page: 1, PerPage: 50}},
		{"per_page capped at 500", "per_page=501", model.QueryParams{Page: 1, PerPage: 500}},
		{"page below 1 clamps to 1", "page=-4", model.QueryParams{Page: 1, PerPage: 50}},
		{"page capped at MaxPage", "page=99999999999999999999999", model.QueryParams{Page: 1_000_000, PerPage: 50}},
		{"leading digits like String#to_i", "page=12abc&per_page=+7x", model.QueryParams{Page: 12, PerPage: 7}},
		{"leading whitespace like String#to_i", "page=%20%209", model.QueryParams{Page: 9, PerPage: 50}},
		{"underscore between digits like String#to_i", "page=1_0&per_page=2__0", model.QueryParams{Page: 10, PerPage: 2}},
		{"non numeric", "page=abc&per_page=x", model.QueryParams{Page: 1, PerPage: 50}},
		{"valid dates kept", "date_from=2026-01-31&date_to=2026-02-28",
			model.QueryParams{Page: 1, PerPage: 50, DateFrom: "2026-01-31", DateTo: "2026-02-28"}},
		{"impossible dates dropped", "date_from=2026-13-45&date_to=2026-02-30", model.QueryParams{Page: 1, PerPage: 50}},
		{"wrong date shape dropped", "date_from=2026-1-5&date_to=20260105", model.QueryParams{Page: 1, PerPage: 50}},
		{"filters passed through", "order_type=rush&status=shipped&customer_name=%25_%5C&product_name=a%3Fb",
			model.QueryParams{Page: 1, PerPage: 50, OrderType: "rush", Status: "shipped", CustomerName: `%_\`, ProductName: "a?b"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q, err := url.ParseQuery(c.query)
			if err != nil {
				t.Fatal(err)
			}
			if got := parseV1Params(q); got != c.want {
				t.Errorf("parseV1Params(%q) = %+v, want %+v", c.query, got, c.want)
			}
		})
	}
}

func TestFormatCents_MatchesBigDecimalToS(t *testing.T) {
	cases := map[int64]string{
		0:              "0.0",
		10100:          "101.0",
		10150:          "101.5",
		10125:          "101.25",
		10105:          "101.05",
		5:              "0.05",
		-50:            "-0.5",
		-12345:         "-123.45",
		12895800510000: "128958005100.0",
	}
	for cents, want := range cases {
		if got := formatCents(cents); got != want {
			t.Errorf("formatCents(%d) = %q, want %q", cents, got, want)
		}
	}
}

func TestToCents_RoundsFloatNoise(t *testing.T) {
	if got := toCents(0.1 + 0.2); got != 30 {
		t.Errorf("toCents(0.1+0.2) = %d, want 30", got)
	}
	if got := toCents(101.07); got != 10107 {
		t.Errorf("toCents(101.07) = %d, want 10107", got)
	}
}

func sampleOrder(id int64, customer, product, status string, qty int, total float64) model.Order {
	ts := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	delivery := "2026-11-02T00:00:00Z"
	return model.Order{
		ID: id, OrderNumber: "ORD-" + customer + product, OrderType: "rush",
		OrderDate: "2026-10-31T00:00:00Z", CustomerName: "Customer " + customer, CustomerCode: "C" + customer,
		ProductName: "Product " + product, ProductCode: "P" + product, Quantity: qty,
		UnitPrice: 0.1, TotalAmount: total, Status: status, DeliveryDate: &delivery,
		CreatedAt: ts, UpdatedAt: ts.Add(1500 * time.Millisecond),
	}
}

func TestV1OrderPayload_UsesRailsRepresentations(t *testing.T) {
	got := v1OrderFrom(sampleOrder(7, "1", "2", "shipped", 3, 202.5))
	if got.OrderDate != "2026-10-31" || got.DeliveryDate == nil || *got.DeliveryDate != "2026-11-02" {
		t.Errorf("dates = %q, %v", got.OrderDate, got.DeliveryDate)
	}
	if got.UnitPrice != "0.1" || got.TotalAmount != "202.5" {
		t.Errorf("amounts = %q, %q", got.UnitPrice, got.TotalAmount)
	}
	if got.CreatedAt != "2026-11-01T00:00:00.000Z" || got.UpdatedAt != "2026-11-01T00:00:01.500Z" {
		t.Errorf("timestamps = %q, %q", got.CreatedAt, got.UpdatedAt)
	}
}

func TestBuildV1Tree_GroupsAndSumsInCents(t *testing.T) {
	orders := []model.Order{
		sampleOrder(1, "1", "A", "shipped", 1, 0.1),
		sampleOrder(2, "1", "A", "pending", 2, 0.2),
		sampleOrder(3, "1", "B", "shipped", 3, 0.3),
		sampleOrder(4, "2", "A", "shipped", 4, 1),
	}
	got := describeTree(buildV1Tree(orders))
	want := []string{
		"customer customer:C1 d0 Customer 1 n3 q6 t0.6 [shipped pending] order=false",
		"product customer:C1:product:PA d1 Product A n2 q3 t0.3 [shipped pending] order=false",
		"order order:1 d2 ORD-1A n1 q1 t0.1 [shipped] order=true",
		"order order:2 d2 ORD-1A n1 q2 t0.2 [pending] order=true",
		"product customer:C1:product:PB d1 Product B n1 q3 t0.3 [shipped] order=false",
		"order order:3 d2 ORD-1B n1 q3 t0.3 [shipped] order=true",
		"customer customer:C2 d0 Customer 2 n1 q4 t1.0 [shipped] order=false",
		"product customer:C2:product:PA d1 Product A n1 q4 t1.0 [shipped] order=false",
		"order order:4 d2 ORD-2A n1 q4 t1.0 [shipped] order=true",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("tree:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// describeTree flattens the tree depth-first into one line per node.
func describeTree(nodes []v1TreeNode) []string {
	var out []string
	for _, n := range nodes {
		s := n.Summary
		out = append(out, fmt.Sprintf("%s %s d%d %s n%d q%d t%s %v order=%t",
			n.Kind, n.ID, n.Depth, n.Label, s.OrderCount, s.Quantity, s.TotalAmount, s.Statuses, n.Order != nil))
		out = append(out, describeTree(n.Children)...)
	}
	return out
}

type fakeService struct {
	got    model.QueryParams
	orders []model.Order
	total  int64
	err    error
}

func (f *fakeService) GetOrders(p model.QueryParams) (model.OrdersResponse, error) {
	f.got = p
	return model.OrdersResponse{Data: f.orders, Total: f.total, Page: p.Page, PerPage: p.PerPage}, f.err
}

func (f *fakeService) GetOrderTree(p model.QueryParams) (model.OrderTreeResponse, error) {
	panic("v1 handlers build the tree themselves")
}

func TestV1OrdersHandler_Envelope(t *testing.T) {
	svc := &fakeService{orders: []model.Order{sampleOrder(1, "1", "A", "shipped", 1, 2)}, total: 101}
	rec := httptest.NewRecorder()
	V1OrdersHandler(svc).ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/orders?per_page=50&page=2", nil))

	if rec.Code != http.StatusOK || rec.Header().Get("X-Total-Count") != "101" {
		t.Fatalf("status=%d X-Total-Count=%q", rec.Code, rec.Header().Get("X-Total-Count"))
	}
	var body struct {
		Orders []map[string]any `json:"orders"`
		Meta   map[string]int   `json:"meta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Orders) != 1 || body.Orders[0]["total_amount"] != "2.0" {
		t.Errorf("orders = %v", body.Orders)
	}
	want := map[string]int{"total": 101, "page": 2, "per_page": 50, "total_pages": 3}
	for k, v := range want {
		if body.Meta[k] != v {
			t.Errorf("meta[%s] = %d, want %d", k, body.Meta[k], v)
		}
	}
}

func TestV1OrderTreeHandler_Envelope(t *testing.T) {
	svc := &fakeService{orders: []model.Order{sampleOrder(1, "1", "A", "shipped", 1, 2)}, total: 0}
	rec := httptest.NewRecorder()
	V1OrderTreeHandler(svc).ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/orders/tree", nil))

	var body struct {
		Tree []map[string]any `json:"tree"`
		Meta map[string]int   `json:"meta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Tree) != 1 || body.Meta["total_pages"] != 0 || rec.Header().Get("X-Total-Count") != "0" {
		t.Errorf("tree=%v meta=%v", body.Tree, body.Meta)
	}
}

func TestV1Handlers_RejectNonGetAndReportDBErrors(t *testing.T) {
	rec := httptest.NewRecorder()
	V1OrdersHandler(&fakeService{}).ServeHTTP(rec, httptest.NewRequest("POST", "/api/v1/orders", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST status = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	V1OrderTreeHandler(&fakeService{err: errors.New("boom")}).ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/orders/tree", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("error status = %d", rec.Code)
	}
}

func TestV1OrderPayload_NullDeliveryDateStaysNull(t *testing.T) {
	o := sampleOrder(8, "1", "2", "shipped", 1, 1)
	o.DeliveryDate = nil
	b, err := json.Marshal(v1OrderFrom(o))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"delivery_date":null`) {
		t.Errorf("payload = %s, want delivery_date null", b)
	}
}

func TestV1Handlers_RejectNulBytes(t *testing.T) {
	for _, h := range []http.HandlerFunc{V1OrdersHandler(&fakeService{}), V1OrderTreeHandler(&fakeService{})} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/orders?customer_name=a%00b", nil))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	V1OrdersHandler(&fakeService{}).ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/orders?unknown=a%00b", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("a NUL byte outside the permitted keys: status = %d, want 200", rec.Code)
	}
}

func TestV1Handlers_RejectInvalidUTF8(t *testing.T) {
	for _, q := range []string{"customer_name=%FF", "status=%C3%28"} {
		rec := httptest.NewRecorder()
		V1OrdersHandler(&fakeService{}).ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/orders?"+q, nil))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", q, rec.Code)
		}
	}
}

func TestV1Handlers_RepeatedParameterUsesTheLastValueLikeRails(t *testing.T) {
	svc := &fakeService{}
	rec := httptest.NewRecorder()
	V1OrdersHandler(svc).ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/orders?status=%00&status=shipped&page=2&page=3", nil))
	if rec.Code != http.StatusOK || svc.got.Status != "shipped" || svc.got.Page != 3 {
		t.Errorf("status=%d params=%+v, want 200 with the last values", rec.Code, svc.got)
	}
	rec = httptest.NewRecorder()
	V1OrdersHandler(svc).ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/orders?status=shipped&status=%00", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("a NUL byte in the last value: status = %d, want 400", rec.Code)
	}
}
