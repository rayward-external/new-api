package service

import (
	"net/http"
	"strings"
	"testing"
)

func TestResolveGeoRoute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		country    string
		regionCode string
		want       string
	}{
		{name: "us south route", country: "US", regionCode: "TX", want: GeoRouteUSSouth},
		{name: "us default east route", country: "US", regionCode: "NY", want: GeoRouteUSEast},
		{name: "canada route", country: "CA", regionCode: "ON", want: GeoRouteCA},
		{name: "global default route", country: "SG", regionCode: "01", want: GeoRouteDefault},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ResolveGeoRoute(tt.country, tt.regionCode); got != tt.want {
				t.Fatalf("ResolveGeoRoute(%q, %q) = %q, want %q", tt.country, tt.regionCode, got, tt.want)
			}
		})
	}
}

func TestResolveGeoRouteFromHeaders(t *testing.T) {
	t.Parallel()

	headers := http.Header{}
	headers.Set("x-geo-route", "us-south")
	if got := ResolveGeoRouteFromHeaders(headers); got != GeoRouteUSSouth {
		t.Fatalf("x-geo-route override = %q, want %q", got, GeoRouteUSSouth)
	}

	headers = http.Header{}
	headers.Set("x-rayward-geo-bucket", "geo-us-south")
	if got := ResolveGeoRouteFromHeaders(headers); got != GeoRouteUSSouth {
		t.Fatalf("legacy header alias = %q, want %q", got, GeoRouteUSSouth)
	}

	headers = http.Header{}
	headers.Set("CF-IPCountry", "US")
	headers.Set("CF-IPRegionCode", "CA")
	if got := ResolveGeoRouteFromHeaders(headers); got != GeoRouteUSSouth {
		t.Fatalf("US CA region = %q, want %q", got, GeoRouteUSSouth)
	}

	headers = http.Header{}
	headers.Set("CF-IPCountry", "US")
	headers.Set("CF-Region-Code", "MA")
	if got := ResolveGeoRouteFromHeaders(headers); got != GeoRouteUSEast {
		t.Fatalf("US MA region = %q, want %q", got, GeoRouteUSEast)
	}
}

func TestResolveGeoRouteFromHeadersRequireHeader(t *testing.T) {
	t.Setenv("NEWAPI_GEO_REQUIRE_HEADER", "1")

	headers := http.Header{}
	headers.Set("CF-IPCountry", "US")
	headers.Set("CF-IPRegionCode", "CA")
	if got := ResolveGeoRouteFromHeaders(headers); got != GeoRouteDefault {
		t.Fatalf("require-header fallback disabled = %q, want %q", got, GeoRouteDefault)
	}

	headers.Set("x-geo-route", "us-south")
	if got := ResolveGeoRouteFromHeaders(headers); got != GeoRouteUSSouth {
		t.Fatalf("require-header with x-geo-route = %q, want %q", got, GeoRouteUSSouth)
	}
}

func TestCanonicalUpstreamBaseURL(t *testing.T) {
	t.Parallel()

	// Reference hosts (lowercase, canonical). Each case feeds a variant that must
	// canonicalize back to "https://<host>/" so the expectation is derived, never
	// a hand-typed literal that could be mistyped.
	const eastHost = "lawgic-openclaw.openai.azure.com"
	const scusHost = "lawgic-openclaw-sc.openai.azure.com"

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "uppercase host with path", raw: "https://" + strings.ToUpper(eastHost) + "/openai/", want: "https://" + eastHost + "/"},
		{name: "host only", raw: "https://" + scusHost, want: "https://" + scusHost + "/"},
		{name: "missing scheme defaults https", raw: "//" + eastHost + "/", want: "https://" + eastHost + "/"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := CanonicalUpstreamBaseURL(tt.raw); got != tt.want {
				t.Fatalf("CanonicalUpstreamBaseURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestShouldApplyGeoRouteOverride(t *testing.T) {
	t.Parallel()

	if !ShouldApplyGeoRouteOverride("default") {
		t.Fatal("default group should be geo-routed")
	}
	if !ShouldApplyGeoRouteOverride("") {
		t.Fatal("empty group should be geo-routed")
	}
	if ShouldApplyGeoRouteOverride("playground-east") {
		t.Fatal("explicit non-default group should not be geo-routed")
	}
}
