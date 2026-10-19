package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nuriland/a2kit/game"
	"github.com/nuriland/a2kit/wire"
)

type radarCategory string

const (
	radarCategoryChest    radarCategory = "chest"
	radarCategoryOdyle    radarCategory = "odyle"
	radarCategoryResource radarCategory = "resource"
	radarCategoryObject   radarCategory = "object"
)

type radarDefinition struct {
	Code     uint32
	Name     string
	Category radarCategory
}

type radarEntity struct {
	id         game.Entity
	npc        uint32
	mask       uint32
	inlineName string
	name       string
	category   radarCategory
	pos        game.Pos
	updated    time.Time
}

type radarState struct {
	entities   map[game.Entity]*radarEntity
	selfPos    game.Pos
	hasSelfPos bool
	headingX   float32
	headingY   float32
	hasHeading bool
	revision   uint64
}

type radarDisplay struct {
	ID       game.Entity   `json:"id"`
	NPC      uint32        `json:"npc"`
	Name     string        `json:"name"`
	Category radarCategory `json:"category"`
	Distance float64       `json:"distance"`
	ZDelta   float64       `json:"zDelta"`
	Angle    float64       `json:"angle"`
	HorizDir string        `json:"horizDir"`
	VertDir  string        `json:"vertDir"`
	HasDir   bool          `json:"hasDir"`
}

