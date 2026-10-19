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

	// 1. Hidden Cube (1200039) 20m in Front (+X) and 5m Up (+Z) -> (13000, 10000, 5500)
	chestPayload := buildSpawnPayload(501, 4, "", 1200039, 13000, 10000, 5500)
	fight.observeRadar(aMessage{
		opcode:  0x3640,
		flags:   wire.FromServer,
		payload: chestPayload,
		t:       now.Add(200 * time.Millisecond),
	}, self)

	// 2. Odyle (1080001) 15m to the Right (+Y) and 4m Down (-Z) -> (11000, 11500, 4600)
	odylePayload := buildSpawnPayload(502, 4, "", 1080001, 11000, 11500, 4600)
	fight.observeRadar(aMessage{
		opcode:  0x3641,
		flags:   wire.FromServer,
		payload: odylePayload,
		t:       now.Add(300 * time.Millisecond),
	}, self)

	// 3. Orichalcum Ore (1080101) 10m to the Left (-Y) and Level (Z=5000) -> (11000, 9000, 5000)
	orePayload := buildSpawnPayload(503, 2, "", 1080101, 11000, 9000, 5000)
	fight.observeRadar(aMessage{
		opcode:  0x3640,
		flags:   wire.FromServer,
		payload: orePayload,
		t:       now.Add(400 * time.Millisecond),
	}, self)

	// 4. Sapphire Gemstone (1080501) 12m to the Right (+Y) and 3m Up (+Z) -> (11000, 11200, 5300)
	gemPayload := buildSpawnPayload(504, 4, "", 1080501, 11000, 11200, 5300)
	fight.observeRadar(aMessage{
		opcode:  0x3640,
		flags:   wire.FromServer,
		payload: gemPayload,
		t:       now.Add(450 * time.Millisecond),
	}, self)

	// 5. Azpha (1090601) 25m Behind (-X) -> (8500, 10000, 5000)
	herbPayload := buildSpawnPayload(505, 4, "", 1090601, 8500, 10000, 5000)
	fight.observeRadar(aMessage{
		opcode:  0x3640,
		flags:   wire.FromServer,
		payload: herbPayload,
		t:       now.Add(480 * time.Millisecond),
	}, self)

	// 6. Asvata Log (1090201) 18m Front (+X) -> (12800, 10000, 5000)
	woodPayload := buildSpawnPayload(506, 4, "", 1090201, 12800, 10000, 5000)
	fight.observeRadar(aMessage{
		opcode:  0x3640,
		flags:   wire.FromServer,
		payload: woodPayload,
		t:       now.Add(500 * time.Millisecond),
	}, self)

	snap := fight.snapshot(identities{self: self, name: "Tester"}, now.Add(time.Second))
	if len(snap.Radar) != 6 {
		t.Fatalf("expected 6 radar items, got %d", len(snap.Radar))
	}

	byID := make(map[game.Entity]radarDisplay)
	for _, item := range snap.Radar {
		byID[item.ID] = item
	}

	chest := byID[501]
	if chest.Category != radarCategoryChest || chest.Name != "Hidden Cube" || chest.HorizDir != "Frente" || chest.VertDir != "Arriba" || chest.ZDelta != 5.0 || chest.RelX != 0.0 || chest.RelY != 20.0 {
		t.Fatalf("unexpected chest radar item: %+v", chest)
	}
	odyle := byID[502]
	if odyle.Category != radarCategoryOdyle || odyle.Name != "Odyle" || odyle.HorizDir != "Der" || odyle.VertDir != "Abajo" || odyle.ZDelta != -4.0 || odyle.RelX != 15.0 || odyle.RelY != 0.0 {
		t.Fatalf("unexpected odyle radar item: %+v", odyle)
	}
	ore := byID[503]
	if ore.Category != radarCategoryOre || ore.Name != "Orichalcum Ore" || ore.HorizDir != "Izq" || ore.VertDir != "Nivel" || ore.Distance != 10.0 || ore.RelX != -10.0 || ore.RelY != 0.0 {
		t.Fatalf("unexpected ore radar item: %+v", ore)
	}
	gem := byID[504]
	if gem.Category != radarCategoryGem || gem.Name != "Sapphire Gemstone" || gem.HorizDir != "Der" || gem.VertDir != "Arriba" || gem.ZDelta != 3.0 || gem.RelX != 12.0 || gem.RelY != 0.0 {
		t.Fatalf("unexpected gem radar item: %+v", gem)
	}
	herb := byID[505]
	if herb.Category != radarCategoryHerb || herb.Name != "Azpha" || herb.HorizDir != "Atrás" || herb.VertDir != "Nivel" || herb.Distance != 25.0 || herb.RelX != 0.0 || herb.RelY != -25.0 {
		t.Fatalf("unexpected herb radar item: %+v", herb)
	}
	wood := byID[506]
	if wood.Category != radarCategoryWood || wood.Name != "Asvata Log" || wood.HorizDir != "Frente" || wood.VertDir != "Nivel" || wood.Distance != 18.0 || wood.RelX != 0.0 || wood.RelY != 18.0 {
		t.Fatalf("unexpected wood radar item: %+v", wood)
	}

	// Despawn chest on Death packet
	fight.observeRadar(aMessage{
		opcode: 0x3642,
		flags:  wire.FromServer,
		event:  game.Death{Entity: 501, Flag: 3},
		t:      now.Add(600 * time.Millisecond),
	}, self)

	snapAfter := fight.snapshot(identities{self: self, name: "Tester"}, now.Add(2*time.Second))
	if len(snapAfter.Radar) != 5 {
		t.Fatalf("expected 5 radar items after chest despawn, got %d", len(snapAfter.Radar))
	}
}

