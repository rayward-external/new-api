package service

import (
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	GeoRouteDefault = "default"
	GeoRouteCA      = "ca"
	GeoRouteUSEast  = "us-east"
	GeoRouteUSSouth = "us-south"
)

var legacyRouteAlias = map[string]string{
	"geo-default": GeoRouteDefault, "geo-ca": GeoRouteCA,
	"geo-us-east": GeoRouteUSEast, "geo-us-south": GeoRouteUSSouth,
}

var geoUSSouthStates = map[string]struct{}{
	"AR": {},
	"CA": {},
	"KS": {},
	"LA": {},
	"MO": {},
	"NM": {},
	"OK": {},
	"TX": {},
}

func ResolveGeoRoute(country string, regionCode string) string {
	country = strings.ToUpper(strings.TrimSpace(country))
	regionCode = strings.ToUpper(strings.TrimSpace(regionCode))

	switch country {
	case "US":
		if _, ok := geoUSSouthStates[regionCode]; ok {
			return GeoRouteUSSouth
		}
		return GeoRouteUSEast
	case "CA":
		return GeoRouteCA
	default:
		return GeoRouteDefault
	}
}

func ResolveGeoRouteFromHeaders(headers http.Header) string {
	if r := normalizeGeoRoute(headers.Get("x-geo-route")); r != "" {
		return r
	}
	if raw := strings.TrimSpace(headers.Get("x-rayward-geo-bucket")); raw != "" {
		if r := legacyRouteAlias[strings.ToLower(raw)]; r != "" {
			return r
		}
	}
	// Validation-only: NEWAPI_GEO_REQUIRE_HEADER=1 disables the cf-ipcountry/region
	// fallback so the rollout gate can PROVE routing came from x-geo-route (or the
	// legacy header), not the equivalent cf-* fallback. Unset in production.
	if os.Getenv("NEWAPI_GEO_REQUIRE_HEADER") == "1" {
		return GeoRouteDefault
	}
	country := firstNonEmptyHeader(headers, "cf-ipcountry", "x-geo-country")
	region := firstNonEmptyHeader(headers, "cf-ipregioncode", "cf-region-code", "x-geo-state")
	return ResolveGeoRoute(country, region)
}

// CanonicalUpstreamBaseURL returns scheme://lowercase-host/ (single trailing
// slash, no path) so the value byte-matches LiteLLM's x-litellm-model-api-base.
func CanonicalUpstreamBaseURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return raw
	}
	scheme := parsed.Scheme
	if scheme == "" {
		scheme = "https"
	}
	return scheme + "://" + strings.ToLower(parsed.Host) + "/"
}

func ShouldApplyGeoRouteOverride(group string) bool {
	normalized := strings.ToLower(strings.TrimSpace(group))
	return normalized == "" || normalized == "default"
}

func SanitizeBaseURLHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if parsed.Host != "" {
		return parsed.Host
	}
	return raw
}

func normalizeGeoRoute(route string) string {
	switch strings.ToLower(strings.TrimSpace(route)) {
	case GeoRouteUSSouth:
		return GeoRouteUSSouth
	case GeoRouteUSEast:
		return GeoRouteUSEast
	case GeoRouteCA:
		return GeoRouteCA
	case GeoRouteDefault:
		return GeoRouteDefault
	default:
		return ""
	}
}

func firstNonEmptyHeader(headers http.Header, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(headers.Get(name)); value != "" {
			return value
		}
	}
	return ""
}
