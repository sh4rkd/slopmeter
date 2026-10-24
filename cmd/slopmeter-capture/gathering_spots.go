package main

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"math"
	"sync"

	"github.com/nuriland/a2kit/game"
)

//go:embed data/gathering_spots.json.gz
var gatheringSpotsGz []byte

type gatheringSpot struct {
	MapID    uint16        `json:"m"`
	Category radarCategory `json:"c"`
	Name     string        `json:"n"`
	X        float32       `json:"x"`
	Y        float32       `json:"y"`
	Z        float32       `json:"z"`
}

var (
	gatheringSpotsOnce sync.Once
	gatheringSpotsList []gatheringSpot
	spotsByMap         map[uint16][]gatheringSpot
)

func initGatheringSpots() {
	gatheringSpotsOnce.Do(func() {
		if len(gatheringSpotsGz) == 0 {
			return
		}
		gz, err := gzip.NewReader(bytes.NewReader(gatheringSpotsGz))
		if err != nil {
			return
		}
		defer gz.Close()

		var spots []gatheringSpot
		if err := json.NewDecoder(gz).Decode(&spots); err != nil {
			return
		}
		gatheringSpotsList = spots
		spotsByMap = make(map[uint16][]gatheringSpot)
		for _, s := range spots {
			spotsByMap[s.MapID] = append(spotsByMap[s.MapID], s)
		}
	})
}

// detectMapFromPos identifies which zone map the player is on by checking the nearest gathering spot cluster.
func detectMapFromPos(pos game.Pos) uint32 {
	initGatheringSpots()
	if len(gatheringSpotsList) == 0 {
		return 0
	}
	// Max distance to consider: 10,000 meters (1,000,000 cm)
	var bestDistSq float64 = 1000000.0 * 1000000.0
	var bestMap uint32
	for _, s := range gatheringSpotsList {
		dx := float64(s.X - pos.X)
		dy := float64(s.Y - pos.Y)
		dsq := dx*dx + dy*dy
		if dsq < bestDistSq {
			bestDistSq = dsq
			bestMap = uint32(s.MapID)
		}
	}
	return bestMap
}

// queryNearbyGatheringSpots finds all gathering spots within maxMeters of pos for mapID.
func queryNearbyGatheringSpots(mapID uint32, pos game.Pos, maxMeters float64) []gatheringSpot {
	initGatheringSpots()
	if mapID == 0 {
		mapID = detectMapFromPos(pos)
	}
	spots := spotsByMap[uint16(mapID)]
	if len(spots) == 0 {
		// Fallback: if map is unknown, check all spots
		spots = gatheringSpotsList
	}
	if len(spots) == 0 {
		return nil
	}
	maxCm := maxMeters * 100.0
	maxCmSq := maxCm * maxCm
	var out []gatheringSpot
	for _, s := range spots {
		dx := float64(s.X - pos.X)
		dy := float64(s.Y - pos.Y)
		dsq := dx*dx + dy*dy
		if dsq <= maxCmSq {
			dz := float64(s.Z - pos.Z)
			if math.Hypot(math.Sqrt(dsq), dz) <= maxCm {
				out = append(out, s)
			}
		}
	}
	return out
}
