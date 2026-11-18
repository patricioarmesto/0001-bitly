package main

import (
	"fmt"
	"sync"
	"time"
)

const (
	// 2024-01-01T00:00:00Z: el timestamp arranca en 0 y cabe en 41 bits.
	snowflakeEpochMS int64 = 1704067200000

	workerIDBits  = uint(10)
	sequenceBits  = uint(12)
	timestampBits = uint(41)

	workerIDShift  = sequenceBits
	timestampShift = workerIDBits + sequenceBits

	maxWorkerID  = int64(1<<workerIDBits - 1)
	sequenceMask = int64(1<<sequenceBits - 1)
	maxTimestamp = int64(1<<timestampBits - 1)
)

// Generator produce IDs Snowflake: [timestamp 41 | worker 10 | secuencia 12].
// El estado es local al proceso; la unicidad depende de que cada proceso
// tenga un workerID distinto.
type Generator struct {
	mu            sync.Mutex
	lastTimestamp int64 // milisegundos desde snowflakeEpochMS
	sequence      int64
	workerID      int64
	now           func() int64
	sleep         func(time.Duration)
}

func NewGenerator(workerID int64) *Generator {
	return &Generator{
		workerID: workerID,
		now:      func() int64 { return time.Now().UnixMilli() - snowflakeEpochMS },
		sleep:    time.Sleep,
	}
}

func (g *Generator) NextID() (int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.now()
	for now < g.lastTimestamp {
		// El reloj retrocedió: esperar a alcanzar el último timestamp para
		// no volver a emitir los mismos IDs.
		g.sleep(time.Duration(g.lastTimestamp-now) * time.Millisecond)
		now = g.now()
	}

	if now == g.lastTimestamp {
		g.sequence++
		if g.sequence > sequenceMask {
			// Se llenaron los 4096 valores del milisegundo: esperar al siguiente.
			for now <= g.lastTimestamp {
				g.sleep(time.Millisecond)
				now = g.now()
			}
			g.sequence = 0
		}
	} else {
		g.sequence = 0
	}
	g.lastTimestamp = now

	if now > maxTimestamp {
		return 0, fmt.Errorf("timestamp %d excede los 41 bits", now)
	}
	return now<<timestampShift | g.workerID<<workerIDShift | g.sequence, nil
}
