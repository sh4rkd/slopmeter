package main

import (
	"bytes"
	"encoding/binary"
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
	radarCategoryGem      radarCategory = "gem"
	radarCategoryOre      radarCategory = "ore"
	radarCategoryHerb     radarCategory = "herb"
	radarCategoryWood     radarCategory = "wood"
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
	RelX     float64       `json:"relX"`
	RelY     float64       `json:"relY"`
	ZDelta   float64       `json:"zDelta"`
	Angle    float64       `json:"angle"`
	HorizDir string        `json:"horizDir"`
	VertDir  string        `json:"vertDir"`
	HasDir   bool          `json:"hasDir"`
}

var radarCatalog = map[uint32]radarDefinition{
	// Hidden Cubes (Cubos Ocultos) - Verified IDs from Aion 2 game client / benhighlanderr tracker
	1000295: {1000295, "Hidden Cube", radarCategoryChest},
	1000548: {1000548, "Hidden Cube", radarCategoryChest},
	1000556: {1000556, "Hidden Cube", radarCategoryChest},
	1000568: {1000568, "Hidden Cube", radarCategoryChest},
	1000566: {1000566, "Hidden Cube V04", radarCategoryChest},
	1000567: {1000567, "Hidden Cube", radarCategoryChest},
	1100056: {1100056, "Hidden Cube", radarCategoryChest},
	1100094: {1100094, "Hidden Cube", radarCategoryChest},
	1200018: {1200018, "Hidden Cube", radarCategoryChest},
	1200036: {1200036, "Hidden Cube", radarCategoryChest},
	1200037: {1200037, "Hidden Cube", radarCategoryChest},
	1200039: {1200039, "Hidden Cube", radarCategoryChest},
	1200040: {1200040, "Hidden Cube V04", radarCategoryChest},
	1200041: {1200041, "Hidden Cube V04 (Water)", radarCategoryChest},
	1200667: {1200667, "Hidden Cube", radarCategoryChest},
	1200707: {1200707, "Hidden Cube", radarCategoryChest},
	1201054: {1201054, "Hidden Cube", radarCategoryChest},
	1201055: {1201055, "Hidden Cube", radarCategoryChest},
	1300081: {1300081, "Asmodian Hidden Cube", radarCategoryChest},
	1300082: {1300082, "Asmodian Hidden Cube", radarCategoryChest},
	1300754: {1300754, "Asmodian Hidden Cube", radarCategoryChest},
	1301199: {1301199, "Asmodian Hidden Cube V04", radarCategoryChest},
	1301200: {1301200, "Asmodian Hidden Cube V04 (Water)", radarCategoryChest},
	1401335: {1401335, "Hidden Cube", radarCategoryChest},
	1401336: {1401336, "Hidden Cube", radarCategoryChest},
	3682560: {3682560, "Special Hidden Cube", radarCategoryChest},
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
	// 1. Direct catalog lookup for confirmed Hidden Cubes
	if def, ok := radarCatalog[code]; ok {
		name := def.Name
		if inlineName != "" {
			name = inlineName
		}
		return name, def.Category, true
	}
	// 2. Exact keyword match in inlineName for Hidden Cubes / Chests
	lower := strings.ToLower(inlineName)
	if lower != "" {
		if strings.Contains(lower, "cube") || strings.Contains(lower, "cubo") ||
			strings.Contains(lower, "hidden") || strings.Contains(lower, "oculto") ||
			strings.Contains(lower, "chest") || strings.Contains(lower, "supply") ||
			strings.Contains(lower, "cofre") || strings.Contains(lower, "suministro") ||
			strings.Contains(lower, "treasure") || strings.Contains(lower, "tesoro") ||
			strings.Contains(lower, "relic") || strings.Contains(lower, "box") ||
			strings.Contains(lower, "caja") || strings.Contains(lower, "큐브") ||
			strings.Contains(lower, "상자") || strings.Contains(lower, "보급") {
			return inlineName, radarCategoryChest, true
		}
	}
	// Only cubes/chests are tracked on radar. Ignore all resources and background objects.
	return "", "", false
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
	if len(r.entities) > 100 && r.hasSelfPos {
		// Prune entities that are farther than 350m away (35,000 Unreal cm)
		maxDistSq := float64(35000.0 * 35000.0)
		for k, v := range r.entities {
			dx := float64(v.pos.X - r.selfPos.X)
			dy := float64(v.pos.Y - r.selfPos.Y)
			if dx*dx+dy*dy > maxDistSq {
				delete(r.entities, k)
			}
		}
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

func (r *radarState) removeNearbyEntities(radiusCm float32) {
	if r.entities == nil || !r.hasSelfPos || radiusCm <= 0 {
		return
	}
	rSq := float64(radiusCm * radiusCm)
	changed := false
	for id, ent := range r.entities {
		dx := float64(ent.pos.X - r.selfPos.X)
		dy := float64(ent.pos.Y - r.selfPos.Y)
		dz := float64(ent.pos.Z - r.selfPos.Z)
		if dx*dx+dy*dy+dz*dz <= rSq {
			delete(r.entities, id)
			changed = true
		}
	}
	if changed {
		r.revision++
	}
}

var selfPosSentinel = []byte{0x80, 0x75, 0xd5, 0x2a, 0xbb, 0x03, 0x00, 0x00}

func (e *encounter) observeRadar(m aMessage, self game.Entity) {
	if m.flags&wire.FromServer == 0 {
		// Packets sent by the local client to the server are from the local player
		if mv, ok := m.event.(game.Move); ok && validWorldPos(mv.Pos) {
			e.radar.updateSelfPos(mv.Pos)
		} else if c, ok := m.event.(game.Cast); ok && validWorldPos(c.Pos) {
			e.radar.updateSelfPos(c.Pos)
		} else if m.opcode == 0x371A || m.opcode == 0x371B || m.opcode == 0x371C {
			if id, rest, ok := readVar(m.payload); ok && id != 0 {
				for skip := 0; skip <= 4 && skip+12 <= len(rest); skip++ {
					if pos, valid := readVec3LE(rest[skip : skip+12]); valid {
						e.radar.updateSelfPos(pos)
						break
					}
				}
			}
		}
		return
	}
	switch ev := m.event.(type) {
	case game.Zone:
		e.radar.clear()
		e.radar.updateSelfPos(ev.Pos)
	case game.Move:
		if self != 0 && ev.Entity == self {
			e.radar.updateSelfPos(ev.Pos)
		} else if self == 0 && !e.radar.hasSelfPos && (e.radar.entities == nil || e.radar.entities[ev.Entity] == nil) && validWorldPos(ev.Pos) {
			e.radar.updateSelfPos(ev.Pos)
		} else if e.radar.entities != nil {
			if ent, ok := e.radar.entities[ev.Entity]; ok && validWorldPos(ev.Pos) {
				ent.pos = ev.Pos
				ent.updated = m.t
				e.radar.revision++
			}
		}
	case game.Cast:
		if (self != 0 && ev.Actor == self) || (self == 0 && !e.radar.hasSelfPos && validWorldPos(ev.Pos)) {
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

	// World object / Hidden Cube spawn (0x3634 - wire "34 36")
	if m.opcode == 0x3634 {
		e.observeRadarObjectSpawn(m.t, m.payload)
	}

	// World object removed / opened / despawned (0x3635, 0x363A, 0x3642)
	if m.opcode == 0x3635 || m.opcode == 0x363A || m.opcode == 0x3642 {
		if id, _, ok := readVar(m.payload); ok && id != 0 {
			e.radar.removeEntity(game.Entity(id))
		}
	}

	// Interaction / loot collection nearby (0x8D6D / 36205) removes nearby gathered/opened items within 6m
	if m.opcode == 0x8D6D {
		e.radar.removeNearbyEntities(600.0)
	}

	// Continuous player position update from server buff/stat records (0x382A / 0x382B)
	if (m.opcode == 0x382A || m.opcode == 0x382B) && len(m.payload) >= 16 {
		if pos, valid := readVec3LE(m.payload[len(m.payload)-12:]); valid {
			e.radar.updateSelfPos(pos)
		}
	}

	// In-map teleport position fix (0x3623)
	if m.opcode == 0x3623 && len(m.payload) >= 18 {
		if pos, valid := readVec3LE(m.payload[6:18]); valid {
			e.radar.updateSelfPos(pos)
		}
	}

	// Direct cube/actor spawn opcode 0x3641 (13889)
	if m.opcode == 0x3641 {
		if id, rest, ok := readVar(m.payload); ok && id != 0 && len(rest) >= 5+18 {
			offset := 5
			npcID := binary.LittleEndian.Uint32(rest[offset : offset+4])
			if def, found := radarCatalog[npcID]; found && def.Category == radarCategoryChest {
				if pos, valid := readVec3LE(rest[offset+6 : offset+18]); valid {
					e.radar.upsertEntity(m.t, game.Entity(id), npcID, 0, def.Name, pos)
				}
			}
		}
	}

	if m.opcode == 0x3640 || m.opcode == 0x3641 || m.opcode == 0x3646 || m.opcode == 0x3647 || m.opcode == 0x3649 || m.opcode == 0x364A {
		e.observeRadarSpawn(m.t, m.opcode, m.payload)
	}
}

func (e *encounter) observeRadarObjectSpawn(t time.Time, payload []byte) {
	id, rest, ok := readVar(payload)
	if !ok || id == 0 || len(rest) < 2+4+12 {
		return
	}
	kind := binary.LittleEndian.Uint16(rest[0:2])
	template := binary.LittleEndian.Uint32(rest[2:6])
	if template == 0 || template > 99_999_999 {
		return
	}
	pos, valid := readVec3LE(rest[6:18])
	if !valid {
		return
	}
	e.radar.upsertEntity(t, game.Entity(id), template, uint32(kind), "", pos)
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
			nameLen := int(rest[offset])
			if nameLen == 0 {
				offset++
			} else if nameLen > 0 && nameLen <= 128 && offset+1+nameLen+4+12 <= len(rest) {
				candidate := rest[offset+1 : offset+1+nameLen]
				if utf8.Valid(candidate) {
					inlineName = string(candidate)
					offset = offset + 1 + nameLen
				}
			} else {
				length, tail, valid := readVar(rest[offset:])
				if valid && length <= 128 && uint32(len(tail)) >= length {
					if length > 0 {
						nameBytes := tail[:length]
						if utf8.Valid(nameBytes) {
							inlineName = string(nameBytes)
						}
					}
					offset = len(rest) - len(tail) + int(length)
				}
			}
		}
		if offset+4+12 > len(rest) {
			continue
		}
		code := binary.LittleEndian.Uint32(rest[offset : offset+4])
		if code == 0 || code > 99_999_999 {
			continue
		}
		var foundPos game.Pos
		var hasPos bool
		// Standard subpacket layout has 2B subtype then 12B pos at offset+6 (skip=2).
		// Try skip=2 first, then alternatives (0, 4, 6, 8).
		for _, skip := range []int{2, 0, 4, 6, 8} {
			if offset+4+skip+12 <= len(rest) {
				if pos, valid := readVec3LE(rest[offset+4+skip : offset+4+skip+12]); valid {
					foundPos = pos
					hasPos = true
					break
				}
			}
		}
		if !hasPos {
			continue
		}
		e.radar.upsertEntity(t, game.Entity(id), code, mask, inlineName, foundPos)
		return
	}
}

func computeDirection(selfPos game.Pos, hasSelfPos bool, hx, hy float32, hasHeading bool, targetPos game.Pos) (distMeters, relX, relY, zDeltaMeters, angleDeg float64, horizDir, vertDir string, hasDir bool) {
	if !hasSelfPos {
		return 0, 0, 0, 0, 0, "Frente", "Nivel", false
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
		return distMeters, 0, 0, zDeltaMeters, 0, "Aquí", vertDir, true
	}

	// Default reference vector is North/+X if player hasn't moved yet; otherwise relative to player's movement/cast heading.
	fx, fy := 1.0, 0.0
	if hasHeading {
		fx, fy = float64(hx), float64(hy)
	}
	// Forward/Back component in meters (+relY = Front/Up on radar, -relY = Back/Down on radar)
	relY = (fx*dx + fy*dy) / 100.0
	// Right/Left component in meters (+relX = Right on radar, -relX = Left on radar)
	relX = (fx*dy - fy*dx) / 100.0
	angleDeg = math.Atan2(relX, relY) * 180.0 / math.Pi

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
	return distMeters, relX, relY, zDeltaMeters, angleDeg, horizDir, vertDir, true
}

func (e *encounter) radarSnapshot() []radarDisplay {
	if len(e.radar.entities) == 0 {
		return nil
	}
	out := make([]radarDisplay, 0, len(e.radar.entities))
	for _, ent := range e.radar.entities {
		dist, rx, ry, dz, angle, hDir, vDir, hasDir := computeDirection(
			e.radar.selfPos,
			e.radar.hasSelfPos,
			e.radar.headingX,
			e.radar.headingY,
			e.radar.hasHeading,
			ent.pos,
		)
		if e.radar.hasSelfPos && dist > 500 {
			continue
		}
		out = append(out, radarDisplay{
			ID:       ent.id,
			NPC:      ent.npc,
			Name:     ent.name,
			Category: ent.category,
			Distance: math.Round(dist*10) / 10,
			RelX:     math.Round(rx*10) / 10,
			RelY:     math.Round(ry*10) / 10,
			ZDelta:   math.Round(dz*10) / 10,
			Angle:    math.Round(angle),
			HorizDir: hDir,
			VertDir:  vDir,
			HasDir:   hasDir,
		})
	}
	sort.Slice(out, func(i, j int) bool {
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
