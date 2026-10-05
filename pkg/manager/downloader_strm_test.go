package manager

import (
	"net/url"
	"strings"
	"testing"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/sirrobot01/decypharr/internal/config"
	debrid "github.com/sirrobot01/decypharr/pkg/debrid/common"
	"github.com/sirrobot01/decypharr/pkg/debrid/providers/torbox"
	"github.com/sirrobot01/decypharr/pkg/debrid/types"
	"github.com/sirrobot01/decypharr/pkg/storage"
	"google.golang.org/protobuf/proto"
)

func TestSTRMProviderIDsSurviveStorageAndSeasons(t *testing.T) {
	config.SetConfigPath(t.TempDir())
	token := "secret +&?=日本語"
	m := &Manager{config: &config.Config{}, clients: xsync.NewMap[string, debrid.Client]()}
	m.clients.Store("torbox", &torbox.Torbox{Host: "https://api.torbox.app/v1", APIKey: token})
	d := &Downloader{manager: m, strmURL: "http://localhost:8282"}
	entry := &storage.Entry{Action: config.DownloadActionStrmCDN, InfoHash: "hash", Name: "Show S01-S02", Files: map[string]*storage.File{}}
	remote := &types.Torrent{Id: "123", Debrid: "torbox", Files: map[string]types.File{}}
	names := []string{"Show S01E01 spaces.mkv", "日本語 S02E01 [!&?#].mkv"}
	ids := []string{"456", "91"}
	for i, name := range names {
		remote.Files[name] = types.File{TorrentId: "123", Id: ids[i], Name: name, Link: "torbox://123/" + ids[i]}
	}
	applyDebridTorrentToEntry(entry, remote)
	data, err := proto.Marshal(storage.EntryToProto(entry))
	if err != nil {
		t.Fatal(err)
	}
	var pb storage.EntryProto
	if err := proto.Unmarshal(data, &pb); err != nil {
		t.Fatal(err)
	}
	entry = storage.ProtoToEntry(&pb)
	seasons := []SeasonInfo{
		{SeasonNumber: 1, Name: "Show S01", Files: []*storage.File{entry.Files[names[0]]}},
		{SeasonNumber: 2, Name: "Show S02", Files: []*storage.File{entry.Files[names[1]]}},
	}
	for i, child := range convertToMultiSeason(entry, seasons) {
		child = storage.ProtoToEntry(storage.EntryToProto(child))
		raw, err := d.strmFileURL(child, child.Files[names[i]])
		if err != nil {
			t.Fatal(err)
		}
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal("invalid URL")
		}
		q := u.Query()
		if q.Get("torrent_id") != "123" || q.Get("file_id") != ids[i] || q.Get("redirect") != "true" || q.Get("token") != token {
			t.Fatal("incorrect query parameters (values omitted to protect token)")
		}
		if len(q) != 4 || u.Host != "api.torbox.app" || u.Path != "/v1/api/torrents/requestdl" {
			t.Fatal("incorrect endpoint")
		}
	}
}

func TestSTRMDefaultAndErrors(t *testing.T) {
	config.SetConfigPath(t.TempDir())
	m := &Manager{config: &config.Config{}, clients: xsync.NewMap[string, debrid.Client]()}
	d := &Downloader{manager: m, strmURL: "http://localhost:8282"}
	entry := &storage.Entry{Action: config.DownloadActionStrm, Name: "Release"}
	file := &storage.File{Name: "日本語 spaces [!].mkv"}
	{
		raw, err := d.strmFileURL(entry, file)
		if err != nil {
			t.Fatal(err)
		}
		u, err := url.Parse(raw)
		if err != nil || !strings.Contains(u.Path, "/webdav/stream/__all__/") || !strings.HasSuffix(u.Path, file.Name) {
			t.Fatal("default STRM changed")
		}
	}
	entry.Action = config.DownloadActionStrmCDN
	if _, err := d.strmFileURL(entry, file); err == nil {
		t.Fatal("unsupported provider accepted")
	}
	m.clients.Store("torbox", &torbox.Torbox{Host: "https://api.torbox.app/v1", APIKey: "secret"})
	entry.ActiveProvider = "torbox"
	if _, err := d.strmFileURL(entry, file); err == nil {
		t.Fatal("missing placement accepted")
	}
	entry.Providers = map[string]*storage.ProviderEntry{"torbox": {ID: "123", Files: map[string]*storage.ProviderFile{file.Name: {Id: ""}}}}
	if _, err := d.strmFileURL(entry, file); err == nil {
		t.Fatal("missing ID accepted")
	}
	entry.Action = "invalid"
	if _, err := d.strmFileURL(entry, file); err == nil {
		t.Fatal("invalid action accepted")
	}
}
