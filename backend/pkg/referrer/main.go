package referrer

import (
	"net/url"
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
