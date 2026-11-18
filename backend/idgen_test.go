package main

import (
	"testing"
	"time"
)

func splitSnowflake(id int64) (timestampMS, workerID, sequence int64) {
	return id >> 22, (id >> 12) & 0x3FF, id & 0xFFF
}

func TestGeneratorEncodesSnowflake(t *testing.T) {
	current := int64(63158400123) // milisegundos desde 2024-01-01
	generator := NewGenerator(7)
	generator.now = func() int64 { return current }
	generator.sleep = func(time.Duration) {}

	id, err := generator.NextID()
	if err != nil {
		t.Fatal(err)
	}
	if want := uint64(264905530269528064); uint64(id) != want {
		t.Errorf("id = %d, esperaba %d", id, want)
	}
	if key := encodeBase62(uint64(id)); key != "jzgKpHNygg" {
		t.Errorf("key = %q, esperaba %q", key, "jzgKpHNygg")
	}
	timestampMS, workerID, sequence := splitSnowflake(id)
	if timestampMS != current || workerID != 7 || sequence != 0 {
		t.Errorf("campos = (%d, %d, %d), esperaba (%d, 7, 0)", timestampMS, workerID, sequence, current)
	}
	if got := id >> 22; got > 1<<41-1 {
		t.Errorf("el timestamp ocupa más de 41 bits: %d", got)
	}
}

func TestGeneratorResetsSequenceEachMillisecond(t *testing.T) {
	current := int64(1000)
	generator := NewGenerator(0)
	generator.now = func() int64 { return current }
	generator.sleep = func(time.Duration) {}

	var sequences []int64
	for i, next := range []int64{1000, 1000, 1001, 1001, 1001} {
		current = next
		id, err := generator.NextID()
		if err != nil {
			t.Fatal(err)
		}
		_, _, sequence := splitSnowflake(id)
		sequences = append(sequences, sequence)
		if i > 0 && sequence != 0 && sequences[i-1] == sequence {
			t.Errorf("secuencia no se reinició en el ms %d: %v", next, sequences)
		}
	}

	want := []int64{0, 1, 0, 1, 2}
	for i := range want {
		if sequences[i] != want[i] {
			t.Fatalf("secuencias = %v, esperaba %v", sequences, want)
		}
	}
}

func TestGeneratorWaitsForNextMillisecondOnSequenceOverflow(t *testing.T) {
	current := int64(1000)
	generator := NewGenerator(0)
	generator.now = func() int64 { return current }
	generator.sleep = func(d time.Duration) { current += int64(d / time.Millisecond) }

	const total = 4097 // 4096 valores de secuencia (0..4095) + uno que desborda
	ids := make([]int64, total)
	for i := range ids {
		id, err := generator.NextID()
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = id
	}

	for i := 1; i < len(ids); i++ {
		if ids[i] <= ids[i-1] {
			t.Fatalf("ids no crecientes en %d: %d, %d", i, ids[i-1], ids[i])
		}
	}
	timestampMS, _, sequence := splitSnowflake(ids[total-1])
	if timestampMS != 1001 || sequence != 0 {
		t.Errorf("último id: timestamp=%d secuencia=%d, esperaba 1001 y 0", timestampMS, sequence)
	}
}

func TestGeneratorWaitsWhenClockMovesBackwards(t *testing.T) {
	current := int64(5000)
	generator := NewGenerator(3)
	generator.now = func() int64 { return current }
	generator.sleep = func(d time.Duration) { current += int64(d / time.Millisecond) }

	first, err := generator.NextID()
	if err != nil {
		t.Fatal(err)
	}
	current = 4998 // el reloj retrocede 2 ms
	second, err := generator.NextID()
	if err != nil {
		t.Fatal(err)
	}
	if second <= first {
		t.Fatalf("id duplicado o decreciente: %d, %d", first, second)
	}
	ts1, worker1, seq1 := splitSnowflake(first)
	ts2, worker2, seq2 := splitSnowflake(second)
	if ts1 != 5000 || ts2 != 5000 || seq1 != 0 || seq2 != 1 || worker1 != 3 || worker2 != 3 {
		t.Errorf("estado = (%d,%d,%d) → (%d,%d,%d)", ts1, worker1, seq1, ts2, worker2, seq2)
	}
}

func TestGeneratorKeepsWaitingWhenClockRemainsBehind(t *testing.T) {
	current := int64(5000)
	generator := NewGenerator(3)
	generator.now = func() int64 { return current }
	waits := 0
	generator.sleep = func(d time.Duration) {
		waits++
		if waits > 1 {
			current += int64(d / time.Millisecond)
		}
	}

	first, err := generator.NextID()
	if err != nil {
		t.Fatal(err)
	}

	current = 4998 // el reloj retrocede y no se recupera en el primer sleep
	second, err := generator.NextID()
	if err != nil {
		t.Fatalf("el generador desistió antes de recuperar el reloj: %v", err)
	}
	if second <= first {
		t.Fatalf("id duplicado o decreciente: %d, %d", first, second)
	}
	if waits < 2 {
		t.Errorf("esperó %d veces, esperaba al menos 2", waits)
	}
}

func TestGeneratorRejectsOutOfRangeTimestampWithoutCorruptingState(t *testing.T) {
	current := int64(1000)
	generator := NewGenerator(0)
	generator.now = func() int64 { return current }
	generator.sleep = func(d time.Duration) { current += int64(d / time.Millisecond) }

	current = maxTimestamp + 1
	if _, err := generator.NextID(); err == nil {
		t.Fatal("esperaba error por timestamp de 42 bits")
	}

	current = 1000
	id, err := generator.NextID()
	if err != nil {
		t.Fatalf("el generador quedó en estado inválido: %v", err)
	}
	timestampMS, _, sequence := splitSnowflake(id)
	if timestampMS != 1000 || sequence != 0 {
		t.Errorf("estado = (%d, %d), esperaba (1000, 0)", timestampMS, sequence)
	}
}

func TestGeneratorRejectsTimestampOverflowAfterSequenceWait(t *testing.T) {
	current := maxTimestamp - 1
	generator := NewGenerator(0)
	generator.now = func() int64 { return current }
	generator.sleep = func(d time.Duration) { current += 2 * int64(d/time.Millisecond) }

	for i := 0; i <= int(sequenceMask); i++ {
		if _, err := generator.NextID(); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := generator.NextID(); err == nil {
		t.Fatal("esperaba error por timestamp de 42 bits tras desbordar la secuencia")
	}
}

func TestGeneratorIDsIncreaseWithRealClock(t *testing.T) {
	generator := NewGenerator(0)
	var previous int64
	for i := 0; i < 1000; i++ {
		id, err := generator.NextID()
		if err != nil {
			t.Fatal(err)
		}
		if id <= previous {
			t.Fatalf("ids no crecientes en %d: %d, %d", i, previous, id)
		}
		previous = id
	}
}

func TestParseWorkerID(t *testing.T) {
	cases := []struct {
		raw     string
		want    int64
		wantErr bool
	}{
		{"0", 0, false},
		{"7", 7, false},
		{" 1023 ", 1023, false},
		{"-1", 0, true},
		{"1024", 0, true},
		{"abc", 0, true},
		{"", 0, true},
	}

	for _, test := range cases {
		got, err := parseWorkerID(test.raw)
		if test.wantErr {
			if err == nil {
				t.Errorf("parseWorkerID(%q) no devolvió error", test.raw)
			}
			continue
		}
		if err != nil || got != test.want {
			t.Errorf("parseWorkerID(%q) = (%d, %v), esperaba (%d, nil)", test.raw, got, err, test.want)
		}
	}
}