var radarCatalog = map[uint32]radarDefinition{
	// Hidden Cubes (Cubos Ocultos) - Ground & Water variants across Elyos (Verteron) & Asmodian (Altgard)
	1200039: {1200039, "Hidden Cube", radarCategoryChest},
	1200040: {1200040, "Hidden Cube", radarCategoryChest},
	1200041: {1200041, "Hidden Cube (Water)", radarCategoryChest},
	1301199: {1301199, "Hidden Cube", radarCategoryChest},
	1301200: {1301200, "Hidden Cube (Water)", radarCategoryChest},

	// Odyle / Odylium / Crystals / Spirits (GatherSource & World Entities)
	1080001: {1080001, "Odyle", radarCategoryOdyle},
	1080002: {1080002, "Fine Odyle", radarCategoryOdyle},
	1080003: {1080003, "Pure Odyle", radarCategoryOdyle},
	1080004: {1080004, "Radiant Odyle", radarCategoryOdyle},
	1090001: {1090001, "Odyle", radarCategoryOdyle},
	1090002: {1090002, "Fine Odyle", radarCategoryOdyle},
	1090003: {1090003, "Pure Odyle", radarCategoryOdyle},
	1090004: {1090004, "Radiant Odyle", radarCategoryOdyle},
	1580001: {1580001, "Odyle", radarCategoryOdyle},
	1580002: {1580002, "Fine Odyle", radarCategoryOdyle},
	1580003: {1580003, "Pure Odyle", radarCategoryOdyle},
	1590001: {1590001, "Odyle", radarCategoryOdyle},
	2000001: {2000001, "Odyle", radarCategoryOdyle},
	2920213: {2920213, "Odylium Ore", radarCategoryOdyle},
	2920225: {2920225, "Odylium Ore", radarCategoryOdyle},
	2920303: {2920303, "Odylium Ore", radarCategoryOdyle},
	2920703: {2920703, "Odylium Ore", radarCategoryOdyle},
	2921409: {2921409, "Odylium Ore", radarCategoryOdyle},
	2702174: {2702174, "Unstable Odylium Gemstone", radarCategoryOdyle},
	2090636: {2090636, "Odyle Shard Storm", radarCategoryOdyle},
	2680000: {2680000, "Odyle Shard Storm", radarCategoryOdyle},
	2701644: {2701644, "Ore Vein Odyle Spirit", radarCategoryOdyle},
	2701645: {2701645, "Ore Vein Odyle Spirit", radarCategoryOdyle},
	2101122: {2101122, "Odyle Spider", radarCategoryOdyle},
	2101132: {2101132, "Odyle Sparkie", radarCategoryOdyle},
	2101286: {2101286, "Odyle Stone Spirit", radarCategoryOdyle},
	2101287: {2101287, "Odyle Stone Spirit", radarCategoryOdyle},
	2000791: {2000791, "Crystal", radarCategoryOdyle},
	2000871: {2000871, "Crystal", radarCategoryOdyle},
	2301319: {2301319, "Crystal", radarCategoryOdyle},
	2311319: {2311319, "Crystal", radarCategoryOdyle},
	2321319: {2321319, "Crystal", radarCategoryOdyle},
	2921332: {2921332, "Atheron's Crystal", radarCategoryOdyle},
	2921333: {2921333, "Atheron's Crystal", radarCategoryOdyle},

	// Gathering Resources: Ores, Gems, Trees, Herbs, Feathers, Cores, Artifacts, Monoliths
	1080101: {1080101, "Orichalcum Ore", radarCategoryResource},
	1080102: {1080102, "Fine Orichalcum Ore", radarCategoryResource},
	1080103: {1080103, "Pure Orichalcum Ore", radarCategoryResource},
	1080104: {1080104, "Radiant Orichalcum Ore", radarCategoryResource},
	1080401: {1080401, "Orichalcum Ore", radarCategoryResource},
	1080402: {1080402, "Fine Orichalcum Ore", radarCategoryResource},
	1080403: {1080403, "Pure Orichalcum Ore", radarCategoryResource},
	1080404: {1080404, "Radiant Orichalcum Ore", radarCategoryResource},
	1090101: {1090101, "Orichalcum Ore", radarCategoryResource},
	1090102: {1090102, "Fine Orichalcum Ore", radarCategoryResource},
	1090103: {1090103, "Pure Orichalcum Ore", radarCategoryResource},
	1090104: {1090104, "Radiant Orichalcum Ore", radarCategoryResource},
	1090401: {1090401, "Orichalcum Ore", radarCategoryResource},
	1090402: {1090402, "Fine Orichalcum Ore", radarCategoryResource},
	1090403: {1090403, "Pure Orichalcum Ore", radarCategoryResource},
	1090404: {1090404, "Radiant Orichalcum Ore", radarCategoryResource},
	1080501: {1080501, "Sapphire Gemstone", radarCategoryResource},
	1080502: {1080502, "Diamond Gemstone", radarCategoryResource},
	1080503: {1080503, "Ruby Gemstone", radarCategoryResource},
	1090501: {1090501, "Sapphire Gemstone", radarCategoryResource},
	1090502: {1090502, "Diamond Gemstone", radarCategoryResource},
	1090503: {1090503, "Ruby Gemstone", radarCategoryResource},
	1090508: {1090508, "Diamond Gemstone", radarCategoryResource},
	1080201: {1080201, "Yggdrasil Log", radarCategoryResource},
	1080301: {1080301, "Yggdrasil Log", radarCategoryResource},
	1090201: {1090201, "Asvata Log", radarCategoryResource},
	1090301: {1090301, "Asvata Log", radarCategoryResource},
	1080601: {1080601, "Aria Herb", radarCategoryResource},
	1080602: {1080602, "Targena Herb", radarCategoryResource},
	1080604: {1080604, "Targena Herb", radarCategoryResource},
	1090601: {1090601, "Azpha Herb", radarCategoryResource},
	1090602: {1090602, "Calendula Herb", radarCategoryResource},
	1081001: {1081001, "Mela", radarCategoryResource},
	1081002: {1081002, "Coriolus", radarCategoryResource},
	1081003: {1081003, "Inina", radarCategoryResource},
	1081004: {1081004, "Cypri", radarCategoryResource},
	1081006: {1081006, "Kukuru", radarCategoryResource},
	1091001: {1091001, "Raydam", radarCategoryResource},
	1091002: {1091002, "Morfa", radarCategoryResource},
	1091003: {1091003, "Conide", radarCategoryResource},
	1091004: {1091004, "Pujery", radarCategoryResource},
	1091006: {1091006, "Ohkra", radarCategoryResource},
	2101376: {2101376, "Wildgrass Herb", radarCategoryResource},
	2701619: {2701619, "Herb Rafflesia", radarCategoryResource},
	2290068: {2290068, "Poison Grass", radarCategoryResource},
	2701147: {2701147, "Deadly Poison Grass", radarCategoryResource},
	2300254: {2300254, "Floating Tourmaline", radarCategoryResource},
	2300255: {2300255, "Floating Tourmaline", radarCategoryResource},
	2300264: {2300264, "Floating Tourmaline", radarCategoryResource},
	2300817: {2300817, "Floating Tourmaline", radarCategoryResource},
	2300818: {2300818, "Floating Tourmaline", radarCategoryResource},
	2590112: {2590112, "Veil Jewel", radarCategoryResource},
	2590224: {2590224, "Veil Jewel", radarCategoryResource},
	2600078: {2600078, "Dimensional Core", radarCategoryResource},
	2600161: {2600161, "Dimensional Core", radarCategoryResource},
	2600328: {2600328, "Dimensional Core", radarCategoryResource},
	2600496: {2600496, "Dimensional Core", radarCategoryResource},
	2600518: {2600518, "Dimensional Core", radarCategoryResource},
	2910008: {2910008, "Activated Ancient Artifact", radarCategoryResource},
	2910009: {2910009, "Deactivated Ancient Artifact", radarCategoryResource},
	2921151: {2921151, "Magic Feather", radarCategoryResource},
	2110690: {2110690, "Monolith", radarCategoryResource},
	2110691: {2110691, "Monolith", radarCategoryResource},
	2110692: {2110692, "Monolith", radarCategoryResource},
	2110693: {2110693, "Monolith", radarCategoryResource},
	2407130: {2407130, "Monolith", radarCategoryResource},
	2407131: {2407131, "Monolith", radarCategoryResource},
	2407132: {2407132, "Monolith", radarCategoryResource},

	// Supply Chests, Relics, Treasure Boxes
	2900100: {2900100, "Hidden Cube / Supply Chest", radarCategoryChest},
	2900101: {2900101, "Hidden Cube / Supply Chest", radarCategoryChest},
	2900102: {2900102, "Hidden Cube / Supply Chest", radarCategoryChest},
	2900200: {2900200, "Supply Chest", radarCategoryChest},
	2900201: {2900201, "Treasure Chest", radarCategoryChest},
	2920100: {2920100, "Supply Chest", radarCategoryChest},
}

