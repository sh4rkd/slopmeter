package main

import (
	"encoding/binary"
	"math"
	"testing"
	"time"

	"github.com/nuriland/a2kit/game"
	"github.com/nuriland/a2kit/wire"
)

func putVec3(b []byte, x, y, z float32) {
	binary.LittleEndian.PutUint32(b[0:4], math.Float32bits(x))
	binary.LittleEndian.PutUint32(b[4:8], math.Float32bits(y))
	binary.LittleEndian.PutUint32(b[8:12], math.Float32bits(z))
}

func buildSpawnPayload(entity uint32, maskWidth int, inlineName string, npcCode uint32, x, y, z float32) []byte {
	var buf []byte
	var varBuf [10]byte
	n := binary.PutUvarint(varBuf[:], uint64(entity))
	buf = append(buf, varBuf[:n]...)
	if maskWidth == 4 {
		buf = append(buf, 0x1c, 0x00, 0x00, 0x00)
	} else {
		buf = append(buf, 0x1c, 0x00)
	}
	if inlineName != "" {
		buf = append(buf, 0x01)
		n = binary.PutUvarint(varBuf[:], uint64(len(inlineName)))
		buf = append(buf, varBuf[:n]...)
		buf = append(buf, []byte(inlineName)...)
	} else {
		buf = append(buf, 0x00)
	}
	var codeBytes [4]byte
	binary.LittleEndian.PutUint32(codeBytes[:], npcCode)
	buf = append(buf, codeBytes[:]...)
	buf = append(buf, 0x40, 0x02) // 2B subtype
	var posBytes [12]byte
	putVec3(posBytes[:], x, y, z)
	buf = append(buf, posBytes[:]...)
	return buf
}

func TestRadarDirectionsAndCategories(t *testing.T) {
	now := time.Now()
	fight := newEncounter(0, 5*time.Second, 90*time.Second)
	self := game.Entity(100)

	// Player starts at (10000, 10000, 5000) and moves +X (North/Forward) to (11000, 10000, 5000)
	fight.observeRadar(aMessage{
		opcode: 0x3623,
		flags:  wire.FromServer,
		event:  game.Zone{Pos: game.Pos{X: 10000, Y: 10000, Z: 5000}},
		t:      now,
	}, self)
	fight.observeRadar(aMessage{
		opcode: 0x371a,
		flags:  wire.FromServer,
		event:  game.Move{Entity: self, Pos: game.Pos{X: 11000, Y: 10000, Z: 5000}},
		t:      now.Add(100 * time.Millisecond),
	}, self)

	// 1. Supply Chest 20m in Front (+X) and 5m Up (+Z) -> (13000, 10000, 5500)
	chestPayload := buildSpawnPayload(501, 4, "Cofre de suministro", 2900100, 13000, 10000, 5500)
	fight.observeRadar(aMessage{
		opcode:  0x3640,
		flags:   wire.FromServer,
		payload: chestPayload,
		t:       now.Add(200 * time.Millisecond),
	}, self)

	// 2. Odylium Ore 15m to the Right (+Y) and 4m Down (-Z) -> (11000, 11500, 4600)
	odylePayload := buildSpawnPayload(502, 4, "", 2920213, 11000, 11500, 4600)
	fight.observeRadar(aMessage{
		opcode:  0x3641,
		flags:   wire.FromServer,
		payload: odylePayload,
		t:       now.Add(300 * time.Millisecond),
	}, self)

	// 3. Wildgrass Herb 10m to the Left (-Y) and Level (Z=5000) -> (11000, 9000, 5000)
	herbPayload := buildSpawnPayload(503, 2, "", 2101376, 11000, 9000, 5000)
	fight.observeRadar(aMessage{
		opcode:  0x3640,
		flags:   wire.FromServer,
		payload: herbPayload,
		t:       now.Add(400 * time.Millisecond),
	}, self)

	// 4. Dimensional Core 30m Behind (-X) -> (8000, 10000, 5000)
	corePayload := buildSpawnPayload(504, 4, "", 2600078, 8000, 10000, 5000)
	fight.observeRadar(aMessage{
		opcode:  0x3640,
		flags:   wire.FromServer,
		payload: corePayload,
		t:       now.Add(500 * time.Millisecond),
	}, self)

	snap := fight.snapshot(identities{self: self, name: "Tester"}, now.Add(time.Second))
	if len(snap.Radar) != 4 {
		t.Fatalf("expected 4 radar items, got %d", len(snap.Radar))
	}

	byID := make(map[game.Entity]radarDisplay)
	for _, item := range snap.Radar {
		byID[item.ID] = item
	}

	chest := byID[501]
	if chest.Category != radarCategoryChest || chest.HorizDir != "Frente" || chest.VertDir != "Arriba" || chest.ZDelta != 5.0 {
		t.Fatalf("unexpected chest radar item: %+v", chest)
	}
	odyle := byID[502]
	if odyle.Category != radarCategoryOdyle || odyle.HorizDir != "Der" || odyle.VertDir != "Abajo" || odyle.ZDelta != -4.0 {
		t.Fatalf("unexpected odyle radar item: %+v", odyle)
	}
	herb := byID[503]
	if herb.Category != radarCategoryResource || herb.HorizDir != "Izq" || herb.VertDir != "Nivel" || herb.Distance != 10.0 {
		t.Fatalf("unexpected herb radar item: %+v", herb)
	}
	core := byID[504]
	if core.Category != radarCategoryResource || core.HorizDir != "Atrás" || core.VertDir != "Nivel" || core.Distance != 30.0 {
		t.Fatalf("unexpected core radar item: %+v", core)
	}

	// Despawn chest on Death packet
	fight.observeRadar(aMessage{
		opcode: 0x3642,
		flags:  wire.FromServer,
		event:  game.Death{Entity: 501, Flag: 3},
		t:      now.Add(600 * time.Millisecond),
	}, self)

	snapAfter := fight.snapshot(identities{self: self, name: "Tester"}, now.Add(2*time.Second))
	if len(snapAfter.Radar) != 3 {
		t.Fatalf("expected 3 radar items after chest despawn, got %d", len(snapAfter.Radar))
	}
}
