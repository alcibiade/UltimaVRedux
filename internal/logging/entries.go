package logging

import "time"

const (
	defaultEntryCapacity = 2048
	maxEntryMessageBytes = 4096
)

// Entry is one structured categorized log record retained for diagnostics.
type Entry struct {
	Sequence  uint64    `json:"sequence"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Category  string    `json:"category"`
	Message   string    `json:"message"`
}

// EntryBatch is an independent cursor snapshot from the bounded log buffer.
type EntryBatch struct {
	Entries        []Entry `json:"entries"`
	NextCursor     uint64  `json:"nextCursor"`
	OldestSequence uint64  `json:"oldestSequence"`
	NewestSequence uint64  `json:"newestSequence"`
	Dropped        uint64  `json:"dropped"`
	HasMore        bool    `json:"hasMore"`
}

func (l *Logger) appendEntryLocked(timestamp time.Time, level LogLevel, category, message string) {
	l.nextSequence++
	entry := Entry{
		Sequence: l.nextSequence, Timestamp: timestamp, Level: level.String(),
		Category: category, Message: boundedMessage(message),
	}
	if l.entryCount < len(l.entries) {
		index := (l.entryStart + l.entryCount) % len(l.entries)
		l.entries[index] = entry
		l.entryCount++

		return
	}
	l.entries[l.entryStart] = entry
	l.entryStart = (l.entryStart + 1) % len(l.entries)
}

// EntriesAfter returns copied records newer than cursor without advancing other readers.
func (l *Logger) EntriesAfter(cursor uint64, limit int) EntryBatch {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if limit <= 0 {
		limit = 1
	}
	batch := EntryBatch{
		Entries: make([]Entry, 0, min(limit, l.entryCount)), NextCursor: cursor,
		OldestSequence: 0, NewestSequence: 0, Dropped: 0, HasMore: false,
	}
	if l.entryCount == 0 {
		return batch
	}
	oldest := l.entries[l.entryStart].Sequence
	newest := l.nextSequence
	batch.OldestSequence, batch.NewestSequence = oldest, newest
	if cursor < oldest-1 {
		batch.Dropped = oldest - 1 - cursor
		cursor = oldest - 1
	}
	for offset := 0; offset < l.entryCount && len(batch.Entries) < limit; offset++ {
		entry := l.entries[(l.entryStart+offset)%len(l.entries)]
		if entry.Sequence <= cursor {
			continue
		}
		batch.Entries = append(batch.Entries, entry)
		batch.NextCursor = entry.Sequence
	}
	batch.HasMore = batch.NextCursor < newest

	return batch
}

func boundedMessage(message string) string {
	if len(message) <= maxEntryMessageBytes {
		return message
	}
	end := maxEntryMessageBytes
	for end > 0 && message[end]&0xc0 == 0x80 {
		end--
	}

	return string([]byte(message[:end]))
}
