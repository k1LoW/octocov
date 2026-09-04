package datastore

import (
	"strings"
	"testing"
)

func TestCredentialsFromJSON(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr string
	}{
		{
			"service_account",
			`{"type":"service_account","project_id":"p","private_key_id":"k","private_key":"dummy","client_email":"sa@p.iam.gserviceaccount.com","client_id":"1","token_uri":"https://oauth2.googleapis.com/token"}`,
			"",
		},
		{
			"authorized_user",
			`{"type":"authorized_user","client_id":"c","client_secret":"s","refresh_token":"r"}`,
			"",
		},
		{
			"missing type",
			`{"client_id":"c","client_secret":"s","refresh_token":"r"}`,
			"unsupported unidentified file type",
		},
		{
			"unknown type",
			`{"type":"unknown"}`,
			"unsupported filetype",
		},
		{
			"invalid json",
			`not json`,
			"invalid character",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds, err := credentialsFromJSON([]byte(tt.in), []string{"https://www.googleapis.com/auth/cloud-platform"})
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("want error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("want error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if creds == nil {
				t.Error("want credentials, got nil")
			}
		})
	}
}
