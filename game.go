package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/op/go-logging"
)

type GameResponse struct {
	Success bool     `json:"success"`
	Data    GameData `json:"data"`
}

type GameData struct {
	Coins          int64         `json:"coins"`
	Cubes          int           `json:"cubes"`
	Energy         int           `json:"energy"`
	FedTimes       string        `json:"fed_times"`
	Food           int           `json:"food"`
	HarvestedTimes string        `json:"harvested_times"`
	Level          int           `json:"level"`
	Ltvalue        float64       `json:"ltvalue"`
	Population     int           `json:"population"`
	Spaceship      string        `json:"spaceship"`
	Colonies       []Colony      `json:"colonies"`
	Satellites     []Satellite   `json:"satellites"`
	Missions       []Mission     `json:"missions"`
	Activities     []Activity    `json:"activities"`
	Spaceships     []Spaceship   `json:"spaceships"`
	Achievements   []Achievement `json:"achievements"`
	Messages       []int         `json:"messages"`
}
type Achievement struct {
	Completed  bool    `json:"completed"`
	Identifier string  `json:"identifier"`
	Progress   float64 `json:"progress"`
	UpdatedAt  string  `json:"updated_at"`
}
type Spaceship struct {
	Identifier string `json:"identifier"`
	UpdatedAt  string `json:"updated_at"`
}
type Activity struct {
	Data      string `json:"data"`
	Running   int    `json:"running"`
	UpdatedAt string `json:"updated_at"`
	Walking   int    `json:"walking"`
}
type Mission struct {
	Aborted    bool   `json:"aborted"`
	Completed  bool   `json:"completed"`
	Identifier string `json:"identifier"`
	ResourceA  int    `json:"resource_a"`
	ResourceB  int    `json:"resource_b"`
	ResourceC  int    `json:"resource_c"`
	UpdatedAt  string `json:"updated_at"`
}
type Colony struct {
	Category     string  `json:"category"`
	ColonyOrder  int     `json:"colony_order"`
	ColonyType   string  `json:"colony_type"`
	Completed    bool    `json:"completed"`
	DiscoveredAt string  `json:"discovered_at"`
	GalaxyId     int     `json:"galaxy_id"`
	Identifier   string  `json:"identifier"`
	Level        int     `json:"level"`
	OffsetX      float64 `json:"offset_x"`
	OffsetY      float64 `json:"offset_y"`
	ProcessTime  float64 `json:"process_time"`

	ProcessedAt string          `json:"processed_at"`
	UpdatedAt   string          `json:"updated_at"`
	ZPosition   float64         `json:"z_position"`
	ZRotation   float64         `json:"z_rotation"`
	Satellites  []MiniSatellite `json:"satellites"`
}
type MiniSatellite struct {
	Identifier string `json:"identifier"`
}
type Satellite struct {
	ColonyId   int    `json:"colony_id"`
	Identifier string `json:"identifier"`
	UpdatedAt  string `json:"updated_at"`
}

var log = logging.MustGetLogger("Walkr")
var format = logging.MustStringFormatter(
	"%{color}%{time:15:04:05.000} %{shortfile} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}",
)

func main() {
	// 初始化Log
	stdOutput := logging.NewLogBackend(os.Stderr, "", 0)
	stdOutputFormatter := logging.NewBackendFormatter(stdOutput, format)

	logging.SetBackend(stdOutputFormatter)

	data, err := ioutil.ReadFile("./game.json")
	if err != nil {
		panic(err)
	}
	result := &GameResponse{}
	if err := json.Unmarshal(data, result); err != nil {
		panic(err)
	}

	// log.Debug("result: %v", result.Data)
	positionMap := map[int]Colony{}
	colonyMap := map[int]Colony{}
	replicatorMap := map[int]Colony{}
	tmpColonies := make([]Colony, len(result.Data.Colonies))

	startCIndex := 0
	startRIndex := 0
	cIndex := startCIndex
	rIndex := startRIndex
	for index, c := range result.Data.Colonies {
		positionMap[c.ColonyOrder] = c

		if index < 2 {
			continue
		}
		if c.ColonyType == "planet" {
			colonyMap[cIndex] = c
			cIndex += 1
		} else {
			replicatorMap[rIndex] = c
			rIndex += 1
		}
	}

	for index, _ := range tmpColonies {
		if index < 2 {
			tmpColonies[index] = positionMap[index+1]
		} else {
			tmpColonies[index] = Colony{
				Category:     "",
				ColonyOrder:  index + 1,
				ColonyType:   "replicator",
				Completed:    true,
				DiscoveredAt: positionMap[index+1].DiscoveredAt,
				GalaxyId:     1,
				Identifier:   "blender",
				Level:        8,
				OffsetX:      positionMap[index+1].OffsetX,
				OffsetY:      positionMap[index+1].OffsetY,
				ProcessTime:  0.0,
				ProcessedAt:  positionMap[index+1].ProcessedAt,
				UpdatedAt:    positionMap[index+1].UpdatedAt,
				ZPosition:    positionMap[index+1].ZPosition,
				ZRotation:    positionMap[index+1].ZRotation,
				Satellites:   []MiniSatellite{},
			}

			if (index-1)%4 != 0 { // Planet
				c := colonyMap[startCIndex]
				if c.Category != "" {
					tmpColonies[index] = Colony{
						Category:     c.Category,
						ColonyOrder:  index + 1,
						ColonyType:   c.ColonyType,
						Completed:    true,
						DiscoveredAt: positionMap[index+1].DiscoveredAt,
						GalaxyId:     1,
						Identifier:   c.Identifier,
						Level:        c.Level,
						OffsetX:      positionMap[index+1].OffsetX,
						OffsetY:      positionMap[index+1].OffsetY,
						ProcessTime:  positionMap[index+1].ProcessTime,
						ProcessedAt:  positionMap[index+1].ProcessedAt,
						UpdatedAt:    positionMap[index+1].UpdatedAt,
						ZPosition:    positionMap[index+1].ZPosition,
						ZRotation:    positionMap[index+1].ZRotation,
						Satellites:   c.Satellites,
					}
					startCIndex += 1
				}

			}
		}
	}

	result.Data.Colonies = tmpColonies
	jsonStr, _ := json.Marshal(result)
	ioutil.WriteFile("./out.json", jsonStr, 0777)

}
