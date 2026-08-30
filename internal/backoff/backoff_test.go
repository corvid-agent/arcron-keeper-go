package backoff

import (
	"path/filepath"
	"testing"
)

func TestWaitSchedule(t *testing.T) {
	cases := []struct {
		fail, interval, want uint64
	}{
		{1, 10, 10},
		{2, 10, 20},
		{3, 10, 40},
		{4, 10, 80},
		{5, 10, 80},
		{1, 2000, 1286},
		{1, 0, 1286},
	}
	for _, c := range cases {
		got := WaitRounds(c.fail, c.interval)
		if got != c.want {
			t.Fatalf("fail=%d interval=%d got=%d want=%d", c.fail, c.interval, got, c.want)
		}
	}
}

func TestPersistAcrossOpen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backoff.json")
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Fail(19, 100, 10); err != nil {
		t.Fatal(err)
	}
	if !f.Blocked(19, 109) || f.Blocked(19, 110) {
		t.Fatal("blocked window")
	}
	f2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if !f2.Blocked(19, 109) {
		t.Fatal("did not persist")
	}
	if err := f2.Success(19); err != nil {
		t.Fatal(err)
	}
	if f2.Blocked(19, 109) {
		t.Fatal("success should clear")
	}
}
