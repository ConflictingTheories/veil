package plugins

import (
	"database/sql"
	"io/ioutil"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

func TestQueuePublishJob_InsertsRow(t *testing.T) {
	tmp, err := ioutil.TempDir("", "plugins-db-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	dbPath := tmp + "/test.db"
	d, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	// create minimal publish_jobs table
	_, err = d.Exec(`CREATE TABLE publish_jobs (id TEXT PRIMARY KEY, node_id TEXT, version_id TEXT, channel_id TEXT, status TEXT, progress INTEGER, created_at INTEGER)`)
	if err != nil {
		t.Fatal(err)
	}

	SetDB(d)

	job := PublishJob{NodeID: "node1", ChannelID: "chan1"}
	j, err := QueuePublishJob(job)
	if err != nil {
		t.Fatalf("QueuePublishJob failed: %v", err)
	}
	// verify row exists
	var id string
	row := d.QueryRow(`SELECT id FROM publish_jobs WHERE id = ?`, j.ID)
	if err := row.Scan(&id); err != nil {
		t.Fatalf("expected row inserted: %v", err)
	}
}
