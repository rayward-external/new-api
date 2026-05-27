package service

import (
	"net/http"
	"net/url"
	"strings"
)

const (
	GeoBucketDefault = "geo-default"
	GeoBucketCA      = "geo-ca"
	GeoBucketUSEast  = "geo-us-east"
	GeoBucketUSSouth = "geo-us-south"
)

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

func ResolveGeoBucket(country string, regionCode string) string {
	country = strings.ToUpper(strings.TrimSpace(country))
	regionCode = strings.ToUpper(strings.TrimSpace(regionCode))

	switch country {
	case "US":
		if _, ok := geoUSSouthStates[regionCode]; ok {
			return GeoBucketUSSouth
		}
		return GeoBucketUSEast
	case "CA":
		return GeoBucketCA
	default:
		return GeoBucketDefault
	}
}

func ResolveGeoBucketFromHeaders(headers http.Header) string {
	if bucket := normalizeGeoBucket(headers.Get("x-rayward-geo-bucket")); bucket != "" {
		return bucket
	}

	country := firstNonEmptyHeader(headers, "cf-ipcountry", "x-geo-country")
	regionCode := firstNonEmptyHeader(headers, "cf-ipregioncode", "cf-region-code", "x-geo-state")
	return ResolveGeoBucket(country, regionCode)
}

func ShouldApplyGeoBucketOverride(group string) bool {
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

func normalizeGeoBucket(bucket string) string {
	switch strings.ToLower(strings.TrimSpace(bucket)) {
	case GeoBucketDefault:
		return GeoBucketDefault
	case GeoBucketCA:
		return GeoBucketCA
	case GeoBucketUSEast:
		return GeoBucketUSEast
	case GeoBucketUSSouth:
		return GeoBucketUSSouth
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
