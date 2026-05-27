package service

import (
	"net/http"
	"testing"
)

func TestResolveGeoBucket(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		country    string
		regionCode string
		want       string
	}{
		{name: "us south bucket", country: "US", regionCode: "TX", want: GeoBucketUSSouth},
		{name: "us default east bucket", country: "US", regionCode: "NY", want: GeoBucketUSEast},
		{name: "canada bucket", country: "CA", regionCode: "ON", want: GeoBucketCA},
		{name: "global default bucket", country: "SG", regionCode: "01", want: GeoBucketDefault},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ResolveGeoBucket(tt.country, tt.regionCode); got != tt.want {
				t.Fatalf("ResolveGeoBucket(%q, %q) = %q, want %q", tt.country, tt.regionCode, got, tt.want)
			}
		})
	}
}

func TestResolveGeoBucketFromHeaders(t *testing.T) {
	t.Parallel()

	headers := http.Header{}
	headers.Set("X-Rayward-Geo-Bucket", "geo-us-south")
	if got := ResolveGeoBucketFromHeaders(headers); got != GeoBucketUSSouth {
		t.Fatalf("trusted header override = %q, want %q", got, GeoBucketUSSouth)
	}

	headers = http.Header{}
	headers.Set("CF-IPCountry", "US")
	headers.Set("CF-IPRegionCode", "CA")
	if got := ResolveGeoBucketFromHeaders(headers); got != GeoBucketUSSouth {
		t.Fatalf("US CA region = %q, want %q", got, GeoBucketUSSouth)
	}

	headers = http.Header{}
	headers.Set("CF-IPCountry", "US")
	headers.Set("CF-Region-Code", "MA")
	if got := ResolveGeoBucketFromHeaders(headers); got != GeoBucketUSEast {
		t.Fatalf("US MA region = %q, want %q", got, GeoBucketUSEast)
	}
}

func TestShouldApplyGeoBucketOverride(t *testing.T) {
	t.Parallel()

	if !ShouldApplyGeoBucketOverride("default") {
		t.Fatal("default group should be geo-routed")
	}
	if !ShouldApplyGeoBucketOverride("") {
		t.Fatal("empty group should be geo-routed")
	}
	if ShouldApplyGeoBucketOverride("playground-east") {
		t.Fatal("explicit non-default group should not be geo-routed")
	}
}
