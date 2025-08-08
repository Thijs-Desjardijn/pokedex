package main

import (
	"bufio"
	"encoding/json"
	"fmt"

	//"log"
	"encoding/gob"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Thijs-Desjardijn/pokedex/pokecache"
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

var catchablePokemon map[string]PokemonInformation
var PokeDex map[string]PokemonInformation
var cache *pokecache.Cache
var supportedCommands map[string]cliCommand

func commandFind(_ *Config, area string) error {
	fmt.Printf("Looking for pokemon at %s\n", area)
	url := "https://pokeapi.co/api/v2/location-area/" + area + "/"
	data, err := GetData(cache, url)
	if err != nil {
		return err
	}
	var areaInfo LocationArea
	err = json.Unmarshal(data, &areaInfo)
	if err != nil {
		return err
	}
	index := rand.Intn(len(areaInfo.PokemonEncounters))
	pokemonName := areaInfo.PokemonEncounters[index].Pokemon.Name
	url = "https://pokeapi.co/api/v2/pokemon/" + pokemonName
	data1, err := GetData(cache, url)
	if err != nil {
		return err
	}

	var pokemon PokemonInformation
	err = json.Unmarshal(data1, &pokemon)
	if err != nil {
		return err
	}
	fmt.Printf("You found a %s!\nYou are now able to catch it using the catch command\n", pokemonName)
	catchablePokemon[pokemonName] = pokemon
	return nil
}

func commandPokedex(_ *Config, _ string) error {
	fmt.Println("Your Pokedex:")
	for _, pokemon := range PokeDex {
		fmt.Printf("- %s\n", pokemon.Name)
	}
	return nil
}
func commandInspect(_ *Config, pokemonName string) error {
	pokemon, ok := PokeDex[pokemonName]
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
	err := commandSave(&Config{}, "")
	if err != nil {
		return err
	}
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
	//function for all https requests automatically adds the info to the cache
	data, ok := c.Get(url)
	if ok {
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
			return []byte{}, err
		}
		c.Add(url, body)
		return body, nil
	}
}

func commandExplore(_ *Config, nameLocation string) error {
	url := "https://pokeapi.co/api/v2/location-area/" + nameLocation + "/"
	data, err := GetData(cache, url)
	if err != nil {
		return err
	}
	var area LocationArea
	err = json.Unmarshal(data, &area)
	if err != nil {
		return err
	}
	for _, pokemon := range area.PokemonEncounters {
		fmt.Println(pokemon.Pokemon.Name)
	}
	return nil
}

func commandCatch(_ *Config, pokemonName string) error {
	pokemon, ok := catchablePokemon[pokemonName]
	if !ok {
		fmt.Printf("You have not yet found %s", pokemonName)
		return nil
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
	PokeDex[pokemonName] = pokemon
	return nil
}

func commandSave(_ *Config, _ string) error {
	fmt.Printf("Saving your progress\nDo not shut off the program\n")
	filename := "save_" + time.Now().Format("20060102_150405") + ".bin"
	file, err := os.Create("save_folder/" + filename)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(PokeDex)
	if err != nil {
		return err
	}
	fmt.Println("Save sucessful")
	return nil
}

func readSave() error {
	files, err := os.ReadDir("save_folder")
	if err != nil {
		return err
	}
	var mostRecentFile string
	var mostRecentTime time.Time
	for _, file := range files {
		name := file.Name()
		time_s := name[5:20]
		t, err := time.Parse("20060102_150405", time_s)
		if err != nil {
			return err
		}
		if t.After(mostRecentTime) {
			mostRecentTime = t
			mostRecentFile = name
		}
	}
	fullpath := filepath.Join("save_folder", mostRecentFile)
	file, err := os.Open(fullpath)
	if err != nil {
		return err
	}
	gob.NewDecoder(file).Decode(&PokeDex)
	return nil
}
func main() {
	cfg := &Config{}
	cache = pokecache.NewCache(120 * time.Second)
	PokeDex = make(map[string]PokemonInformation)
	err := readSave()
	if err != nil {
		fmt.Printf("Unable to load save: %v\nPlease try again\n", err)
	}
	catchablePokemon = make(map[string]PokemonInformation)
	supportedCommands = map[string]cliCommand{
		"exit": {
			name:        "exit",
			description: "Exit the Pokedex and save your progress",
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
			name:        "explore",
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

		"find": {
			name:        "find",
			description: "Use this command to find pokemon in an area",
			callback:    commandFind,
		},

		"save": {
			name:        "save",
			description: "Command to save you progress",
			callback:    commandSave,
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
				err := cmd.callback(cfg, secondCommand)
				if err != nil {
					fmt.Println(err)
				}
			} else {
				fmt.Println("Unknown command")
			}
		}
	}
}
