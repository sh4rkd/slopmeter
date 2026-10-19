package main

import (
	"fmt"
	"github.com/nuriland/a2kit/game"
	"sort"
)

type identities struct {
	names   map[game.Entity]string
	classes map[game.Entity]string
	players map[game.Entity]bool
	stats   map[string]playerStats
	self    game.Entity
	name    string
}

func (i *identities) observe(event game.Event) {
	if p, ok := event.(game.Player); ok {
		if i.names == nil {
			i.names = make(map[game.Entity]string)
		}
		if i.players == nil {
			i.players = make(map[game.Entity]bool)
		}
		if p.Entity != 0 {
			i.players[p.Entity] = true
		}
		if p.Name != "" {
			i.names[p.Entity] = p.Name
		}
		if p.Self {
			i.self = p.Entity
			i.name = p.Name
		}
	}
}

type row struct {
	Timeline          []skillUse  `json:"timeline,omitempty"`
	TimelineTruncated bool        `json:"timelineTruncated,omitempty"`
	Enemy             bool        `json:"enemy,omitempty"`
	GearScore         uint32      `json:"gearScore"`
	CombatPower       uint64      `json:"combatPower"`
	ID                game.Entity `json:"id"`
	Name              string      `json:"name"`
	Class             string      `json:"class"`
	Damage            uint64      `json:"damage"`
	DamageTaken       uint64      `json:"damageTaken,omitempty"`
	DPS               float64     `json:"dps"`
	CriticalRate      float64     `json:"criticalRate,omitempty"`
	CriticalKnown     bool        `json:"criticalKnown,omitempty"`
	MaxHit            uint32      `json:"maxHit,omitempty"`
	Share             float64     `json:"share"`
	Fill              float64     `json:"fill"`
	Skills            []skillRow  `json:"skills"`
}

type pvpDisplay struct {
	OpponentID    game.Entity `json:"opponentId,omitempty"`
	OpponentName  string      `json:"opponentName,omitempty"`
	OpponentClass string      `json:"opponentClass,omitempty"`
	DamageDealt   uint64      `json:"damageDealt"`
	DamageTaken   uint64      `json:"damageTaken"`
	DealtDPS      float64     `json:"dealtDps"`
	TakenDPS      float64     `json:"takenDps"`
	DealtShare    float64     `json:"dealtShare"`
	Outcome       string      `json:"outcome,omitempty"`
}

type snapshot struct {
	Cleared      bool            `json:"cleared,omitempty"`
	PvP          bool            `json:"pvp,omitempty"`
	PvPSummary   *pvpDisplay     `json:"pvpSummary,omitempty"`
	Scene        sceneDisplay    `json:"scene"`
	Character    string          `json:"character"`
	Actors       []row           `json:"actors"`
	Enemies      []row           `json:"enemies,omitempty"`
	Duration     float64         `json:"duration"`
	Session      uint64          `json:"session"`
	Encounter    string          `json:"encounter"`
	Active       bool            `json:"active"`
	Boss         *bossDisplay    `json:"boss"`
	HealthStatus string          `json:"healthStatus"`
	History      *[]historyEntry `json:"history,omitempty"`
}

func (m *meter) rows(i identities, seconds float64, enemy bool) []row {
	if seconds < 1 {
		seconds = 1
	}
	var sum, largest uint64
	for _, a := range m.actors {
		sum += a.damage
		largest = max(largest, a.damage)
	}
	out := []row{}
	for id, a := range m.actors {
		if a.damage == 0 && (!enemy || a.damageTaken == 0) {
			continue
		}
		name := i.names[id]
		if name == "" {
			name = fmt.Sprintf("Entity %d", id)
		}
		share := 0.0
		if sum > 0 {
			share = float64(a.damage) / float64(sum) * 100
		}
		fill := 0.0
		if largest > 0 {
			fill = float64(a.damage) / float64(largest) * 100
		}
		critRate := 0.0
		if a.typed > 0 {
			critRate = float64(a.critical) / float64(a.typed) * 100
		}
		out = append(out, row{
			Timeline:          a.timelineRows(m.first),
			TimelineTruncated: a.timelineTruncated,
			Enemy:             enemy,
			GearScore:         i.stats[name].GearScore,
			CombatPower:       i.stats[name].CombatPower,
			ID:                id,
			Name:              name,
			Class:             i.classes[id],
			Damage:            a.damage,
			DamageTaken:       a.damageTaken,
			DPS:               float64(a.damage) / seconds,
			CriticalRate:      critRate,
			CriticalKnown:     a.typed > 0,
			MaxHit:            a.maxHit,
			Share:             share,
			Fill:              fill,
			Skills:            a.skillRows(seconds, sum),
		})
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Damage == out[b].Damage {
			if out[a].DamageTaken != out[b].DamageTaken {
				return out[a].DamageTaken > out[b].DamageTaken
			}
			return out[a].ID < out[b].ID
		}
		return out[a].Damage > out[b].Damage
	})
	return out
}

func (m *meter) snapshot(i identities) snapshot {
	duration := m.last.Sub(m.first).Seconds()
	return snapshot{
		Character: i.name,
		Actors:    m.rows(i, duration, false),
		Duration:  duration,
	}
}
