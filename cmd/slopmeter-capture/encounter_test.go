package main

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/nuriland/a2kit/game"
	"github.com/nuriland/a2kit/wire"
)

func bossCode() uint32 { return 2090175 }
func encounterFixture() (*encounter, time.Time) {
	return newEncounter(0, 5*time.Second, 90*time.Second), time.Unix(100, 0)
}
func TestOpenWorldIdleAndNextPull(t *testing.T) {
	e, t0 := encounterFixture()
	e.hit(t0, game.Hit{Actor: 1, Target: 7, Damage: 100})
	e.hit(t0.Add(time.Second), game.Hit{Actor: 1, Target: 8, Damage: 200})
	if e.tick(t0.Add(5 * time.Second)) {
		t.Fatal("idle counted from first hit")
	}
	if !e.tick(t0.Add(6*time.Second)) || e.active || len(e.meter.actors) != 0 {
		t.Fatal("idle did not clear current encounter")
	}
	e.hit(t0.Add(7*time.Second), game.Hit{Actor: 1, Target: 9, Damage: 50})
	if e.number != 2 || e.meter.actors[1].damage != 50 {
		t.Fatal("new pull retained damage")
	}
}
func TestBossIsolationAndLongMechanic(t *testing.T) {
	e, t0 := encounterFixture()
	e.npc(20, bossCode())
	e.health(t0, 20, 1000, 1000)
	e.hit(t0, game.Hit{Actor: 1, Target: 9, Damage: 500})
	e.hit(t0.Add(time.Second), game.Hit{Actor: 1, Target: 20, Damage: 100})
	e.hit(t0.Add(2*time.Second), game.Hit{Actor: 1, Target: 9, Damage: 9000})
	if e.meter.actors[1].damage != 100 || e.boss != 20 {
		t.Fatal("trash included in boss totals")
	}
	if e.tick(t0.Add(20*time.Second)) || !e.active {
		t.Fatal("boss mechanic split encounter")
	}
	s := e.snapshot(identities{}, t0.Add(21*time.Second))
	if s.Duration != 20 || s.Actors[0].DPS != 5 {
		t.Fatal("boss DPS must include mechanics", s)
	}
	e.health(t0.Add(22*time.Second), 20, 0, 0)
	if e.active || e.status != "Boss defeated" {
		t.Fatal("boss death did not finish encounter")
	}
	s = e.snapshot(identities{}, t0.Add(100*time.Second))
	if s.Duration != 21 {
		t.Fatal("completed duration was not frozen")
	}
	e.health(t0.Add(101*time.Second), 20, 1000, 1000)
	e.hit(t0.Add(102*time.Second), game.Hit{Actor: 1, Target: 20, Damage: 80})
	if e.meter.actors[1].damage != 80 || e.number != 3 {
		t.Fatal("repull retained totals")
	}
}
func TestBossResetAndOpenWorldAfterKill(t *testing.T) {
	e, t0 := encounterFixture()
	e.npc(20, bossCode())
	e.health(t0, 20, 1000, 1000)
	e.hit(t0, game.Hit{Actor: 1, Target: 20, Damage: 100})
	e.health(t0.Add(time.Second), 20, 800, 0)
	e.health(t0.Add(2*time.Second), 20, 1000, 0)
	s := e.snapshot(identities{}, t0.Add(2*time.Second))
	if e.active || !s.Cleared || len(s.Actors) != 0 || s.Duration != 0 || s.Boss == nil || s.Boss.HP != 1000 || s.Boss.Percent != 100 {
		t.Fatalf("reset to full health did not clear damage and timers: %+v", s)
	}
	e.hit(t0.Add(3*time.Second), game.Hit{Actor: 1, Target: 9, Damage: 50})
	if e.boss != 0 || e.meter.target != 0 || e.meter.actors[1].damage != 50 {
		t.Fatal("boss filter leaked into open world")
	}
	// Reset must also trigger when max HP was not captured in spawn data, if HP returns to initial peak.
	e2, _ := encounterFixture()
	e2.npc(20, bossCode())
	e2.health(t0, 20, 5000, 0)
	e2.hit(t0.Add(time.Second), game.Hit{Actor: 1, Target: 20, Damage: 400})
	e2.health(t0.Add(2*time.Second), 20, 4600, 0)
	e2.health(t0.Add(3*time.Second), 20, 5000, 0)
	s2 := e2.snapshot(identities{}, t0.Add(3*time.Second))
	if e2.active || !s2.Cleared || len(s2.Actors) != 0 || s2.Duration != 0 {
		t.Fatalf("reset without spawn max HP did not clear fight: %+v", s2)
	}
}
func TestBossPacketHealthAndUnknownMaximum(t *testing.T) {
	e, t0 := encounterFixture()
	p := binary.AppendUvarint(nil, 20)
	rest := make([]byte, 30)
	rest[0] = 5
	rest[1] = 0x20
	binary.LittleEndian.PutUint32(rest[5:], bossCode())
	p = append(p, rest...)
	p = binary.AppendUvarint(p, 1000)
	p = binary.AppendUvarint(p, 2000)
	p = append(p, 100, 0, 0, 0, 100, 0, 0, 0)
	e.observe(aMessage{opcode: 0x3641, flags: wire.FromServer, payload: p, t: t0})
	e.hit(t0, game.Hit{Actor: 1, Target: 20, Damage: 100})
	hp := binary.AppendUvarint(nil, 20)
	hp = append(hp, 2, 1, 0)
	hp = binary.LittleEndian.AppendUint32(hp, 600)
	e.observe(aMessage{opcode: 0x8d00, flags: wire.FromServer, payload: hp, t: t0.Add(time.Second)})
	s := e.snapshot(identities{}, t0.Add(time.Second))
	if s.Boss == nil || !s.Boss.MaxKnown || s.Boss.HP != 600 || s.Boss.Percent != 30 || s.Boss.Name != "Fediv Wraith" {
		t.Fatal("incorrect packet HP", s.Boss)
	}
	hp[1] = 3 // A non-health value must not overwrite HP.
	e.observe(aMessage{opcode: 0x8d00, flags: wire.FromServer, payload: hp, t: t0})
	if e.npcs[20].hp != 600 {
		t.Fatal("non-health resource accepted")
	}
	e2, _ := encounterFixture()
	e2.npc(20, bossCode())
	e2.health(t0, 20, 700, 0)
	e2.hit(t0, game.Hit{Actor: 1, Target: 20, Damage: 10})
	if e2.snapshot(identities{}, t0).Boss.MaxKnown {
		t.Fatal("invented maximum HP")
	}
}
func TestRelativeBars(t *testing.T) {
	m := meter{}
	t0 := time.Unix(100, 0)
	m.add(t0, game.Hit{Actor: 1, Damage: 100})
	m.add(t0, game.Hit{Actor: 2, Damage: 50})
	s := m.snapshot(identities{})
	if s.Actors[0].Fill != 100 || s.Actors[1].Fill != 50 {
		t.Fatal("bars are not relative to leader")
	}
}
func TestBossIdleDeathAndZone(t *testing.T) {
	e, t0 := encounterFixture()
	e.npc(20, bossCode())
	e.hit(t0, game.Hit{Actor: 1, Target: 20, Damage: 10})
	e.observe(aMessage{event: game.Death{Entity: 20, Flag: 3}, flags: wire.FromServer, t: t0.Add(time.Second)})
	if e.active {
		t.Fatal("death packet did not stop boss fight")
	}
	e.scene.ID = 600021
	e.scene.Name = mapCatalog[600021]
	e.scene.candidate = 600031
	e.observe(aMessage{opcode: 0x3623, payload: make([]byte, 20), flags: wire.FromServer, t: t0.Add(2 * time.Second)})
	if e.active || e.boss != 0 || len(e.npcs) != 0 {
		t.Fatal("zone retained old boss")
	}
	e.npc(21, bossCode())
	e.hit(t0.Add(3*time.Second), game.Hit{Actor: 1, Target: 21, Damage: 20})
	if e.tick(t0.Add(194*time.Second)) || !e.active {
		t.Fatal("long boss downtime split encounter")
	}
}

