package cmd

import (
	"gopkg.in/h2non/gock.v1"
	"net/http"
	"os"
	"testing"
)

func Test_getGitHubUserID(t *testing.T) {
	_ = os.Setenv("GITHUB_TOKEN", "fake")
	gock.New("https://api.github.com").Get("/user").
		MatchHeader("Accept", "application/vnd.github.v3+json").
		MatchHeader("Authorization", "token fake").
		Reply(http.StatusOK).
		File("data/linuxsuren.json")

	tests := []struct {
		name   string
		wantId string
	}{{
		name:   "normal case",
		wantId: "LinuxSuRen",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotId := getGitHubUserID(); gotId != tt.wantId {
				t.Errorf("getGitHubUserID() = %v, want %v", gotId, tt.wantId)
			}
		})
	}
}
