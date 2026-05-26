package vertex

import "testing"

func TestIsADCKey(t *testing.T) {
	tests := map[string]bool{
		"adc":           true,
		" google_adc ":  true,
		"cloud_run_adc": true,
		"metadata":      true,
		"{}":            false,
		"":              false,
	}

	for input, want := range tests {
		if got := IsADCKey(input); got != want {
			t.Fatalf("IsADCKey(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestGetADCProjectIDUsesEnvironment(t *testing.T) {
	t.Setenv("GOOGLE_CLOUD_PROJECT", "project-from-google-cloud-project")
	t.Setenv("GCP_PROJECT_ID", "project-from-gcp-project-id")
	t.Setenv("GCLOUD_PROJECT", "project-from-gcloud-project")

	got, err := GetADCProjectID()
	if err != nil {
		t.Fatalf("GetADCProjectID() error = %v", err)
	}
	if got != "project-from-google-cloud-project" {
		t.Fatalf("GetADCProjectID() = %q, want GOOGLE_CLOUD_PROJECT value", got)
	}
}
