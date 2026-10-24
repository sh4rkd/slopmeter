package main

import (
	"encoding/binary"
	"encoding/hex"
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

	// 1. Hidden Cube 1 (1200039) 20m East (+X) and 5m Up (+Z) -> (13000, 10000, 5500)
	chest1Payload := buildSpawnPayload(501, 4, "", 1200039, 13000, 10000, 5500)
	fight.observeRadar(aMessage{
		opcode:  0x3640,
		flags:   wire.FromServer,
		payload: chest1Payload,
		t:       now.Add(200 * time.Millisecond),
	}, self)

	// 2. Hidden Cube 2 (1000295) 15m South (+Y) and 4m Down (-Z) -> (11000, 11500, 4600)
	chest2Payload := buildSpawnPayload(502, 4, "", 1000295, 11000, 11500, 4600)
	fight.observeRadar(aMessage{
		opcode:  0x3641,
		flags:   wire.FromServer,
		payload: chest2Payload,
		t:       now.Add(300 * time.Millisecond),
	}, self)

	// 3. Asmodian Hidden Cube (1301199) 10m North (-Y) and Level (Z=5000) -> (11000, 9000, 5000)
	chest3Payload := buildSpawnPayload(503, 2, "", 1301199, 11000, 9000, 5000)
	fight.observeRadar(aMessage{
		opcode:  0x3640,
		flags:   wire.FromServer,
		payload: chest3Payload,
		t:       now.Add(400 * time.Millisecond),
	}, self)

	// 4. Cube with inline name "Rare Hidden Chest" (code 9999999) 12m South (+Y) and 3m Up (+Z) -> (11000, 11200, 5300)
	chest4Payload := buildSpawnPayload(504, 4, "Rare Hidden Chest", 9999999, 11000, 11200, 5300)
	fight.observeRadar(aMessage{
		opcode:  0x3640,
		flags:   wire.FromServer,
		payload: chest4Payload,
		t:       now.Add(450 * time.Millisecond),
	}, self)

	// 5. Hidden Cube V04 (1200040) 25m West (-X) -> (8500, 10000, 5000)
	chest5Payload := buildSpawnPayload(505, 4, "", 1200040, 8500, 10000, 5000)
	fight.observeRadar(aMessage{
		opcode:  0x3640,
		flags:   wire.FromServer,
		payload: chest5Payload,
		t:       now.Add(480 * time.Millisecond),
	}, self)

	// 6. Asvata Log resource (1090201) -> tracked on radar as Wood
	woodPayload := buildSpawnPayload(506, 4, "", 1090201, 12800, 10000, 5000)
	fight.observeRadar(aMessage{
		opcode:  0x3640,
		flags:   wire.FromServer,
		payload: woodPayload,
		t:       now.Add(500 * time.Millisecond),
	}, self)

	// 7. Non-catalog monster or scenery entity (9990001) -> should NOT be tracked on radar
	ignoredPayload := buildSpawnPayload(507, 4, "", 9990001, 13000, 10000, 5000)
	fight.observeRadar(aMessage{
		opcode:  0x3640,
		flags:   wire.FromServer,
		payload: ignoredPayload,
		t:       now.Add(550 * time.Millisecond),
	}, self)

	snap := fight.snapshot(identities{self: self, name: "Tester"}, now.Add(time.Second))
	if len(snap.Radar) != 6 {
		t.Fatalf("expected 6 radar items (cubes + gathering), got %d", len(snap.Radar))
	}

	byID := make(map[game.Entity]radarDisplay)
	for _, item := range snap.Radar {
		byID[item.ID] = item
	}

	chest1 := byID[501]
	if chest1.Category != radarCategoryChest || chest1.Name != "Hidden Cube" || chest1.HorizDir != "Este" || chest1.VertDir != "Arriba" || chest1.ZDelta != 5.0 || chest1.RelX != 20.0 || chest1.RelY != 0.0 {
		t.Fatalf("unexpected chest1 radar item: %+v", chest1)
	}
	chest2 := byID[502]
	if chest2.Category != radarCategoryChest || chest2.Name != "Hidden Cube" || chest2.HorizDir != "Sur" || chest2.VertDir != "Abajo" || chest2.ZDelta != -4.0 || chest2.RelX != 0.0 || chest2.RelY != -15.0 {
		t.Fatalf("unexpected chest2 radar item: %+v", chest2)
	}
	chest3 := byID[503]
	if chest3.Category != radarCategoryChest || chest3.Name != "Asmodian Hidden Cube V04" || chest3.HorizDir != "Norte" || chest3.VertDir != "Nivel" || chest3.Distance != 10.0 || chest3.RelX != 0.0 || chest3.RelY != 10.0 {
		t.Fatalf("unexpected chest3 radar item: %+v", chest3)
	}
	chest4 := byID[504]
	if chest4.Category != radarCategoryChest || chest4.Name != "Rare Hidden Chest" || chest4.HorizDir != "Sur" || chest4.VertDir != "Arriba" || chest4.ZDelta != 3.0 || chest4.RelX != 0.0 || chest4.RelY != -12.0 {
		t.Fatalf("unexpected chest4 radar item: %+v", chest4)
	}
	chest5 := byID[505]
	if chest5.Category != radarCategoryChest || chest5.Name != "Hidden Cube V04" || chest5.HorizDir != "Oeste" || chest5.VertDir != "Nivel" || chest5.Distance != 25.0 || chest5.RelX != -25.0 || chest5.RelY != 0.0 {
		t.Fatalf("unexpected chest5 radar item: %+v", chest5)
	}
	wood6 := byID[506]
	if wood6.Category != radarCategoryWood || wood6.Name != "Asvata Log" || wood6.HorizDir != "Este" || wood6.VertDir != "Nivel" || wood6.RelX != 18.0 {
		t.Fatalf("unexpected wood6 radar item: %+v", wood6)
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
	name := "Special Hidden Cube"
	p = append(p, byte(len(name)))
	p = append(p, []byte(name)...)
	var codeBytes [4]byte
	binary.LittleEndian.PutUint32(codeBytes[:], 9999999)
	p = append(p, codeBytes[:]...)
	p = append(p, 0x40, 0x02) // subtype
	var posBytes [12]byte
	putVec3(posBytes[:], 5000, 7000, 1000) // 20m South (+Y)
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
	if item.Name != "Special Hidden Cube" || item.Category != radarCategoryChest || item.Distance != 20.0 || item.HorizDir != "Sur" {
		t.Fatalf("unexpected item parsed: %+v", item)
	}
}

func TestRadarClassificationAccuracy(t *testing.T) {
	// Verified Hidden Cubes and Gathering Nodes should succeed
	validTargets := []struct {
		code     uint32
		wantCat  radarCategory
		wantName string
	}{
		{1000295, radarCategoryChest, "Hidden Cube"},
		{1200039, radarCategoryChest, "Hidden Cube"},
		{1301199, radarCategoryChest, "Asmodian Hidden Cube V04"},
		{1401335, radarCategoryChest, "Hidden Cube"},
		{1080101, radarCategoryOre, "Orichalcum Ore"},
		{1080601, radarCategoryHerb, "Aria"},
		{1080501, radarCategoryGem, "Sapphire Gemstone"},
		{1090201, radarCategoryWood, "Asvata Log"},
		{1080001, radarCategoryOdyle, "Odyle"},
	}

	for _, tc := range validTargets {
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

	// Non-catalog monsters, combat NPCs and random background entities should be completely ignored (ok == false)
	ignoredCodes := []uint32{
		1081500, // Non-catalog ID
		1050000, // Random mob
		1580050, // Non-catalog ID
		1080550, // Non-catalog ID
		1080150, // Non-catalog ID
		1080250, // Non-catalog ID
		1200550, // Misc NPC
		1305100, // Misc NPC
		2900150, // Misc Object
		700100,  // Misc Object
	}

	for _, code := range ignoredCodes {
		_, _, ok := classifyRadarTarget(code, 0, "")
		if ok {
			t.Errorf("code %d: expected non-cube entity to be ignored (ok=false), but got ok=true", code)
		}
	}
}

func TestRadarRealWorldCubeLifecycle3436(t *testing.T) {
	now := time.Now()
	fight := newEncounter(0, 5*time.Second, 90*time.Second)

	// Set player position ~20m from cube
	selfPosBytes := buildSelfBuff2A(100, -50205.0, 126283.0-2000, 15449.0)
	fight.observeRadar(aMessage{
		opcode:  0x382A,
		flags:   wire.FromServer,
		payload: selfPosBytes,
		t:       now,
	}, 100)

	// 1. Spawn cube 65509 (template 1200039) via opcode 0x3634
	spawnHex := "e5ff030400a74f1200001d44c780a5f647006471460000000000000000010101aae4a534a101000000000000000000002c4d4502070000"
	spawnBytes, err := hex.DecodeString(spawnHex)
	if err != nil {
		t.Fatalf("hex decode error: %v", err)
	}
	fight.observeRadar(aMessage{
		opcode:  0x3634,
		flags:   wire.FromServer,
		payload: spawnBytes,
		t:       now.Add(10 * time.Millisecond),
	}, 100)

	snap := fight.snapshot(identities{self: 100, name: "Tester"}, now.Add(20*time.Millisecond))
	if len(snap.Radar) != 1 {
		t.Fatalf("expected 1 radar item from 0x3634 spawn, got %d", len(snap.Radar))
	}
	cube := snap.Radar[0]
	if cube.ID != 65509 || cube.Category != radarCategoryChest || cube.Name != "Hidden Cube" {
		t.Fatalf("unexpected cube radar item: %+v", cube)
	}
	if math.Abs(cube.Distance-20.0) > 0.5 {
		t.Fatalf("expected distance ~20m, got %.1f", cube.Distance)
	}

	// 2. Remove cube 65509 via opcode 0x3635
	removeHex := "e5ff03040300"
	removeBytes, err := hex.DecodeString(removeHex)
	if err != nil {
		t.Fatalf("hex decode error: %v", err)
	}
	fight.observeRadar(aMessage{
		opcode:  0x3635,
		flags:   wire.FromServer,
		payload: removeBytes,
		t:       now.Add(30 * time.Millisecond),
	}, 100)

	snapAfter := fight.snapshot(identities{self: 100, name: "Tester"}, now.Add(40*time.Millisecond))
	if len(snapAfter.Radar) != 0 {
		t.Fatalf("expected 0 radar items after 0x3635 removal, got %d", len(snapAfter.Radar))
	}
}

func TestRadarDespawnOpcodes(t *testing.T) {
	now := time.Now()
	fight := newEncounter(0, 5*time.Second, 90*time.Second)

	// Set player position at origin
	selfPosBytes := buildSelfBuff2A(100, 1000.0, 1000.0, 100.0)
	fight.observeRadar(aMessage{
		opcode:  0x382A,
		flags:   wire.FromServer,
		payload: selfPosBytes,
		t:       now,
	}, 100)

	// Spawn cube 888 close to player (within 3m = 300cm)
	fight.radar.upsertEntity(now, 888, 1200039, 0, "Hidden Cube", game.Pos{X: 1100.0, Y: 1000.0, Z: 100.0})
	// Spawn cube 999 far from player (50m away)
	fight.radar.upsertEntity(now, 999, 1200039, 0, "Hidden Cube", game.Pos{X: 6000.0, Y: 1000.0, Z: 100.0})

	if len(fight.radar.entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(fight.radar.entities))
	}

	// 1. Test 0x8D6D: collects nearby item (888 within 600cm)
	fight.observeRadar(aMessage{
		opcode:  0x8D6D,
		flags:   wire.FromServer,
		payload: []byte{0x01, 0x02},
		t:       now.Add(10 * time.Millisecond),
	}, 100)

	if _, found := fight.radar.entities[888]; found {
		t.Fatalf("expected entity 888 to be removed by nearby loot opcode 0x8D6D")
	}
	if _, found := fight.radar.entities[999]; !found {
		t.Fatalf("expected entity 999 to remain after 0x8D6D")
	}

	// 2. Test 0x3642: despawns entity 999
	payload999 := append(binary.AppendUvarint(nil, 999), 0x01, 0x02)
	fight.observeRadar(aMessage{
		opcode:  0x3642,
		flags:   wire.FromServer,
		payload: payload999,
		t:       now.Add(20 * time.Millisecond),
	}, 100)

	if len(fight.radar.entities) != 0 {
		t.Fatalf("expected 0 entities after 0x3642 despawn, got %d", len(fight.radar.entities))
	}
}




func TestRadarNavigationArrowIsStable(t *testing.T) {
	now := time.Now()
	fight := newEncounter(0, 5*time.Second, 90*time.Second)
	fight.radar.upsertEntity(now, 900, 1200039, 0, "", game.Pos{X: 1000, Y: 3000, Z: 100})
	// Walk north (+X) in 50 cm steps with sideways jitter.
	for i := 0; i <= 10; i++ {
		jitter := float32(i%2) * 20
		fight.radar.updateSelfPos(game.Pos{X: 1000 + float32(i)*50 - 500, Y: 1000 + jitter, Z: 100})
	}
	snap := fight.radarSnapshot()
	if len(snap) != 1 || !snap[0].HasHeading {
		t.Fatalf("expected heading after walking, got %+v", snap)
	}
	if math.Abs(snap[0].RelAngle-90) > 10 {
		t.Fatalf("cube east while walking north should be ~90° right, got %v", snap[0].RelAngle)
	}
	// Standing still with tiny jitter must not change the arrow.
	before := snap[0].RelAngle
	for i := 0; i < 20; i++ {
		fight.radar.updateSelfPos(game.Pos{X: 1000 + float32(i%3)*10, Y: 1000 - float32(i%2)*10, Z: 100})
	}
	if after := fight.radarSnapshot()[0].RelAngle; math.Abs(after-before) > 15 {
		t.Fatalf("arrow moved while standing still: %v -> %v", before, after)
	}
	// Casting at a target must not turn the arrow.
	fight.observeRadar(aMessage{flags: wire.FromServer, event: game.Cast{Actor: 7, Target: 900, Pos: game.Pos{X: 1000, Y: 3000, Z: 100}}, t: now}, 7)
	if after := fight.radarSnapshot()[0].RelAngle; math.Abs(after-before) > 15 {
		t.Fatalf("cast turned the arrow: %v -> %v", before, after)
	}
}

// Real 0x382A/0x382B tails from dr89mh/aion2-cube-radar docs/FINDINGS.md (P2), self entity 9180.
func TestParseSelfBuffPosRealBodies(t *testing.T) {
	b2a := []byte{0xdc, 0x47, 0x01, 0x11, 0x90, 0x03, 0xe8, 0x03, 0x23, 0x51, 0x96, 0x34, 0xa1, 0x01, 0x00, 0x00,
		0xdc, 0x47, 0x01, 0x00, 0xb3, 0x98, 0x4a, 0x46, 0xa2, 0x79, 0x9f, 0xc7, 0x51, 0x3e, 0x59, 0x44}
	pos, id, ok := parseSelfBuffPos(b2a, false)
	if !ok || id != 9180 || math.Abs(float64(pos.X)-12966) > 1 || math.Abs(float64(pos.Y)+81651) > 1 || math.Abs(float64(pos.Z)-869) > 1 {
		t.Fatalf("2A: ok=%v id=%d pos=%+v", ok, id, pos)
	}
	// Record about another entity (trailing id differs) must be rejected.
	other := append([]byte{0x9a, 0x70}, b2a[2:]...)
	if _, _, ok := parseSelfBuffPos(other, false); ok {
		t.Fatal("mismatched caster ids must be rejected")
	}
	b2b := []byte{0xdc, 0x47, 0x13, 0x84, 0x03, 0x28, 0x61, 0x96, 0x34, 0xa1, 0x01, 0x00, 0x00,
		0xdc, 0x47, 0x01, 0x02, 0xe7, 0x40, 0x4a, 0x46, 0x95, 0xa1, 0x9f, 0xc7, 0x00, 0x40, 0x59, 0x44}
	pos, id, ok = parseSelfBuffPos(b2b, true)
	if !ok || id != 9180 || math.Abs(float64(pos.X)-12944) > 1 || math.Abs(float64(pos.Y)+81731) > 1 {
		t.Fatalf("2B: ok=%v id=%d pos=%+v", ok, id, pos)
	}
}

// buildSelfBuff2A builds a 0x382A body in the real layout:
// caster varint | ... | caster varint | 01 00 | x y z.
func buildSelfBuff2A(id uint32, x, y, z float32) []byte {
	v := binary.AppendUvarint(nil, uint64(id))
	p := append([]byte{}, v...)
	p = append(p, 0x01, 0x11, 0x90, 0x03, 0xe8, 0x03, 0, 0)
	p = append(p, v...)
	p = append(p, 0x01, 0x00)
	var pos [12]byte
	putVec3(pos[:], x, y, z)
	return append(p, pos[:]...)
}
