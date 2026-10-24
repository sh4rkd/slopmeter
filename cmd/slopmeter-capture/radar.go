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
	lastDist   float64 // distance at the last trend sample (m), 0 = not sampled yet
	trend      int     // +1 getting closer, -1 getting farther, 0 unknown
}

type radarState struct {
	entities   map[game.Entity]*radarEntity
	selfPos    game.Pos
	hasSelfPos bool
	headingX   float32
	headingY   float32
	hasHeading bool
	// anchorPos is the last position used to sample the walking direction.
	// The heading is only re-sampled after moving headingSampleCm away from it,
	// so small jitter, strafing steps or standing still never make the arrow spin.
	anchorPos game.Pos
	hasAnchor bool
	revision  uint64
}

const (
	// headingSampleCm is the minimum travelled distance before the walking direction is re-sampled (1.5 m).
	headingSampleCm = 150.0
	// headingSmoothing is the weight of a new heading sample (exponential smoothing of the direction vector).
	headingSmoothing = 0.45
	// headingTeleportCm resets the heading anchor on teleports/zone jumps instead of producing a bogus direction.
	headingTeleportCm = 3000.0
)

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
	// RelAngle is the bearing to the target relative to the character's walking
	// direction: 0 = straight ahead, +90 = right, -90 = left, 180 = behind.
	RelAngle   float64 `json:"relAngle"`
	HasHeading bool    `json:"hasHeading"`
	// Trend: +1 getting closer, -1 getting farther, 0 steady/unknown.
	Trend int `json:"trend"`
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

	// Interactive Supply / Field Chests
	2900100: {2900100, "Supply Chest", radarCategoryChest},
	2900101: {2900101, "Supply Chest", radarCategoryChest},
	2900102: {2900102, "Supply Chest", radarCategoryChest},
	2900200: {2900200, "Supply Chest", radarCategoryChest},
	2900201: {2900201, "Treasure Chest", radarCategoryChest},
	2920100: {2920100, "Supply Chest", radarCategoryChest},

	// Odyle / Odylium / Crystals (GatherSource & World Entities)
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
	2000791: {2000791, "Crystal", radarCategoryOdyle},
	2000871: {2000871, "Crystal", radarCategoryOdyle},
	2301319: {2301319, "Crystal", radarCategoryOdyle},
	2311319: {2311319, "Crystal", radarCategoryOdyle},
	2321319: {2321319, "Crystal", radarCategoryOdyle},
	2921332: {2921332, "Atheron's Crystal", radarCategoryOdyle},
	2921333: {2921333, "Atheron's Crystal", radarCategoryOdyle},

	// Gems (Gemas): Sapphire, Ruby, Diamond, Tourmaline, Veil Jewel
	1080501: {1080501, "Sapphire Gemstone", radarCategoryGem},
	1080504: {1080504, "Fine Sapphire Gemstone", radarCategoryGem},
	1080507: {1080507, "Pure Sapphire Gemstone", radarCategoryGem},
	1080510: {1080510, "Radiant Sapphire Gemstone", radarCategoryGem},
	1090501: {1090501, "Sapphire Gemstone", radarCategoryGem},
	1090504: {1090504, "Fine Sapphire Gemstone", radarCategoryGem},
	1090507: {1090507, "Pure Sapphire Gemstone", radarCategoryGem},
	1090510: {1090510, "Radiant Sapphire Gemstone", radarCategoryGem},
	1080502: {1080502, "Diamond Gemstone", radarCategoryGem},
	1080505: {1080505, "Fine Diamond Gemstone", radarCategoryGem},
	1080508: {1080508, "Pure Diamond Gemstone", radarCategoryGem},
	1080511: {1080511, "Radiant Diamond Gemstone", radarCategoryGem},
	1090502: {1090502, "Diamond Gemstone", radarCategoryGem},
	1090505: {1090505, "Fine Diamond Gemstone", radarCategoryGem},
	1090508: {1090508, "Pure Diamond Gemstone", radarCategoryGem},
	1090511: {1090511, "Radiant Diamond Gemstone", radarCategoryGem},
	1080503: {1080503, "Ruby Gemstone", radarCategoryGem},
	1080506: {1080506, "Fine Ruby Gemstone", radarCategoryGem},
	1080509: {1080509, "Pure Ruby Gemstone", radarCategoryGem},
	1080512: {1080512, "Radiant Ruby Gemstone", radarCategoryGem},
	1090503: {1090503, "Ruby Gemstone", radarCategoryGem},
	1090506: {1090506, "Fine Ruby Gemstone", radarCategoryGem},
	1090509: {1090509, "Pure Ruby Gemstone", radarCategoryGem},
	1090512: {1090512, "Radiant Ruby Gemstone", radarCategoryGem},
	2300254: {2300254, "Floating Tourmaline", radarCategoryGem},
	2300255: {2300255, "Floating Tourmaline", radarCategoryGem},
	2300264: {2300264, "Floating Tourmaline", radarCategoryGem},
	2300817: {2300817, "Floating Tourmaline", radarCategoryGem},
	2300818: {2300818, "Floating Tourmaline", radarCategoryGem},
	2590112: {2590112, "Veil Jewel", radarCategoryGem},
	2590224: {2590224, "Veil Jewel", radarCategoryGem},

	// Ores (Minerales): Orichalcum Ore
	1080101: {1080101, "Orichalcum Ore", radarCategoryOre},
	1080102: {1080102, "Fine Orichalcum Ore", radarCategoryOre},
	1080103: {1080103, "Pure Orichalcum Ore", radarCategoryOre},
	1080104: {1080104, "Radiant Orichalcum Ore", radarCategoryOre},
	1080401: {1080401, "Orichalcum Ore", radarCategoryOre},
	1080402: {1080402, "Fine Orichalcum Ore", radarCategoryOre},
	1080403: {1080403, "Pure Orichalcum Ore", radarCategoryOre},
	1080404: {1080404, "Radiant Orichalcum Ore", radarCategoryOre},
	1090101: {1090101, "Orichalcum Ore", radarCategoryOre},
	1090102: {1090102, "Fine Orichalcum Ore", radarCategoryOre},
	1090103: {1090103, "Pure Orichalcum Ore", radarCategoryOre},
	1090104: {1090104, "Radiant Orichalcum Ore", radarCategoryOre},
	1090401: {1090401, "Orichalcum Ore", radarCategoryOre},
	1090402: {1090402, "Fine Orichalcum Ore", radarCategoryOre},
	1090403: {1090403, "Pure Orichalcum Ore", radarCategoryOre},
	1090404: {1090404, "Radiant Orichalcum Ore", radarCategoryOre},

	// Wood / Logs (Madera): Asvata Log, Yggdrasil Log
	1080201: {1080201, "Yggdrasil Log", radarCategoryWood},
	1080202: {1080202, "Fine Yggdrasil Log", radarCategoryWood},
	1080203: {1080203, "Pure Yggdrasil Log", radarCategoryWood},
	1080204: {1080204, "Radiant Yggdrasil Log", radarCategoryWood},
	1080301: {1080301, "Yggdrasil Log", radarCategoryWood},
	1080302: {1080302, "Fine Yggdrasil Log", radarCategoryWood},
	1080303: {1080303, "Pure Yggdrasil Log", radarCategoryWood},
	1080304: {1080304, "Radiant Yggdrasil Log", radarCategoryWood},
	1090201: {1090201, "Asvata Log", radarCategoryWood},
	1090202: {1090202, "Fine Asvata Log", radarCategoryWood},
	1090203: {1090203, "Pure Asvata Log", radarCategoryWood},
	1090204: {1090204, "Radiant Asvata Log", radarCategoryWood},
	1090301: {1090301, "Asvata Log", radarCategoryWood},
	1090302: {1090302, "Fine Asvata Log", radarCategoryWood},
	1090303: {1090303, "Pure Asvata Log", radarCategoryWood},
	1090304: {1090304, "Radiant Asvata Log", radarCategoryWood},

	// Herbs & Plants (Hierbas / Flores / Ingredientes): Azpha, Calendula, Aria, Targena, etc.
	1080601: {1080601, "Aria", radarCategoryHerb},
	1080603: {1080603, "Fresh Aria", radarCategoryHerb},
	1080605: {1080605, "Aria", radarCategoryHerb},
	1080607: {1080607, "Fresh Aria", radarCategoryHerb},
	1080602: {1080602, "Targena", radarCategoryHerb},
	1080604: {1080604, "Targena", radarCategoryHerb},
	1080606: {1080606, "Fresh Targena", radarCategoryHerb},
	1080608: {1080608, "Fresh Targena", radarCategoryHerb},
	1090601: {1090601, "Azpha", radarCategoryHerb},
	1090603: {1090603, "Fresh Azpha", radarCategoryHerb},
	1090605: {1090605, "Azpha", radarCategoryHerb},
	1090607: {1090607, "Fresh Azpha", radarCategoryHerb},
	1090602: {1090602, "Calendula", radarCategoryHerb},
	1090604: {1090604, "Fresh Calendula", radarCategoryHerb},
	1090606: {1090606, "Calendula", radarCategoryHerb},
	1090608: {1090608, "Fresh Calendula", radarCategoryHerb},
	1081001: {1081001, "Mela", radarCategoryHerb},
	1081007: {1081007, "Mela", radarCategoryHerb},
	1081013: {1081013, "Mela", radarCategoryHerb},
	1081002: {1081002, "Coriolus", radarCategoryHerb},
	1081008: {1081008, "Coriolus", radarCategoryHerb},
	1081014: {1081014, "Coriolus", radarCategoryHerb},
	1081003: {1081003, "Inina", radarCategoryHerb},
	1081009: {1081009, "Inina", radarCategoryHerb},
	1081015: {1081015, "Inina", radarCategoryHerb},
	1081004: {1081004, "Cypri", radarCategoryHerb},
	1081010: {1081010, "Cypri", radarCategoryHerb},
	1081016: {1081016, "Cypri", radarCategoryHerb},
	1081019: {1081019, "Cypri", radarCategoryHerb},
	1081006: {1081006, "Kukuru", radarCategoryHerb},
	1081012: {1081012, "Kukuru", radarCategoryHerb},
	1081018: {1081018, "Kukuru", radarCategoryHerb},
	1081021: {1081021, "Kukuru", radarCategoryHerb},
	1091001: {1091001, "Raydam", radarCategoryHerb},
	1091007: {1091007, "Raydam", radarCategoryHerb},
	1091013: {1091013, "Raydam", radarCategoryHerb},
	1091002: {1091002, "Morfa", radarCategoryHerb},
	1091008: {1091008, "Morfa", radarCategoryHerb},
	1091014: {1091014, "Morfa", radarCategoryHerb},
	1091003: {1091003, "Conide", radarCategoryHerb},
	1091009: {1091009, "Conide", radarCategoryHerb},
	1091015: {1091015, "Conide", radarCategoryHerb},
	1091004: {1091004, "Pujery", radarCategoryHerb},
	1091010: {1091010, "Pujery", radarCategoryHerb},
	1091016: {1091016, "Pujery", radarCategoryHerb},
	1091019: {1091019, "Pujery", radarCategoryHerb},
	1091006: {1091006, "Ohkra", radarCategoryHerb},
	1091012: {1091012, "Ohkra", radarCategoryHerb},
	1091018: {1091018, "Ohkra", radarCategoryHerb},
	1091021: {1091021, "Ohkra", radarCategoryHerb},
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
	// 1. Direct catalog lookup for confirmed Cubes, Chests and GatherSource resources
	if def, ok := radarCatalog[code]; ok {
		name := def.Name
		if inlineName != "" {
			name = inlineName
		}
		return name, def.Category, true
	}
	// 2. Exact keyword match in inlineName for Cubes and Gathering Nodes
	lower := strings.ToLower(inlineName)
	if lower != "" {
		switch {
		case strings.Contains(lower, "cube") || strings.Contains(lower, "cubo") ||
			strings.Contains(lower, "hidden") || strings.Contains(lower, "oculto") ||
			strings.Contains(lower, "chest") || strings.Contains(lower, "supply") ||
			strings.Contains(lower, "cofre") || strings.Contains(lower, "suministro") ||
			strings.Contains(lower, "treasure") || strings.Contains(lower, "tesoro") ||
			strings.Contains(lower, "relic") || strings.Contains(lower, "box") ||
			strings.Contains(lower, "caja") || strings.Contains(lower, "큐브") ||
			strings.Contains(lower, "상자") || strings.Contains(lower, "보급"):
			return inlineName, radarCategoryChest, true

		case strings.Contains(lower, "orichalcum") || strings.Contains(lower, "ore") ||
			strings.Contains(lower, "vein") || strings.Contains(lower, "mineral") ||
			strings.Contains(lower, "oricalco") || strings.Contains(lower, "광석"):
			return inlineName, radarCategoryOre, true

		case strings.Contains(lower, "herb") || strings.Contains(lower, "azpha") ||
			strings.Contains(lower, "calendula") || strings.Contains(lower, "aria") ||
			strings.Contains(lower, "targena") || strings.Contains(lower, "pujery") ||
			strings.Contains(lower, "conide") || strings.Contains(lower, "ohkra") ||
			strings.Contains(lower, "morfa") || strings.Contains(lower, "raydam") ||
			strings.Contains(lower, "mela") || strings.Contains(lower, "coriolus") ||
			strings.Contains(lower, "inina") || strings.Contains(lower, "cypri") ||
			strings.Contains(lower, "kukuru") || strings.Contains(lower, "hierba") ||
			strings.Contains(lower, "plant") || strings.Contains(lower, "약초"):
			return inlineName, radarCategoryHerb, true

		case strings.Contains(lower, "sapphire") || strings.Contains(lower, "ruby") ||
			strings.Contains(lower, "diamond") || strings.Contains(lower, "gem") ||
			strings.Contains(lower, "tourmaline") || strings.Contains(lower, "jewel") ||
			strings.Contains(lower, "zafiro") || strings.Contains(lower, "rubí") ||
			strings.Contains(lower, "diamante") || strings.Contains(lower, "gema") ||
			strings.Contains(lower, "보석"):
			return inlineName, radarCategoryGem, true

		case strings.Contains(lower, "log") || strings.Contains(lower, "asvata") ||
			strings.Contains(lower, "yggdrasil") || strings.Contains(lower, "tree") ||
			strings.Contains(lower, "wood") || strings.Contains(lower, "tronco") ||
			strings.Contains(lower, "madera") || strings.Contains(lower, "목재"):
			return inlineName, radarCategoryWood, true

		case strings.Contains(lower, "odyle") || strings.Contains(lower, "odylium") ||
			strings.Contains(lower, "odil") || strings.Contains(lower, "aether") ||
			strings.Contains(lower, "오드"):
			return inlineName, radarCategoryOdyle, true
		}
	}
	// Any non-catalog NPC, monster or decorative object is ignored to prevent clutter
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
	r.sampleHeading(pos)
	if !r.hasSelfPos || math.Abs(float64(pos.X-r.selfPos.X)) > 5 || math.Abs(float64(pos.Y-r.selfPos.Y)) > 5 || math.Abs(float64(pos.Z-r.selfPos.Z)) > 5 {
		r.selfPos = pos
		r.hasSelfPos = true
		r.revision++
	}
}