func validWorldCoord(v float32) bool {
	if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
		return false
	}
	return v >= -2_000_000 && v <= 2_000_000
}

func validWorldPos(p game.Pos) bool {
	if !validWorldCoord(p.X) || !validWorldCoord(p.Y) || !validWorldCoord(p.Z) {
		return false
	}
	// Reject all-zero uninitialized coordinates.
	return math.Abs(float64(p.X))+math.Abs(float64(p.Y))+math.Abs(float64(p.Z)) > 1.0
}

func readVec3LE(b []byte) (game.Pos, bool) {
	if len(b) < 12 {
		return game.Pos{}, false
	}
	p := game.Pos{
		X: math.Float32frombits(binary.LittleEndian.Uint32(b[0:4])),
		Y: math.Float32frombits(binary.LittleEndian.Uint32(b[4:8])),
		Z: math.Float32frombits(binary.LittleEndian.Uint32(b[8:12])),
	}
	return p, validWorldPos(p)
}

func classifyRadarTarget(code uint32, mask uint32, inlineName string) (string, radarCategory, bool) {
	if def, ok := radarCatalog[code]; ok {
		name := def.Name
		if inlineName != "" {
			name = inlineName
		}
		return name, def.Category, true
	}
	lower := strings.ToLower(inlineName)
	if lower != "" {
		switch {
		case strings.Contains(lower, "cube") || strings.Contains(lower, "cubo") || strings.Contains(lower, "hidden") || strings.Contains(lower, "oculto") || strings.Contains(lower, "chest") || strings.Contains(lower, "supply") || strings.Contains(lower, "cofre") || strings.Contains(lower, "suministro") || strings.Contains(lower, "treasure") || strings.Contains(lower, "tesoro") || strings.Contains(lower, "relic") || strings.Contains(lower, "box") || strings.Contains(lower, "caja") || strings.Contains(lower, "큐브") || strings.Contains(lower, "상자") || strings.Contains(lower, "보급"):
			return inlineName, radarCategoryChest, true
		case strings.Contains(lower, "odyle") || strings.Contains(lower, "odylium") || strings.Contains(lower, "odil") || strings.Contains(lower, "aether") || strings.Contains(lower, "오드"):
			return inlineName, radarCategoryOdyle, true
		case strings.Contains(lower, "herb") || strings.Contains(lower, "ore") || strings.Contains(lower, "crystal") || strings.Contains(lower, "vein") || strings.Contains(lower, "tourmaline") || strings.Contains(lower, "jewel") || strings.Contains(lower, "artifact") || strings.Contains(lower, "core") || strings.Contains(lower, "hierba") || strings.Contains(lower, "mineral") || strings.Contains(lower, "cristal") || strings.Contains(lower, "광석") || strings.Contains(lower, "약초"):
			return inlineName, radarCategoryResource, true
		}
	}
	// Hidden Cube ID ranges in Aion 2 (1200000–1200099 Elyos Hidden Cubes, 1301190–1301210 Asmodian Hidden Cubes)
	if (code >= 1200000 && code <= 1200099) || (code >= 1301190 && code <= 1301210) {
		name := inlineName
		if name == "" {
			name = fmt.Sprintf("Hidden Cube (%d)", code)
		}
		return name, radarCategoryChest, true
	}
	// Odyle GatherSource ranges (1080001–1080099, 1090001–1090099, 1580001–1590099)
	if (code >= 1080001 && code <= 1080099) || (code >= 1090001 && code <= 1090099) || (code >= 1580001 && code <= 1590099) {
		name := inlineName
		if name == "" {
			name = fmt.Sprintf("Odyle (%d)", code)
		}
		return name, radarCategoryOdyle, true
	}
	// General GatherSource ranges (1080100–1091099: Orichalcum, Gemstones, Yggdrasil/Asvata Logs, Aria/Targena/Azpha/Calendula Herbs, etc.)
	if code >= 1080100 && code <= 1091099 {
		name := inlineName
		if name == "" {
			name = fmt.Sprintf("Gathering Node (%d)", code)
		}
		return name, radarCategoryResource, true
	}
	// 2900000–2929999 interactive field objects, chests, artifacts, and Odylium ores.
	if code >= 2900000 && code <= 2909999 {
		name := inlineName
		if name == "" {
			name = fmt.Sprintf("Hidden Cube / Chest (%d)", code)
		}
		return name, radarCategoryChest, true
	}
	if code >= 2920000 && code <= 2929999 {
		name := inlineName
		if name == "" {
			name = fmt.Sprintf("Field Node (%d)", code)
		}
		return name, radarCategoryResource, true
	}
	// If it's a confirmed boss, do not clutter the object scanner.
	if _, isBoss := bossCatalog[code]; isBoss {
		return "", "", false
	}
	name := inlineName
	if name == "" {
		name = fmt.Sprintf("Object #%d", code)
	}
	return name, radarCategoryObject, true
}

