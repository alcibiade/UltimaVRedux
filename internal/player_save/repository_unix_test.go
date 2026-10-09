//go:build aix || android || darwin || dragonfly || freebsd || illumos || ios || linux || netbsd || openbsd || solaris

package player_save

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestRepositoryListRejectsFIFOWithoutWriter(t *testing.T) {
	repository := newTestRepository(t, t.TempDir(), nil)
	if _, err := repository.Save(Slot02, testEnvelope(testTime(1)), ""); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(repository.path(Slot01), 0o600); err != nil {
		t.Fatal(err)
	}
	result := make(chan []SlotSummary, 1)
	go func() { result <- repository.List() }()
	select {
	case slots := <-result:
		if slots[0].Status != SlotUnavailable || slots[1].Status != SlotReady {
			t.Fatalf("FIFO/healthy status = %s/%s", slots[0].Status, slots[1].Status)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("listing blocked on a FIFO without a writer")
	}
	require.Empty(t, repository.List()[0].Revision)
	record, err := repository.Load(Slot01)
	require.Error(t, err)
	require.Empty(t, record)
	for _, revision := range []string{"", "stale"} {
		_, err = repository.Save(Slot01, testEnvelope(testTime(1)), revision)
		require.Error(t, err)
		require.Error(t, repository.Delete(Slot01, revision))
	}
}
