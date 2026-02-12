package referrer

import (
	"net/url"
	"regexp"
	"strings"
)

// hrefHasParam checks if the href URL's query string contains a parameter with the given value.
// If value is empty, it only checks for parameter existence.
func hrefHasParam(href *url.URL, key, value string) bool {
	q := href.Query()
	if value == "" {
		return q.Has(key)
	}
	return q.Get(key) == value
}

func hostContains(u *url.URL, sub string) bool {
	return strings.Contains(strings.ToLower(u.Host), sub)
}

// --- Ad matchers ---

func matchAdBing(href, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "bing.com") && hrefHasParam(href, "utm_medium", "cpc") {
		return &ReferrerInfo{Type: "ad", Network: "bing"}
	}
	return nil
}

func matchAdGoogle(href, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "google") &&
		(hrefHasParam(href, "utm_medium", "cpc") || hrefHasParam(href, "gclid", "")) {
		return &ReferrerInfo{Type: "ad", Network: "google"}
	}
	return nil
}

func matchAdYahoo(href, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "search.yahoo") && hrefHasParam(href, "utm_medium", "cpc") {
		return &ReferrerInfo{Type: "ad", Network: "yahoo"}
	}
	return nil
}

func matchAdFacebook(href, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "facebook") {
		if hrefHasParam(href, "utm_medium", "cpc") || hrefHasParam(href, "fbclid", "") {
			return &ReferrerInfo{Type: "ad", Network: "facebook"}
		}
	}
	return nil
}

func matchAdOthers(href, ref *url.URL, _ string) *ReferrerInfo {
	if ref.Host != "" && hrefHasParam(href, "utm_medium", "cpc") {
		return &ReferrerInfo{Type: "ad", Network: "unknown"}
	}
	return nil
}

// --- Email matchers ---

func matchEmailGmail(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "mail.google.com") {
		return &ReferrerInfo{Type: "email", Network: "gmail"}
	}
	return nil
}

func matchEmailHotmail(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, ".live.com") {
		return &ReferrerInfo{Type: "email", Network: "hotmail"}
	}
	return nil
}

func matchEmailYahoo(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "mail.yahoo") {
		return &ReferrerInfo{Type: "email", Network: "yahoo"}
	}
	return nil
}

func matchEmailNaver(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "mail.naver.com") {
		return &ReferrerInfo{Type: "email", Network: "naver"}
	}
	return nil
}

var daumMailRegex = regexp.MustCompile(`mail[0-9]*\.daum\.net`)

func matchEmailDaum(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if daumMailRegex.MatchString(strings.ToLower(ref.Host)) {
		return &ReferrerInfo{Type: "email", Network: "daum"}
	}
	return nil
}

func matchEmailOthers(href *url.URL, _ *url.URL, _ string) *ReferrerInfo {
	q := href.Query()
	if q.Has("civimail_x_q") {
		return &ReferrerInfo{Type: "email", Network: "civimail"}
	}
	if q.Get("utm_medium") == "email" {
		return &ReferrerInfo{Type: "email", Network: "utm_medium"}
	}
	return nil
}

// --- Social matchers ---

func matchSocialFacebook(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "facebook.com") {
		return &ReferrerInfo{Type: "social", Network: "facebook"}
	}
	return nil
}

var lineUARegex = regexp.MustCompile(`\WLine/\d`)

func matchSocialLine(_ *url.URL, _ *url.URL, userAgent string) *ReferrerInfo {
	if lineUARegex.MatchString(userAgent) {
		return &ReferrerInfo{Type: "social", Network: "line"}
	}
	return nil
}

func matchSocialHangouts(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "hangouts.google.com") {
		return &ReferrerInfo{Type: "social", Network: "google hangout"}
	}
	return nil
}

func matchSocialYoutube(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "youtube.com") {
		return &ReferrerInfo{Type: "social", Network: "youtube"}
	}
	return nil
}

func matchSocialLinkedin(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "linkedin.com") {
		return &ReferrerInfo{Type: "social", Network: "linkedin"}
	}
	return nil
}

func matchSocialPinterest(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "pinterest.com") {
		return &ReferrerInfo{Type: "social", Network: "pinterest"}
	}
	return nil
}

func matchSocialTwitter(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "twitter.com") || strings.ToLower(ref.Host) == "t.co" {
		return &ReferrerInfo{Type: "social", Network: "twitter"}
	}
	return nil
}

func matchSocialMe2day(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "me2day.net") {
		return &ReferrerInfo{Type: "social", Network: "me2day"}
	}
	return nil
}

// --- Search matchers ---

func matchSearchAol(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "search.aol.com") {
		return &ReferrerInfo{Type: "search", Network: "aol"}
	}
	return nil
}

func matchSearchBaidu(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "baidu.com") {
		return &ReferrerInfo{Type: "search", Network: "baidu"}
	}
	return nil
}

func matchSearchGoogle(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "google") && ref.String() != "" {
		return &ReferrerInfo{Type: "search", Network: "google"}
	}
	return nil
}

func matchSearchYahoo(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "search.yahoo") {
		return &ReferrerInfo{Type: "search", Network: "yahoo"}
	}
	return nil
}

func matchSearchBing(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "bing.com") {
		return &ReferrerInfo{Type: "search", Network: "bing"}
	}
	return nil
}

func matchSearchYandex(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "yandex.com") {
		return &ReferrerInfo{Type: "search", Network: "yandex"}
	}
	return nil
}

func matchSearchNaver(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "search.naver.com") {
		return &ReferrerInfo{Type: "search", Network: "naver"}
	}
	return nil
}

func matchSearchDaum(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "search.daum.net") {
		return &ReferrerInfo{Type: "search", Network: "daum"}
	}
	return nil
}

func matchSearchNate(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if hostContains(ref, "search.nate.com") {
		return &ReferrerInfo{Type: "search", Network: "nate"}
	}
	return nil
}

// --- Internal, Link, Direct, Unknown matchers ---

func matchInternal(href *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if ref.Host != "" && strings.EqualFold(href.Host, ref.Host) {
		return &ReferrerInfo{Type: "internal"}
	}
	return nil
}

func matchLink(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if ref.Host != "" && ref.Scheme != "about" {
		return &ReferrerInfo{Type: "link", Link: ref.String()}
	}
	return nil
}

func matchDirect(_ *url.URL, ref *url.URL, _ string) *ReferrerInfo {
	if ref.Host == "" || ref.Scheme == "about" {
		return &ReferrerInfo{Type: "direct"}
	}
	return nil
}

func matchUnknown(_ *url.URL, _ *url.URL, _ string) *ReferrerInfo {
	return &ReferrerInfo{Type: "unknown"}
}