func (r *radarState) clear() {
	if len(r.entities) > 0 {
		r.entities = make(map[game.Entity]*radarEntity)
		r.revision++
	}
}

func (r *radarState) updateSelfPos(pos game.Pos) {
	if !validWorldPos(pos) {
		return
	}
	if r.hasSelfPos {
		dx := pos.X - r.selfPos.X
		dy := pos.Y - r.selfPos.Y
		dist := float32(math.Hypot(float64(dx), float64(dy)))
		// Update movement heading when player moves at least 30 units (0.3m)
		if dist >= 30 {
			r.headingX = dx / dist
			r.headingY = dy / dist
			r.hasHeading = true
		}
	}
	if !r.hasSelfPos || math.Abs(float64(pos.X-r.selfPos.X)) > 5 || math.Abs(float64(pos.Y-r.selfPos.Y)) > 5 || math.Abs(float64(pos.Z-r.selfPos.Z)) > 5 {
		r.selfPos = pos
		r.hasSelfPos = true
		r.revision++
	}
}

func (r *radarState) updateHeadingToward(targetPos game.Pos) {
	if !r.hasSelfPos || !validWorldPos(targetPos) {
		return
	}
	dx := targetPos.X - r.selfPos.X
	dy := targetPos.Y - r.selfPos.Y
	dist := float32(math.Hypot(float64(dx), float64(dy)))
	if dist >= 20 {
		r.headingX = dx / dist
		r.headingY = dy / dist
		r.hasHeading = true
		r.revision++
	}
}

