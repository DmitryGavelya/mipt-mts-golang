package main

import (
	"fmt"
	"strings"
)

// Решение пишите в этом файле. Контракт — только initGame и
// handleCommand, без экспортируемых типов и функций. Состояние партии
// между вызовами держится на уровне пакета: фиксированные сигнатуры не
// дают передать его параметром.

// initGame инициализирует героя и волну врагов. Числа фиксированы,
// тесты рассчитаны на них:
//
//	герой: HP 20/20, атака 10, инвентарь — зелье x1, меч x1
//	волна по порядку:
//	  1. goblin,   HP  8/8,  атака 3
//	  2. skeleton, HP 10/10, атака 2
//	  3. goblin,   HP 15/15, атака 4

type heroState struct {
	hp        int
	maxHP     int
	attack    int
	inventory map[string]int
}

type enemy interface {
	name() string
	maxHP() int
	currentHP() int
	attackValue() int
	isDefeated() bool
	setDefeated(bool)
	takeDamage(int) (bool, string)
}

type enemyBase struct {
	nameText string
	hp       int
	maxHp    int
	atk      int
	defeated bool
}

func (e *enemyBase) name() string       { return e.nameText }
func (e *enemyBase) maxHP() int         { return e.maxHp }
func (e *enemyBase) currentHP() int     { return e.hp }
func (e *enemyBase) attackValue() int   { return e.atk }
func (e *enemyBase) isDefeated() bool   { return e.defeated }
func (e *enemyBase) setDefeated(v bool) { e.defeated = v }

type goblin struct{ enemyBase }

func (g *goblin) takeDamage(dmg int) (bool, string) {
	if g.defeated {
		return true, ""
	}
	g.hp -= dmg
	if g.hp <= 0 {
		g.hp = 0
		g.defeated = true
		return true, "Противник повержен."
	}
	return false, fmt.Sprintf("HP противника: %d/%d.", g.hp, g.maxHp)
}

type skeleton struct {
	enemyBase
	revived bool
}

func (s *skeleton) takeDamage(dmg int) (bool, string) {
	if s.defeated {
		return true, ""
	}
	s.hp -= dmg
	if s.hp > 0 {
		return false, fmt.Sprintf("HP противника: %d/%d.", s.hp, s.maxHp)
	}
	if s.revived {
		s.hp = 0
		s.defeated = true
		return true, "Противник повержен."
	}
	s.revived = true
	s.hp = s.maxHp / 2
	s.defeated = false
	return false, fmt.Sprintf("Скелет восстаёт с HP %d.", s.hp)
}

var (
	hero = heroState{
		hp:        20,
		maxHP:     20,
		attack:    10,
		inventory: map[string]int{"зелье": 1, "меч": 1},
	}
	wave     []enemy
	current  enemy
	index    int
	waveDone bool
	gameOver bool
)

func initGame() {
	hero = heroState{
		hp:        20,
		maxHP:     20,
		attack:    10,
		inventory: map[string]int{"зелье": 1, "меч": 1},
	}
	wave = []enemy{
		&goblin{enemyBase: enemyBase{nameText: "goblin", hp: 8, maxHp: 8, atk: 3}},
		&skeleton{enemyBase: enemyBase{nameText: "skeleton", hp: 10, maxHp: 10, atk: 2}},
		&goblin{enemyBase: enemyBase{nameText: "goblin", hp: 15, maxHp: 15, atk: 4}},
	}
	index = 0
	current = wave[index]
	waveDone = false
	gameOver = false
}

func statusText() string {
	if current == nil {
		return fmt.Sprintf("Ваше HP: %d/%d, атака: %d. Противников не осталось.", hero.hp, hero.maxHP, hero.attack)
	}
	if current.isDefeated() {
		return fmt.Sprintf("Ваше HP: %d/%d, атака: %d. Текущий противник повержен.", hero.hp, hero.maxHP, hero.attack)
	}
	return fmt.Sprintf("Ваше HP: %d/%d, атака: %d. Текущий противник: %s, HP %d/%d.", hero.hp, hero.maxHP, hero.attack, current.name(), current.currentHP(), current.maxHP())
}

func handleUseItem(item string) string {
	if gameOver || waveDone {
		return "Игра окончена."
	}
	if hero.inventory[item] <= 0 {
		return fmt.Sprintf("Нет предмета в инвентаре: %s.", item)
	}

	switch item {
	case "зелье":
		hero.inventory[item]--
		hero.hp += 20
		if hero.hp > hero.maxHP {
			hero.hp = hero.maxHP
		}
		return fmt.Sprintf("Использовано: %s. Ваше HP: %d/%d.", item, hero.hp, hero.maxHP)
	case "меч":
		hero.inventory[item]--
		hero.attack += 5
		return fmt.Sprintf("Использовано: %s. Атака увеличена.", item)
	default:
		return fmt.Sprintf("Нет предмета в инвентаре: %s.", item)
	}
}

func handleCommand(command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return "неизвестная команда"
	}

	switch command {
	case "атаковать":
		if gameOver || waveDone {
			return "Игра окончена."
		}
		if current == nil {
			return "Игра окончена."
		}
		if current.isDefeated() {
			return "Текущий противник уже повержен, используйте \"дальше\"."
		}

		attackPower := hero.attack
		killed, msg := current.takeDamage(attackPower)
		if killed {
			current.setDefeated(true)
			return fmt.Sprintf("Урон противнику: %d. %s", attackPower, msg)
		}
		if strings.HasPrefix(msg, "Скелет восстаёт") {
			enemyDamage := current.attackValue()
			hero.hp -= enemyDamage
			if hero.hp <= 0 {
				gameOver = true
			}
			return fmt.Sprintf("Урон противнику: %d. %s Урон вам: %d. Ваше HP: %d/%d.", attackPower, msg, enemyDamage, hero.hp, hero.maxHP)
		}
		enemyDamage := current.attackValue()
		hero.hp -= enemyDamage
		if hero.hp <= 0 {
			gameOver = true
		}
		return fmt.Sprintf("Урон противнику: %d. %s Урон вам: %d. Ваше HP: %d/%d.", attackPower, msg, enemyDamage, hero.hp, hero.maxHP)
	case "дальше":
		if gameOver {
			return "Игра окончена."
		}
		if current == nil {
			return "Дальше идти некуда."
		}
		if !current.isDefeated() {
			return "Текущий противник ещё жив, сначала победите его."
		}
		index++
		if index >= len(wave) {
			waveDone = true
			current = nil
			return fmt.Sprintf("Волна пройдена. Победа! Ваше HP: %d/%d.", hero.hp, hero.maxHP)
		}
		current = wave[index]
		return fmt.Sprintf("Следующий противник: %s, HP %d/%d.", current.name(), current.currentHP(), current.maxHP())
	case "статус":
		return statusText()
	default:
		if strings.HasPrefix(command, "использовать ") {
			item := strings.TrimSpace(strings.TrimPrefix(command, "использовать "))
			if item == "" {
				return "неизвестная команда"
			}
			return handleUseItem(item)
		}
		return "неизвестная команда"
	}
}

// main по заданию не нужен. Можно добавить сюда построчный ввод команд,
// чтобы работал `go run main.go`.
func main() {}