// sampleHeading derives the character's walking direction from its own track.
// Aion 2 does not send the camera yaw, so like the community cube radars we use the
// movement vector; unlike a per-packet delta it is sampled over >= 1.5 m and smoothed,
// which keeps the navigation arrow steady while walking.
func (r *radarState) sampleHeading(pos game.Pos) {
	if !r.hasAnchor {
		r.anchorPos, r.hasAnchor = pos, true
		return
	}
	dx := float64(pos.X - r.anchorPos.X)
	dy := float64(pos.Y - r.anchorPos.Y)
	dist := math.Hypot(dx, dy)
	if dist > headingTeleportCm {
		r.anchorPos = pos
		return
	}
	if dist < headingSampleCm {
		return
	}
	nx, ny := dx/dist, dy/dist
	if r.hasHeading {
		hx := (1-headingSmoothing)*float64(r.headingX) + headingSmoothing*nx
		hy := (1-headingSmoothing)*float64(r.headingY) + headingSmoothing*ny
		if l := math.Hypot(hx, hy); l > 1e-6 {
			nx, ny = hx/l, hy/l
		}
	}
	r.headingX, r.headingY, r.hasHeading = float32(nx), float32(ny), true
	r.anchorPos = pos
	r.revision++
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
		e.radar.hasAnchor, e.radar.hasHeading = false, false
		e.radar.updateSelfPos(ev.Pos)
	case game.Move:
		if self != 0 && ev.Entity == self {
			e.radar.updateSelfPos(ev.Pos)
		} else if self == 0 && !e.radar.hasSelfPos && (e.radar.entities == nil || e.radar.entities[ev.Entity] == nil) && validWorldPos(ev.Pos) {
			e.radar.updateSelfPos(ev.Pos)
		}
	case game.Cast:
		if (self != 0 && ev.Actor == self) || (self == 0 && !e.radar.hasSelfPos && validWorldPos(ev.Pos)) {
			// In game.Cast, Pos is the skill target's position: it must not move the
			// player nor turn the walking heading (that made the arrow jump in combat).
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

	// Own position from self-buff records (0x382A / 0x382B). Layout verified by
	// Cube Watch / ayi-afk / dr89mh: body starts with the caster varint and ends with
	// `caster varint | 01 00 (2A) or 01 02 (2B) | x y z f32`. The same opcodes are
	// also sent for other players, so the leading and trailing ids must match and
	// be our own entity; otherwise the radar jumped to other players' positions.
	if m.opcode == 0x382A || m.opcode == 0x382B {
		if pos, id, ok := parseSelfBuffPos(m.payload, m.opcode == 0x382B); ok && (self == 0 || game.Entity(id) == self) {
			e.radar.updateSelfPos(pos)
		}
	}

	// Map load (0x3621): `u32 count | u32 map | u32 | u32 | x y z | heading` carries our position.
	if m.opcode == 0x3621 && len(m.payload) >= 28 {
		if pos, valid := readVec3LE(m.payload[16:28]); valid {
			e.radar.hasAnchor, e.radar.hasHeading = false, false
			e.radar.updateSelfPos(pos)
		}
	}

	// Generic entity position (0x371A, server side): `id varint | 2 bytes | x y z`.
	if m.opcode == 0x371A && self != 0 {
		if id, rest, ok := readVar(m.payload); ok && game.Entity(id) == self && len(rest) >= 14 {
			if pos, valid := readVec3LE(rest[2:14]); valid {
				e.radar.updateSelfPos(pos)
			}
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

// parseSelfBuffPos decodes the trailing caster position of a 0x382A/0x382B record.
func parseSelfBuffPos(p []byte, is2B bool) (game.Pos, uint32, bool) {
	id, rest, ok := readVar(p)
	if !ok || id == 0 || len(p) < 16 {
		return game.Pos{}, 0, false
	}
	width := len(p) - len(rest)
	start := len(p) - 14 - width
	if start < width {
		return game.Pos{}, 0, false
	}
	tailID, tailRest, ok := readVar(p[start : len(p)-12])
	if !ok || tailID != id || len(tailRest) != 2 || tailRest[0] != 0x01 {
		return game.Pos{}, 0, false
	}
	if (is2B && tailRest[1] != 0x02) || (!is2B && tailRest[1] != 0x00) {
		return game.Pos{}, 0, false
	}
	pos, valid := readVec3LE(p[len(p)-12:])
	return pos, id, valid
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
		return 0, 0, 0, 0, 0, "Norte", "Nivel", false
	}
	// Aion 2 world axes as measured on the in-game minimap by three independent
	// cube radars (Cube Watch, dr89mh/aion2-cube-radar Geo.cs, ayi-afk/aion2cubedetector
	// compass.v): map North = world -Y, map East = world +X, +Z = up.
	dx := float64(targetPos.X - selfPos.X) // East(+)/West(-) offset in Unreal units
	dy := float64(targetPos.Y - selfPos.Y) // South(+)/North(-) offset in Unreal units
	dz := float64(targetPos.Z - selfPos.Z) // Height offset in Unreal units

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

	// World Compass / Radar Projection (aligned with the in-game minimap):
	// - North (-Y) is Top of map (+relY)
	// - East (+X) is Right of map (+relX)
	relY = -dy / 100.0
	relX = dx / 100.0
	angleDeg = math.Atan2(relX, relY) * 180.0 / math.Pi

	// Cardinal and 8-way directional mapping
	switch {
	case angleDeg >= -22.5 && angleDeg <= 22.5:
		horizDir = "Norte"
	case angleDeg > 22.5 && angleDeg <= 67.5:
		horizDir = "Noreste"
	case angleDeg > 67.5 && angleDeg <= 112.5:
		horizDir = "Este"
	case angleDeg > 112.5 && angleDeg <= 157.5:
		horizDir = "Sureste"
	case angleDeg < -22.5 && angleDeg >= -67.5:
		horizDir = "Noroeste"
	case angleDeg < -67.5 && angleDeg >= -112.5:
		horizDir = "Oeste"
	case angleDeg < -112.5 && angleDeg >= -157.5:
		horizDir = "Suroeste"
	default:
		horizDir = "Sur"
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
		relAngle := 0.0
		if e.radar.hasHeading && hasDir {
			// Same compass frame as computeDirection: bearing = atan2(east=+X, north=-Y).
			headingDeg := math.Atan2(float64(e.radar.headingX), -float64(e.radar.headingY)) * 180.0 / math.Pi
			relAngle = math.Mod(angle-headingDeg+540.0, 360.0) - 180.0
		}
		// Hotter/colder trend with 0.5 m hysteresis so it does not flicker.
		if ent.lastDist == 0 {
			ent.lastDist = dist
		} else if d := dist - ent.lastDist; math.Abs(d) >= 0.5 {
			if d < 0 {
				ent.trend = 1
			} else {
				ent.trend = -1
			}
			ent.lastDist = dist
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
			RelAngle:   math.Round(relAngle),
			HasHeading: e.radar.hasHeading && hasDir,
			Trend:      ent.trend,
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