func (r *radarState) upsertEntity(t time.Time, id game.Entity, code uint32, mask uint32, inlineName string, pos game.Pos) {
	if id == 0 || code == 0 || !validWorldPos(pos) {
		return
	}
	name, category, ok := classifyRadarTarget(code, mask, inlineName)
	if !ok {
		return
	}
	if r.entities == nil {
		r.entities = make(map[game.Entity]*radarEntity)
	}
	if existing, found := r.entities[id]; found {
		changed := existing.npc != code || existing.category != category || existing.name != name ||
			math.Abs(float64(existing.pos.X-pos.X)) > 5 || math.Abs(float64(existing.pos.Y-pos.Y)) > 5 || math.Abs(float64(existing.pos.Z-pos.Z)) > 5
		existing.npc = code
		existing.mask = mask
		if inlineName != "" {
			existing.inlineName = inlineName
		}
		existing.name = name
		existing.category = category
		existing.pos = pos
		existing.updated = t
		if changed {
			r.revision++
		}
		return
	}
	if len(r.entities) >= 512 {
		var oldestID game.Entity
		var oldestTime time.Time
		for k, v := range r.entities {
			if oldestID == 0 || v.updated.Before(oldestTime) {
				oldestID = k
				oldestTime = v.updated
			}
		}
		if oldestID != 0 {
			delete(r.entities, oldestID)
		}
	}
	r.entities[id] = &radarEntity{
		id:         id,
		npc:        code,
		mask:       mask,
		inlineName: inlineName,
		name:       name,
		category:   category,
		pos:        pos,
		updated:    t,
	}
	r.revision++
}

func (r *radarState) removeEntity(id game.Entity) {
	if r.entities != nil {
		if _, ok := r.entities[id]; ok {
			delete(r.entities, id)
			r.revision++
		}
	}
}

var selfPosSentinel = []byte{0x80, 0x75, 0xd5, 0x2a, 0xbb, 0x03, 0x00, 0x00}

func (e *encounter) observeRadar(m aMessage, self game.Entity) {
	if m.flags&wire.FromServer == 0 {
		return
	}
	switch ev := m.event.(type) {
	case game.Zone:
		e.radar.clear()
		e.radar.updateSelfPos(ev.Pos)
	case game.Move:
		if self != 0 && ev.Entity == self {
			e.radar.updateSelfPos(ev.Pos)
		} else if e.radar.entities != nil {
			if ent, ok := e.radar.entities[ev.Entity]; ok && validWorldPos(ev.Pos) {
				ent.pos = ev.Pos
				ent.updated = m.t
				e.radar.revision++
			}
		}
	case game.Cast:
		if self != 0 && ev.Actor == self {
			e.radar.updateSelfPos(ev.Pos)
			if ev.Target != 0 && e.radar.entities != nil {
				if targetEnt := e.radar.entities[ev.Target]; targetEnt != nil {
					e.radar.updateHeadingToward(targetEnt.pos)
				}
			}
		}
	case game.Death:
		e.radar.removeEntity(ev.Entity)
	case game.Spawn:
		e.radar.upsertEntity(m.t, ev.Entity, uint32(ev.NPC), ev.Mask, "", ev.Pos)
	}

	// Extract local player initial spawn coordinates from 0x3633 (Self) sentinel block.
	if m.opcode == 0x3633 {
		if idx := bytes.Index(m.payload, selfPosSentinel); idx >= 0 {
			tail := m.payload[idx+len(selfPosSentinel):]
			if id, rest, ok := readVar(tail); ok && (self == 0 || game.Entity(id) == self) {
				for skip := 1; skip <= 3 && skip+12 <= len(rest); skip++ {
					if pos, valid := readVec3LE(rest[skip : skip+12]); valid {
						e.radar.updateSelfPos(pos)
						break
					}
				}
			}
		}
	}

	// Extract coordinates from 0x3640 / 0x3641 spawns (both 4-byte and 2-byte mask widths, with or without inline UTF-8 name).
	if m.opcode == 0x3640 || m.opcode == 0x3641 {
		e.observeRadarSpawn(m.t, m.opcode, m.payload)
	}
}

