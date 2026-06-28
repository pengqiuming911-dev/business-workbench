package agent

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveCode_Name(t *testing.T) {
	if got := ResolveCode("中证1000"); got != "sh000852" {
		t.Errorf("got %q, want sh000852", got)
	}
	if got := ResolveCode("沪深300"); got != "sh000300" {
		t.Errorf("got %q, want sh000300", got)
	}
}

func TestResolveCode_RawCode(t *testing.T) {
	if got := ResolveCode("sh000852"); got != "sh000852" {
		t.Errorf("got %q, want sh000852", got)
	}
}

func TestResolveCode_Unknown(t *testing.T) {
	if got := ResolveCode("不存在的指数"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestFetchQuote_TencentOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `v_sh000852="1~000852~中证1000~8810.34~8790.50~..."`)
	}))
	t.Cleanup(srv.Close)
	orig := tencentBase
	tencentBase = srv.URL
	t.Cleanup(func() { tencentBase = orig })

	name, price, src, err := FetchQuote("sh000852")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if price != 8810.34 {
		t.Errorf("price = %v, want 8810.34", price)
	}
	if src != "tencent" {
		t.Errorf("source = %q, want tencent", src)
	}
	if name != "中证1000" {
		t.Errorf("name = %q, want 中证1000", name)
	}
}

func TestFetchQuote_FallbackToSina(t *testing.T) {
	// 腾讯返回空 body（无引号段）→ 失败；新浪返回有效 → 胜出
	tencent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "")
	}))
	sina := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `hq_str_sh000852="中证1000,8800,8790,8810.34,..."`)
	}))
	t.Cleanup(tencent.Close)
	t.Cleanup(sina.Close)
	ot, osi := tencentBase, sinaBase
	tencentBase, sinaBase = tencent.URL, sina.URL
	t.Cleanup(func() { tencentBase, sinaBase = ot, osi })

	_, price, src, err := FetchQuote("sh000852")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if price != 8810.34 {
		t.Errorf("price = %v, want 8810.34", price)
	}
	if src != "sina" {
		t.Errorf("source = %q, want sina", src)
	}
}

func TestFetchQuote_AllFail(t *testing.T) {
	tencent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	sina := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	em := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	t.Cleanup(tencent.Close)
	t.Cleanup(sina.Close)
	t.Cleanup(em.Close)
	ot, osi, oe := tencentBase, sinaBase, eastmoneyBase
	tencentBase, sinaBase, eastmoneyBase = tencent.URL, sina.URL, em.URL
	t.Cleanup(func() { tencentBase, sinaBase, eastmoneyBase = ot, osi, oe })

	_, _, _, err := FetchQuote("sh000852")
	if err == nil {
		t.Fatal("三源全败应返回 error")
	}
}
