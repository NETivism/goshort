package referrer

import (
	"testing"
)

func TestSocialFacebook(t *testing.T) {
	info := Parse("https://example.com/abc", "https://www.facebook.com/some/page", "")
	if info.Type != "social" || info.Network != "facebook" {
		t.Errorf("expected social/facebook, got %s/%s", info.Type, info.Network)
	}
}

func TestSocialTwitter(t *testing.T) {
	info := Parse("https://example.com/abc", "https://twitter.com/user/status/123", "")
	if info.Type != "social" || info.Network != "twitter" {
		t.Errorf("expected social/twitter, got %s/%s", info.Type, info.Network)
	}
}

func TestSocialTwitterTCo(t *testing.T) {
	info := Parse("https://example.com/abc", "https://t.co/abcdef", "")
	if info.Type != "social" || info.Network != "twitter" {
		t.Errorf("expected social/twitter, got %s/%s", info.Type, info.Network)
	}
}

func TestSocialYoutube(t *testing.T) {
	info := Parse("https://example.com/abc", "https://www.youtube.com/watch?v=123", "")
	if info.Type != "social" || info.Network != "youtube" {
		t.Errorf("expected social/youtube, got %s/%s", info.Type, info.Network)
	}
}

func TestSocialLinkedin(t *testing.T) {
	info := Parse("https://example.com/abc", "https://www.linkedin.com/feed", "")
	if info.Type != "social" || info.Network != "linkedin" {
		t.Errorf("expected social/linkedin, got %s/%s", info.Type, info.Network)
	}
}

func TestSocialLine(t *testing.T) {
	info := Parse("https://example.com/abc", "", "Mozilla/5.0 Line/8.0")
	if info.Type != "social" || info.Network != "line" {
		t.Errorf("expected social/line, got %s/%s", info.Type, info.Network)
	}
}

func TestSearchGoogle(t *testing.T) {
	info := Parse("https://example.com/abc", "https://www.google.com/search?q=test", "")
	if info.Type != "search" || info.Network != "google" {
		t.Errorf("expected search/google, got %s/%s", info.Type, info.Network)
	}
}

func TestSearchBing(t *testing.T) {
	info := Parse("https://example.com/abc", "https://www.bing.com/search?q=test", "")
	if info.Type != "search" || info.Network != "bing" {
		t.Errorf("expected search/bing, got %s/%s", info.Type, info.Network)
	}
}

func TestSearchYahoo(t *testing.T) {
	info := Parse("https://example.com/abc", "https://search.yahoo.com/search?p=test", "")
	if info.Type != "search" || info.Network != "yahoo" {
		t.Errorf("expected search/yahoo, got %s/%s", info.Type, info.Network)
	}
}

func TestSearchBaidu(t *testing.T) {
	info := Parse("https://example.com/abc", "https://www.baidu.com/s?wd=test", "")
	if info.Type != "search" || info.Network != "baidu" {
		t.Errorf("expected search/baidu, got %s/%s", info.Type, info.Network)
	}
}

func TestAdGoogleGclid(t *testing.T) {
	info := Parse("https://example.com/abc?gclid=abc123", "https://www.google.com/", "")
	if info.Type != "ad" || info.Network != "google" {
		t.Errorf("expected ad/google, got %s/%s", info.Type, info.Network)
	}
}

func TestAdGoogleUtmCpc(t *testing.T) {
	info := Parse("https://example.com/abc?utm_medium=cpc", "https://www.google.com/", "")
	if info.Type != "ad" || info.Network != "google" {
		t.Errorf("expected ad/google, got %s/%s", info.Type, info.Network)
	}
}

func TestAdFacebookFbclid(t *testing.T) {
	info := Parse("https://example.com/abc?fbclid=abc123", "https://www.facebook.com/", "")
	if info.Type != "ad" || info.Network != "facebook" {
		t.Errorf("expected ad/facebook, got %s/%s", info.Type, info.Network)
	}
}

func TestEmailGmail(t *testing.T) {
	info := Parse("https://example.com/abc", "https://mail.google.com/mail/u/0/", "")
	if info.Type != "email" || info.Network != "gmail" {
		t.Errorf("expected email/gmail, got %s/%s", info.Type, info.Network)
	}
}

func TestEmailHotmail(t *testing.T) {
	info := Parse("https://example.com/abc", "https://outlook.live.com/mail/inbox", "")
	if info.Type != "email" || info.Network != "hotmail" {
		t.Errorf("expected email/hotmail, got %s/%s", info.Type, info.Network)
	}
}

func TestDirectEmpty(t *testing.T) {
	info := Parse("https://example.com/abc", "", "")
	if info.Type != "direct" {
		t.Errorf("expected direct, got %s", info.Type)
	}
}

func TestInternal(t *testing.T) {
	info := Parse("https://example.com/abc", "https://example.com/other", "")
	if info.Type != "internal" {
		t.Errorf("expected internal, got %s", info.Type)
	}
}

func TestExternalLink(t *testing.T) {
	info := Parse("https://example.com/abc", "https://somesite.org/page", "")
	if info.Type != "link" {
		t.Errorf("expected link, got %s", info.Type)
	}
	if info.Link != "https://somesite.org/page" {
		t.Errorf("expected link URL, got %s", info.Link)
	}
}

func TestParseUTMFull(t *testing.T) {
	utm := ParseUTM("https://example.com/?utm_source=google&utm_medium=cpc&utm_campaign=spring&utm_term=shoes&utm_content=ad1")
	if utm.Source != "google" {
		t.Errorf("expected source=google, got %s", utm.Source)
	}
	if utm.Medium != "cpc" {
		t.Errorf("expected medium=cpc, got %s", utm.Medium)
	}
	if utm.Campaign != "spring" {
		t.Errorf("expected campaign=spring, got %s", utm.Campaign)
	}
	if utm.Term != "shoes" {
		t.Errorf("expected term=shoes, got %s", utm.Term)
	}
	if utm.Content != "ad1" {
		t.Errorf("expected content=ad1, got %s", utm.Content)
	}
}

func TestParseUTMPartial(t *testing.T) {
	utm := ParseUTM("https://example.com/?utm_source=newsletter")
	if utm.Source != "newsletter" {
		t.Errorf("expected source=newsletter, got %s", utm.Source)
	}
	if utm.Medium != "" {
		t.Errorf("expected empty medium, got %s", utm.Medium)
	}
}

func TestParseUTMNone(t *testing.T) {
	utm := ParseUTM("https://example.com/page")
	if utm.Source != "" || utm.Medium != "" || utm.Campaign != "" {
		t.Errorf("expected empty UTM, got %+v", utm)
	}
}