func (e *encounter) observeRadarSpawn(t time.Time, opcode wire.Opcode, payload []byte) {
	id, rest, ok := readVar(payload)
	if !ok || id == 0 {
		return
	}
	for _, width := range []int{4, 2} {
		if len(rest) < width+1+4+2+12 {
			continue
		}
		var mask uint32
		if width == 4 {
			mask = binary.LittleEndian.Uint32(rest[:4])
		} else {
			mask = uint32(binary.LittleEndian.Uint16(rest[:2]))
		}
		gate := rest[width]
		offset := width + 1
		var inlineName string
		if gate&1 != 0 {
			length, tail, valid := readVar(rest[offset:])
			if !valid || length == 0 || length > 128 || uint32(len(tail)) < length {
				continue
			}
			nameBytes := tail[:length]
			if !utf8.Valid(nameBytes) {
				continue
			}
			inlineName = string(nameBytes)
			offset = len(rest) - len(tail) + int(length)
		}
		if offset+4+2+12 > len(rest) {
			continue
		}
		code := binary.LittleEndian.Uint32(rest[offset : offset+4])
		if code == 0 || code > 99_999_999 {
			continue
		}
		// In 0x3640 / 0x3641, the 2 bytes after NPC code are state/subtype (e.g. 0x40 0x02), followed by 3x float32 LE (X, Y, Z).
		pos, valid := readVec3LE(rest[offset+6 : offset+18])
		if !valid {
			continue
		}
		e.radar.upsertEntity(t, game.Entity(id), code, mask, inlineName, pos)
		return
	}
}

func computeDirection(selfPos game.Pos, hasSelfPos bool, hx, hy float32, hasHeading bool, targetPos game.Pos) (distMeters, zDeltaMeters, angleDeg float64, horizDir, vertDir string, hasDir bool) {
	if !hasSelfPos {
		return 0, 0, 0, "Frente", "Nivel", false
	}
	dx := float64(targetPos.X - selfPos.X)
	dy := float64(targetPos.Y - selfPos.Y)
	dz := float64(targetPos.Z - selfPos.Z)

	// 100 Unreal units = 1 meter
	dist2D := math.Hypot(dx, dy) / 100.0
	zDeltaMeters = dz / 100.0
	distMeters = math.Hypot(dist2D, zDeltaMeters)

	if zDeltaMeters >= 3.0 {
		vertDir = "Arriba"
	} else if zDeltaMeters <= -3.0 {
		vertDir = "Abajo"
	} else {
		vertDir = "Nivel"
	}

	if dist2D < 0.2 {
		return distMeters, zDeltaMeters, 0, "Aquí", vertDir, true
	}

	// Default reference vector is North/+X if player hasn't moved yet; otherwise relative to player's movement/cast heading.
	fx, fy := 1.0, 0.0
	if hasHeading {
		fx, fy = float64(hx), float64(hy)
	}
	dot := fx*dx + fy*dy
	cross := fx*dy - fy*dx
	angleDeg = math.Atan2(cross, dot) * 180.0 / math.Pi

	switch {
	case angleDeg >= -45 && angleDeg <= 45:
		horizDir = "Frente"
	case angleDeg > 45 && angleDeg < 135:
		horizDir = "Der"
	case angleDeg < -45 && angleDeg > -135:
		horizDir = "Izq"
	default:
		horizDir = "Atrás"
	}
	return distMeters, zDeltaMeters, angleDeg, horizDir, vertDir, hasHeading
}

func (e *encounter) radarSnapshot() []radarDisplay {
	if len(e.radar.entities) == 0 {
		return nil
	}
	out := make([]radarDisplay, 0, len(e.radar.entities))
	for _, ent := range e.radar.entities {
		dist, dz, angle, hDir, vDir, hasDir := computeDirection(
			e.radar.selfPos,
			e.radar.hasSelfPos,
			e.radar.headingX,
			e.radar.headingY,
			e.radar.hasHeading,
			ent.pos,
		)
		// Filter out stale objects more than 500m away
		if e.radar.hasSelfPos && dist > 500 {
			continue
		}
		out = append(out, radarDisplay{
			ID:       ent.id,
			NPC:      ent.npc,
			Name:     ent.name,
			Category: ent.category,
			Distance: math.Round(dist*10) / 10,
			ZDelta:   math.Round(dz*10) / 10,
			Angle:    math.Round(angle),
			HorizDir: hDir,
			VertDir:  vDir,
			HasDir:   hasDir,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		// Prioritize known categories (chest, odyle, resource) over generic objects, then by distance
		iKnown := out[i].Category != radarCategoryObject
		jKnown := out[j].Category != radarCategoryObject
		if iKnown != jKnown {
			return iKnown
		}
		if out[i].Distance == out[j].Distance {
			return out[i].ID < out[j].ID
		}
		return out[i].Distance < out[j].Distance
	})
	if len(out) > 32 {
		out = out[:32]
	}
	return out
}
