package referrer

import (
	"net/url"

	"github.com/netivism/goshort/backend/pkg/model"
)

type ReferrerInfo struct {
	Type    string
	Network string
	Link    string
}

// Parse runs matchers in priority order against the current URL, referrer URL, and user agent.
func Parse(currentURL, referrerURL, userAgent string) *ReferrerInfo {
	parsedHref, _ := url.Parse(currentURL)
	parsedReferrer, _ := url.Parse(referrerURL)
	if parsedHref == nil {
		parsedHref = &url.URL{}
	}
	if parsedReferrer == nil {
		parsedReferrer = &url.URL{}
	}

	matchers := []func(*url.URL, *url.URL, string) *ReferrerInfo{
		matchAdBing,
		matchAdGoogle,
		matchAdYahoo,
		matchAdFacebook,
		matchAdOthers,
		matchEmailGmail,
		matchEmailHotmail,
		matchEmailYahoo,
		matchEmailNaver,
		matchEmailDaum,
		matchEmailOthers,
		matchSocialFacebook,
		matchSocialLine,
		matchSocialHangouts,
		matchSocialYoutube,
		matchSocialLinkedin,
		matchSocialPinterest,
		matchSocialTwitter,
		matchSocialMe2day,
		matchSearchAol,
		matchSearchBaidu,
		matchSearchGoogle,
		matchSearchYahoo,
		matchSearchBing,
		matchSearchYandex,
		matchSearchNaver,
		matchSearchDaum,
		matchSearchNate,
		matchInternal,
		matchLink,
		matchDirect,
		matchUnknown,
	}

	for _, m := range matchers {
		if info := m(parsedHref, parsedReferrer, userAgent); info != nil {
			return info
		}
	}

	return &ReferrerInfo{Type: "unknown"}
}

// ParseUTM extracts utm_* query parameters from the given URL.
func ParseUTM(rawURL string) model.Utm {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return model.Utm{}
	}
	q := parsed.Query()
	return model.Utm{
		Source:   q.Get("utm_source"),
		Medium:   q.Get("utm_medium"),
		Term:     q.Get("utm_term"),
		Content:  q.Get("utm_content"),
		Campaign: q.Get("utm_campaign"),
	}
}
