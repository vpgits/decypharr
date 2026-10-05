package torbox

import (
	"net/url"
	"strings"
	"testing"

	"github.com/sirrobot01/decypharr/pkg/debrid/account"
	"github.com/sirrobot01/decypharr/pkg/debrid/types"
)

func TestSTRMPermalinkReusesDownloadEndpoint(t *testing.T) {
	token := "secret +&?=日本語"
	tb := &Torbox{Host: "https://api.torbox.app/v1", APIKey: token}
	raw, err := tb.STRMPermalink("123", "456")
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal("invalid URL")
	}
	if u.Query().Get("token") != token || u.Query().Get("torrent_id") != "123" || u.Query().Get("file_id") != "456" || u.Query().Get("redirect") != "true" {
		t.Fatal("incorrect query")
	}
	dl, err := tb.fetchDownloadLink(&account.Account{Token: token}, "123", &types.File{Id: "456", Name: "日本語 spaces !.mkv"})
	if err != nil || dl.DownloadLink != raw {
		t.Fatal("different requestdl builders")
	}
	for _, ids := range [][2]string{{"", "456"}, {"123", ""}} {
		_, err := tb.STRMPermalink(ids[0], ids[1])
		if err == nil || strings.Contains(err.Error(), token) {
			t.Fatal("unsafe or absent validation error")
		}
	}
	tb.APIKey = ""
	if _, err := tb.STRMPermalink("123", "456"); err == nil {
		t.Fatal("missing token accepted")
	}
}