func TestOutOfOrderHitsKeepTotalsAndClock(t *testing.T) {
	e, t0 := encounterFixture()
	e.hit(t0, game.Hit{Actor: 1, Target: 7, Damage: 100})
	e.hit(t0.Add(2*time.Second), game.Hit{Actor: 1, Target: 7, Damage: 200})
	e.hit(t0.Add(time.Second), game.Hit{Actor: 1, Target: 7, Damage: 50})
	if e.meter.actors[1].damage != 350 || e.lastDamage != t0.Add(2*time.Second) {
		t.Fatal("out-of-order hit lost or clock rewound")
	}
}

func TestDelayedHealthCannotResetFight(t *testing.T) {
	e, t0 := encounterFixture()
	e.npc(20, bossCode())
	e.hit(t0, game.Hit{Actor: 1, Target: 20, Damage: 100})
	e.health(t0.Add(2*time.Second), 20, 700, 0)
	e.health(t0, 20, 1000, 1000)
	if !e.active || e.npcs[20].hp != 700 || e.npcs[20].max != 1000 {
		t.Fatal("delayed spawn HP rewound current health or reset fight")
	}
}

func TestPvPDetectionExchangeAndOutcome(t *testing.T) {
	e, t0 := encounterFixture()
	group := scope{identities: identities{names: make(map[game.Entity]string)}}
	group.observeMessage(aMessage{flags: wire.FromServer, event: game.Player{Entity: 1, Name: "Me", Self: true}, t: t0})
	group.observeMessage(aMessage{flags: wire.FromServer, event: game.Player{Entity: 99, Name: "EnemyPlayer"}, t: t0})

	var assassinSkill game.Skill
	for id, cls := range skillClasses {
		if cls == "Assassin" {
			assassinSkill = id
			break
		}
	}
	// Enemy casts and hits us
	castMsg := aMessage{flags: wire.FromServer, event: game.Cast{Actor: 99, Target: 1, Skill: assassinSkill}, t: t0}
	group.observeMessage(castMsg)
	hitIn := game.Hit{Actor: 99, Target: 1, Skill: assassinSkill, Damage: 1200, Type: 3}
	group.observeMessage(aMessage{flags: wire.FromServer, event: hitIn, t: t0})
	e.pvpHit(t0, hitIn, false)
	e.enemyCast(t0, game.Cast{Actor: 99, Target: 1, Skill: assassinSkill})

	// We hit enemy back
	hitOut := game.Hit{Actor: 1, Target: 99, Skill: 100, Damage: 1800, Type: 3}
	e.pvpHit(t0.Add(2*time.Second), hitOut, true)

	// Enemy dies
	e.observe(aMessage{flags: wire.FromServer, event: game.Death{Entity: 99, Flag: 3}, t: t0.Add(2 * time.Second)})

	s := e.snapshot(group.identities, t0.Add(2*time.Second))
	if !s.PvP || s.PvPSummary == nil {
		t.Fatalf("expected PvP mode snapshot, got: %+v", s)
	}
	if s.PvPSummary.DamageDealt != 1800 || s.PvPSummary.DamageTaken != 1200 || s.PvPSummary.OpponentName != "EnemyPlayer" || s.PvPSummary.OpponentClass != "Assassin" || s.PvPSummary.Outcome != "Victory" {
		t.Fatalf("unexpected PvP summary: %+v", s.PvPSummary)
	}
	if len(s.Actors) != 1 || s.Actors[0].Damage != 1800 || s.Actors[0].DamageTaken != 1200 || s.Actors[0].MaxHit != 1800 || s.Actors[0].CriticalRate != 100 {
		t.Fatalf("unexpected ally PvP row: %+v", s.Actors)
	}
	if len(s.Enemies) != 1 || !s.Enemies[0].Enemy || s.Enemies[0].Name != "EnemyPlayer" || s.Enemies[0].Class != "Assassin" || s.Enemies[0].Damage != 1200 || s.Enemies[0].DamageTaken != 1800 || len(s.Enemies[0].Timeline) != 1 {
		t.Fatalf("unexpected enemy PvP row: %+v", s.Enemies)
	}
}

