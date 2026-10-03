package infrastructure

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/divinecoid/one-backend/internal/foundation/config"
)

// rewriteTransport sends every request to the httptest server, keeping path/query.
type rewriteTransport struct{ target *url.URL }

func (r rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme, req.URL.Host = r.target.Scheme, r.target.Host
	return http.DefaultTransport.RoundTrip(req)
}

func newMockTikTok(t *testing.T, h http.HandlerFunc) *TikTokShopClient {
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	c := NewTikTokShopClient(config.TikTokShopConfig{AppKey: "k", AppSecret: "s"})
	c.httpClient = &http.Client{Transport: rewriteTransport{u}}
	return c
}

func TestTikTokShopsAndProducts(t *testing.T) {
	page := 0
	c := newMockTikTok(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("sign") == "" || q.Get("app_key") != "k" || r.Header.Get("x-tts-access-token") != "tok" {
			t.Errorf("unsigned/unauthenticated request to %s", r.URL.Path)
		}
		switch r.URL.Path {
		case "/authorization/202309/shops":
			io.WriteString(w, `{"code":0,"data":{"shops":[{"id":"7495000000000000001","name":"Toko ID","cipher":"CIPHER1","region":"ID"}]}}`)
		case "/product/202309/products/search":
			if q.Get("shop_cipher") != "CIPHER1" {
				t.Errorf("missing shop_cipher")
			}
			page++
			if page == 1 {
				io.WriteString(w, `{"code":0,"data":{"next_page_token":"p2","products":[{"id":"1729000000000000001","title":"Kaos","skus":[{"seller_sku":"KAOS-1","price":{"sale_price":"50000"},"inventory":[{"quantity":7}]}]}]}}`)
			} else {
				io.WriteString(w, `{"code":0,"data":{"products":[{"id":"1729000000000000002","title":"Topi","skus":[{"seller_sku":"TOPI-1","price":{"sale_price":"30000"}}]}]}}`)
			}
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	shops, err := c.GetAuthorizedShops(context.Background(), "tok")
	if err != nil || len(shops) != 1 || shops[0].Cipher != "CIPHER1" {
		t.Fatalf("shops=%+v err=%v", shops, err)
	}
	ps, err := c.SearchProducts(context.Background(), "tok", shops[0].Cipher)
	if err != nil || len(ps) != 2 || !strings.HasPrefix(ps[0].ID, "17") || ps[0].SKUs[0].Inventory[0].Quantity != 7 {
		t.Fatalf("products=%+v err=%v", ps, err)
	}
}

func TestTikTokErrorEnvelope(t *testing.T) {
	c := newMockTikTok(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"code":105001,"message":"invalid sign"}`)
	})
	if _, err := c.GetAuthorizedShops(context.Background(), "tok"); err == nil || !strings.Contains(err.Error(), "invalid sign") {
		t.Fatalf("expected envelope error, got %v", err)
	}
}
