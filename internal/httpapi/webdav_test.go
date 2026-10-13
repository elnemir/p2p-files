package httpapi

import (
	"encoding/xml"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestWebDAVPathAndXMLBoundaries(t *testing.T) {
	base := davPrefix + "00000000-0000-4000-8000-000000000001/"
	for _, name := range []string{"../x", "a//b", "a%2fb", "a%5Cb", "%2e%2e/x", "NUL.txt", "a?token=secret", "a?", "a#fragment", "/", "e%CC%81.txt"} {
		u, e := url.Parse(base + name)
		if e != nil {
			t.Fatal(e)
		}
		if _, _, e = davPath(u, 1024); e == nil {
			t.Errorf("accepted %q", name)
		}
	}
	for _, name := range []string{"", "%D0%BF%D0%B0%D0%BF%D0%BA%D0%B0/", "a%20%26%20b.txt", "x%252Fy.txt"} {
		u, e := url.Parse(base + name)
		if e != nil {
			t.Fatal(e)
		}
		if _, _, e = davPath(u, 1024); e != nil {
			t.Errorf("rejected %q: %v", name, e)
		}
	}
	for _, raw := range []string{`<!DOCTYPE a [<!ENTITY x SYSTEM "file:///etc/passwd">]><a>&x;</a>`, `<a>`, `<a/><b/>`, `<a>&undefined;</a>`, strings.Repeat("<a>", 17) + strings.Repeat("</a>", 17), strings.Repeat("x", 65537)} {
		if _, e := davXML(httptest.NewRecorder(), httptest.NewRequest("PROPFIND", base, strings.NewReader(raw))); e == nil {
			t.Fatal("unsafe XML accepted")
		}
	}
	r := httptest.NewRequest("PROPFIND", base, strings.NewReader(`<D:propfind xmlns:D="DAV:"><D:prop><D:displayname/><X:custom xmlns:X="urn:custom"/></D:prop></D:propfind>`))
	root, e := davXML(httptest.NewRecorder(), r)
	if e != nil {
		t.Fatal(e)
	}
	props, namesOnly, e := davRequested(root)
	if e != nil || namesOnly || len(props) != 2 || props[1].Space != "urn:custom" {
		t.Fatal("namespace lost", props, e)
	}
	w := httptest.NewRecorder()
	p, _ := davPropertyValue(xml.Name{Space: "DAV:", Local: "displayname"}, davNode{Path: "<untrusted & name>"}, false)
	e = davWriteXML(w, davMultistatus{Responses: []davXMLResponse{{Href: base, Stats: []davPropStat{{Properties: []davProperty{p}, Status: "HTTP/1.1 200 OK"}}}}})
	if e != nil {
		t.Fatal(e)
	}
	var result struct {
		Responses []struct {
			Props []struct {
				Value string `xml:"DAV: prop>displayname"`
			} `xml:"DAV: propstat"`
		} `xml:"DAV: response"`
	}
	if e = xml.Unmarshal(w.Body.Bytes(), &result); e != nil {
		t.Fatal(e)
	}
	if len(result.Responses) != 1 || len(result.Responses[0].Props) != 1 || result.Responses[0].Props[0].Value != "<untrusted & name>" {
		t.Fatal("invalid DAV XML", w.Body.String())
	}
	p = davNamedProperty(xml.Name{Local: "unqualified"})
	encoded, e := xml.Marshal(p)
	if e != nil {
		t.Fatal(e)
	}
	var decoded struct{ XMLName xml.Name }
	if e = xml.Unmarshal(encoded, &decoded); e != nil || decoded.XMLName.Space != "" {
		t.Fatal("unqualified property inherited a namespace", string(encoded), e)
	}
}

func FuzzWebDAVPathAndXML(f *testing.F) {
	for _, seed := range []string{"a.txt", "../secret", `<D:propfind xmlns:D="DAV:"><D:allprop/></D:propfind>`, `<!DOCTYPE x [<!ENTITY x SYSTEM "file:///tmp/private">]><x>&x;</x>`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		if len(value) > 70000 {
			return
		}
		if u, e := url.Parse(davPrefix + "00000000-0000-4000-8000-000000000001/" + value); e == nil {
			id, name, e := davPath(u, 1024)
			if e == nil {
				if id != "00000000-0000-4000-8000-000000000001" || strings.HasPrefix(name, "/") {
					t.Fatal("path escaped namespace")
				}
			}
		}
		root, e := davXML(httptest.NewRecorder(), httptest.NewRequest("PROPFIND", "/", strings.NewReader(value)))
		if e == nil {
			_, _, _ = davRequested(root)
		}
	})
}

func TestWebDAVIfGrammar(t *testing.T) {
	for _, v := range []string{"()", "(", "(Not)", "<x>", "(<x>) <y> (<z>)", "([unterminated)", strings.Repeat("x", 8193)} {
		if _, e := davParseIf(v); e == nil {
			t.Fatalf("accepted malformed If %q", v)
		}
	}
	for _, v := range []string{"", "(<urn:uuid:sample>)", "(Not <DAV:no-lock>)", "<https://example.test/path> (<token> [\"etag\"]) (Not <other>)"} {
		if _, e := davParseIf(v); e != nil {
			t.Fatalf("rejected If %q: %v", v, e)
		}
	}
}

func TestWebDAVTimeoutBounds(t *testing.T) {
	for _, tc := range []struct {
		header string
		want   int
	}{
		{"", 3600},
		{"Infinite", 3600},
		{"Second-1", 1},
		{"Second-3599", 3599},
		{"Second-3600", 3600},
		{"Second-3601", 3600},
		{"Second-2147483647", 3600},
		{"Second-2147483648", 3600},
		{"Second-4294967295", 3600},
		{" Second-60 ", 60},
		{"Second-0, Second-60", 60},
		{"Second-4294967296, Second-60", 60},
	} {
		t.Run(tc.header, func(t *testing.T) {
			got, err := davTimeout(tc.header)
			if err != nil || got != tc.want {
				t.Fatalf("davTimeout(%q) = %d, %v; want %d", tc.header, got, err, tc.want)
			}
		})
	}
	for _, header := range []string{"Second-0", "Second--1", "Second-4294967296", "Second-18446744073709551615", "Second-1.5", "Second-", "second-1", "invalid"} {
		t.Run(header, func(t *testing.T) {
			if _, err := davTimeout(header); err != davError(400) {
				t.Fatalf("davTimeout(%q) error = %v; want HTTP 400", header, err)
			}
		})
	}
}

func FuzzWebDAVIf(f *testing.F) {
	for _, v := range []string{"", "(<token>)", "(Not <DAV:no-lock>)", "<https://example.test/path> ([\"etag\"])"} {
		f.Add(v)
	}
	f.Fuzz(func(t *testing.T, raw string) { _, _ = davParseIf(raw) })
}