func TestRadarClientMovementAndByteInlineName(t *testing.T) {
	now := time.Now()
	fight := newEncounter(0, 5*time.Second, 90*time.Second)

	// Client sends outgoing Move (wire.FromServer == 0) from selfEntity=200
	fight.observeRadar(aMessage{
		opcode: 0x371a,
		flags:  0, // client packet
		event:  game.Move{Entity: 200, Pos: game.Pos{X: 5000, Y: 5000, Z: 1000}},
		t:      now,
	}, 0)

	// Spawn with 1-byte length inline name
	var p []byte
	p = binary.AppendUvarint(p, 777)
	p = append(p, 0x1c, 0x00, 0x00, 0x00) // 4-byte mask
	p = append(p, 0x01)                   // gate: inline name present
	name := "Ruby Gemstone"
	p = append(p, byte(len(name)))
	p = append(p, []byte(name)...)
	var codeBytes [4]byte
	binary.LittleEndian.PutUint32(codeBytes[:], 1080503)
	p = append(p, codeBytes[:]...)
	p = append(p, 0x40, 0x02) // subtype
	var posBytes [12]byte
	putVec3(posBytes[:], 5000, 7000, 1000) // 20m Right (+Y)
	p = append(p, posBytes[:]...)

	fight.observeRadar(aMessage{
		opcode:  0x3640,
		flags:   wire.FromServer,
		payload: p,
		t:       now.Add(100 * time.Millisecond),
	}, 200)

	snap := fight.snapshot(identities{self: 200, name: "Tester"}, now.Add(200*time.Millisecond))
	if len(snap.Radar) != 1 {
		t.Fatalf("expected 1 radar item, got %d", len(snap.Radar))
	}
	item := snap.Radar[0]
	if item.Name != "Ruby Gemstone" || item.Category != radarCategoryGem || item.Distance != 20.0 || item.HorizDir != "Der" {
		t.Fatalf("unexpected item parsed: %+v", item)
	}
}

func TestRadarBroadRangesClassification(t *testing.T) {
	cases := []struct {
		code     uint32
		wantCat  radarCategory
		wantName string
	}{
		{1200550, radarCategoryChest, "Hidden Cube (1200550)"},
		{1305100, radarCategoryChest, "Hidden Cube (1305100)"},
		{2900150, radarCategoryChest, "Hidden Cube / Chest (2900150)"},
		{700100, radarCategoryChest, "Hidden Cube (700100)"},
		{1081500, radarCategoryHerb, "Herb / Plant (1081500)"},
		{1050000, radarCategoryHerb, "Herb / Plant (1050000)"},
		{1580050, radarCategoryOdyle, "Odyle (1580050)"},
		{1080550, radarCategoryGem, "Gemstone (1080550)"},
		{1080150, radarCategoryOre, "Orichalcum Ore (1080150)"},
		{1080250, radarCategoryWood, "Wood Log (1080250)"},
	}

	for _, tc := range cases {
		name, cat, ok := classifyRadarTarget(tc.code, 0, "")
		if !ok {
			t.Errorf("code %d: expected ok=true, got false", tc.code)
		}
		if cat != tc.wantCat {
			t.Errorf("code %d: expected cat %s, got %s", tc.code, tc.wantCat, cat)
		}
		if name != tc.wantName {
			t.Errorf("code %d: expected name %s, got %s", tc.code, tc.wantName, name)
		}
	}
}

