package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Thijs-Desjardijn/pokedex/pokecache"

	"io"
	"net/http"

	//"net/url"
	"math/rand"
	"os"
	"strings"
	//"errors"
)

type Config struct {
	Next     string
	Previous string
}

type cliCommand struct {
	name        string
	description string
	callback    func(*Config, string) error
}

type LocationAreaResponse struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"results"`
}

type LocationArea struct {
	PokemonEncounters []struct {
		Pokemon struct {
			Name string `json:"name"`
		} `json:"pokemon"`
	} `json:"pokemon_encounters"`
}

type PokemonInformation struct {
	BaseExperience int     `json:"base_experience"`
	Name           string  `json:"name"`
	Height         int     `json:"height"`
	Weight         int     `json:"weight"`
	Stats          []Stat  `json:"stats"`
	Types          []PType `json:"types"`
}

type Stat struct {
	BaseStat int      `json:"base_stat"`
	StatInfo StatInfo `json:"stat"`
}

type StatInfo struct {
	Name string `json:"name"`
}

type PType struct {
	Type TypeInfo `json:"type"`
}

type TypeInfo struct {
	Name string `json:"name"`
}

func commandPokedex(_ *Config, _ string) error {
	fmt.Println("Your Pokedex:")
	for _, pokemon := range pokeDex {
		fmt.Printf("- %s\n", pokemon.Name)
	}
	return nil
}
func commandInspect(_ *Config, pokemonName string) error {
	pokemon, ok := pokeDex[pokemonName]
	if !ok {
		fmt.Println("You have not yet caught this pokemon")
		return nil
	}
	fmt.Println("stats:")
	fmt.Printf("name: %s\nheight: %d\nweight: %d\n", pokemon.Name, pokemon.Height, pokemon.Weight)
	for _, stat := range pokemon.Stats {
		fmt.Printf("%s: %v\n", stat.StatInfo.Name, stat.BaseStat)
	}
	fmt.Println("type:")
	for _, t := range pokemon.Types {
		fmt.Printf("- %v\n", t.Type.Name)
	}
	return nil
}

func cleanInput(text string) []string {
	lowerText := strings.ToLower(text)
	cleanString := strings.Fields(lowerText)
	return cleanString
}

func commandExit(cfg *Config, _ string) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandHelp(cfg *Config, _ string) error {
	fmt.Printf("Welcome to the Pokedex!\nUsage:\n\n")
	for _, command := range supportedCommands {
		fmt.Printf("%s: %s\n", command.name, command.description)
	}
	return nil
}

func commandMap(cfg *Config, _ string) error {
	var url string
	if cfg.Next == "" {
		url = "https://pokeapi.co/api/v2/location-area?offset=0&limit=20"
	} else {
		url = cfg.Next
	}
	data, err := GetData(cache, url)
	if err != nil {
		return err
	}
	var allLocations LocationAreaResponse
	err = json.Unmarshal(data, &allLocations)
	if err != nil {
		return err
	}
	for _, location := range allLocations.Results {
		fmt.Println(location.Name)
	}
	cfg.Next = allLocations.Next
	cfg.Previous = allLocations.Previous
	return nil
}

func commandMapb(cfg *Config, _ string) error {
	var url string
	if cfg.Previous == "" {
		fmt.Println("you're on the first page")
		return nil
	} else {
		url = cfg.Previous
	}

	data, err := GetData(cache, url)
	if err != nil {
		return err
	}
	var allLocations LocationAreaResponse
	err = json.Unmarshal(data, &allLocations)
	if err != nil {
		return err
	}
	for _, location := range allLocations.Results {
		fmt.Println(location.Name)
	}
	cfg.Next = allLocations.Next
	cfg.Previous = allLocations.Previous
	return nil
}

func GetData(c *pokecache.Cache, url string) ([]byte, error) {
	data, ok := c.Get(url)
	if ok {
		fmt.Println("Cache used!")
		return data, nil
	} else {
		res, err := http.Get(url)
		if err != nil {
			return []byte{}, err
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			return []byte{}, fmt.Errorf("%v, check spelling and/or if the area exists", res.Status)
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Println(err)
			return []byte{}, err
		}
		c.Add(url, body)
		return body, nil
	}
}

var pokeDex map[string]PokemonInformation
var cache *pokecache.Cache
var supportedCommands map[string]cliCommand

func commandExplore(_ *Config, nameLocation string) error {
	url := "https://pokeapi.co/api/v2/location-area/" + nameLocation + "/"
	data, err := GetData(cache, url)
	if err != nil {
		fmt.Printf("1:%v\n", err)
		return err
	}
	var area LocationArea
	err = json.Unmarshal(data, &area)
	if err != nil {
		fmt.Printf("2:%v\n", err)
		return err
	}
	for _, pokemon := range area.PokemonEncounters {
		fmt.Println(pokemon.Pokemon.Name)
	}
	return nil
}

func commandCatch(_ *Config, pokemonName string) error {
	url := "https://pokeapi.co/api/v2/pokemon/" + pokemonName
	data, err := GetData(cache, url)
	if err != nil {
		fmt.Println("1", err)
		return err
	}
	var pokemon PokemonInformation
	err = json.Unmarshal(data, &pokemon)
	if err != nil {
		fmt.Println("2", err)
		return err
	}
	fmt.Printf("Throwing a Pokeball at %s...\n", pokemonName)
	const (
		MaxBaseExp = 635.0 // highest known base experience (e.g. Blissey)
		MinChance  = 5     // minimum capture chance %
		MaxChance  = 90    // maximum capture chance %
	)

	// Normalize base experience into a range and invert
	chance := MaxChance - int((float64(pokemon.BaseExperience)/MaxBaseExp)*float64(MaxChance-MinChance))

	// Clamp to avoid going below MinChance
	if chance < MinChance {
		chance = MinChance
	}
	catchSucces := rand.Intn(100) < chance
	if catchSucces {
		fmt.Printf("%s was caught!\n", pokemonName)
	} else {
		fmt.Printf("%s escaped!\n", pokemonName)
		return nil
	}
	pokeDex[pokemonName] = pokemon
	return nil
}

func main() {
	cfg := &Config{}
	cache = pokecache.NewCache(60 * time.Second)
	pokeDex = make(map[string]PokemonInformation)
	supportedCommands = map[string]cliCommand{
		"exit": {
			name:        "exit",
			description: "Exit the Pokedex",
			callback:    commandExit,
		},

		"help": {
			name:        "help",
			description: "Displays a help message",
			callback:    commandHelp,
		},

		"map": {
			name:        "map",
			description: "Displays the locations in the pokemon world on the next page",
			callback:    commandMap,
		},

		"mapb": {
			name:        "mapb",
			description: "Displays the locations in the pokemon world on the previous page",
			callback:    commandMapb,
		},

		"explore": {
			name:        "commandexplore",
			description: "Displays the pokemons that can be found in a specified location after the command",
			callback:    commandExplore,
		},

		"catch": {
			name:        "catch",
			description: "Command to try to catch a specified pokemon after the command",
			callback:    commandCatch,
		},

		"inspect": {
			name:        "inspect",
			description: "Displays the stats of a caught pokemon",
			callback:    commandInspect,
		},

		"pokedex": {
			name:        "pokedex",
			description: "Displays all your caught pokemons in your Pokedex",
			callback:    commandPokedex,
		},
	}
	scanner := bufio.NewScanner(os.Stdin)

	// This blocks execution and waits for the user to input something and press Enter
	for {
		fmt.Print("Pokedex > ")
		if scanner.Scan() {

			input := scanner.Text()
			cleanedInput := cleanInput(input)
			if len(cleanedInput) < 1 {
				fmt.Printf("\n")
				fmt.Println("input needs to be atleast 1 character long")
				continue
			} // Get the input text
			command := cleanedInput[0]
			secondCommand := ""
			if len(cleanedInput) > 1 {
				secondCommand = cleanedInput[1]
			}
			if cmd, exists := supportedCommands[command]; exists {
				cmd.callback(cfg, secondCommand)
			} else {
				fmt.Println("Unknown command")
			}
		}
	}
}
