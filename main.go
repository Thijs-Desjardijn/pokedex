package main

import (
	"bufio"
	"encoding/json"
	"fmt"

	"github.com/chzyer/readline"
	"github.com/fatih/color"

	//"go/format"
	//"log"
	"encoding/gob"
	"io"
	"math/rand"
	"strconv"

	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Thijs-Desjardijn/pokedex/internal/pokecache"
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
	MaxHp                  int
	Hp                     int
	Speed                  int
	Attack                 int
	MaxAttack              int
	Defense                int
	SpecialDefense         int
	SpecialAttack          int
	Level                  int
	BaseExperience         int                   `json:"base_experience"`
	Name                   string                `json:"name"`
	Height                 int                   `json:"height"`
	Weight                 int                   `json:"weight"`
	PokemonMovesAPIEntries []PokemonMoveAPIEntry `json:"moves"`
	Moves                  map[string]Move
	Stats                  []Stat  `json:"stats"`
	Types                  []PType `json:"types"`
}

type PokemonMoveAPIEntry struct {
	MoveInfo struct { // This struct corresponds to the "move" object within the API entry
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"move"`
}

type Move struct {
	Name     string `json:"name"`
	Power    int    `json:"power"`
	Accuracy int    `json:"accuracy"`

	Type struct {
		Name string `json:"name"`
	} `json:"type"`

	DamageClass struct {
		Name string `json:"name"`
	} `json:"damage_class"`

	EffectEntries []struct {
		Effect      string `json:"effect"`
		ShortEffect string `json:"short_effect"`
	} `json:"effect_entries"`
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

var (
	// Basic colors
	blue   = color.New(color.FgBlue).SprintFunc()
	orange = color.New(color.FgHiYellow).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	cyan   = color.New(color.FgCyan).SprintFunc()

	// Colors with bold styling
	boldRed    = color.New(color.FgRed, color.Bold).SprintFunc()
	boldGreen  = color.New(color.FgGreen, color.Bold).SprintFunc()
	boldYellow = color.New(color.FgYellow, color.Bold).SprintFunc()
)

func simpelLearnMove(pokemon *PokemonInformation) ([]string, error) {
	learnt_moves := []string{}
	for len(learnt_moves) < 3 {
		index := rand.Intn(len(pokemon.PokemonMovesAPIEntries))
		moveData, err := GetData(cache, pokemon.PokemonMovesAPIEntries[index].MoveInfo.URL)
		if err != nil {
			return []string{}, err
		}
		var move Move
		err = json.Unmarshal(moveData, &move)
		if err != nil {
			return []string{}, err
		}
		if move.DamageClass.Name == "status" {
			continue
		} else {
			pokemon.Moves[move.Name] = move
			learnt_moves = append(learnt_moves, move.Name)
		}
	}
	return learnt_moves, nil
}

func commandLearnMove(_ *Config, pokemonName string) error {
	for i, move := range PokeDex[pokemonName].PokemonMovesAPIEntries {
		fmt.Printf("%v: %v, ", boldYellow(i), cyan(move.MoveInfo.Name))
	}
	fmt.Printf("What move do you want to learn for %s\nplease type the number befor the move:", boldGreen(pokemonName))
	scanner := bufio.NewScanner(os.Stdin)
	for {
		if scanner.Scan() {
			input := scanner.Text()
			moveIndex, err := strconv.Atoi(input)
			if err != nil {
				fmt.Printf("\nWhat move do you want to learn for %s\nplease type the number befor the move:", boldGreen(pokemonName))
				continue
			}
			moveData, err := GetData(cache, PokeDex[pokemonName].PokemonMovesAPIEntries[moveIndex].MoveInfo.URL)
			if err != nil {
				return err
			}
			var move Move
			err = json.Unmarshal(moveData, &move)
			if err != nil {
				return err
			}
			if move.DamageClass.Name == "status" {
				fmt.Println("Sorry this move is not yet supported please choose another one")
				continue
			}
			PokeDex[pokemonName].Moves[move.Name] = move
			fmt.Printf("%s learnt move %s\n", boldGreen(pokemonName), cyan(move.Name))
		}
		return nil
	}
}

func playerMove(pokemon PokemonInformation, opponentPokemon *PokemonInformation) {
	for move := range pokemon.Moves {
		fmt.Printf("%s type: %s\n", cyan(pokemon.Moves[move].Name), green(pokemon.Moves[move].Type.Name))
	}
	fmt.Printf("choose a move to play:")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		if scanner.Scan() {
			input := scanner.Text()
			move, ok := pokemon.Moves[input]
			if !ok {
				for move := range pokemon.Moves {
					fmt.Printf("%s type: %s\n", cyan(pokemon.Moves[move].Name), green(pokemon.Moves[move].Type.Name))
				}
				fmt.Printf("\nchoose a move to play:")
				continue
			}
			fmt.Printf("%s plays %s\n", yellow(pokemon.Name), cyan(move.Name))
			calculateDamageMove(pokemon, opponentPokemon, move)
			break
		}
	}
}

func commandBattle(_ *Config, pokemonName string) error {
	if len(PokeDex) < 1 {
		fmt.Printf("You have no pokemon to fight with\nGo catch some pokemon!")
		return nil
	}
	pokemon, ok := catchablePokemon[pokemonName]
	if !ok {
		fmt.Println("You can't fight a pokemon you have not yet found using the find command")
		return nil
	}
	firstMove := true
	var your_pokemon PokemonInformation
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("Choose a pokemon to fight with:")
		if scanner.Scan() {
			input := scanner.Text()
			pokemon1, ok := PokeDex[input]
			if ok {
				your_pokemon = pokemon1
				break
			} else {
				fmt.Printf("This is not a pokemon you can fight with\n")
				commandPokedex(&Config{}, "")
			}
		}
	}
	fmt.Printf("\n")
	resetStats(&pokemon)
	resetStats(&your_pokemon)
	learnt_moves, err := simpelLearnMove(&pokemon)
	if err != nil {
		return err
	}
	for pokemon.Hp > 0 && your_pokemon.Hp > 0 {
		if pokemon.Speed > your_pokemon.Speed {
			firstMove = false
			index := rand.Intn(len(pokemon.Moves))
			move := learnt_moves[index]
			fmt.Printf("%s plays %s\n", yellow(pokemon.Name), cyan(pokemon.Moves[move].Name))
			calculateDamageMove(pokemon, &your_pokemon, pokemon.Moves[move])
			if your_pokemon.Hp <= 0 {
				break
			}
		} else {
			playerMove(your_pokemon, &pokemon)
			if pokemon.Hp <= 0 {
				break
			}
		}
		if !firstMove {
			playerMove(your_pokemon, &pokemon)
			if pokemon.Hp <= 0 {
				break
			}
		} else {
			index := rand.Intn(len(pokemon.Moves))
			move := learnt_moves[index]
			fmt.Printf("%s plays %s\n", yellow(pokemon.Name), cyan(pokemon.Moves[move].Name))
			calculateDamageMove(pokemon, &your_pokemon, pokemon.Moves[move])
			if your_pokemon.Hp <= 0 {
				break
			}
		}
	}
	if pokemon.Hp <= 0 {
		fmt.Printf("%s %s\n", yellow(pokemon.Name), boldRed("fainted"))
		fmt.Printf("maxHp: %v\n", your_pokemon.MaxHp)
		your_pokemon.Level += 1
		your_pokemon.MaxHp = int(math.Round((float64(your_pokemon.MaxHp) * 1.02)))
		fmt.Printf("newMaxHp: %v\n", your_pokemon.MaxHp)
		fmt.Println(boldGreen("You won!"))
	} else {
		fmt.Printf("%s %s\n", yellow(your_pokemon.Name), boldRed("fainted"))
	}
	resetStats(&pokemon)
	resetStats(&your_pokemon)
	return nil
}

func calculateDamageMove(attackerPokemon PokemonInformation, pokemon *PokemonInformation, move Move) {
	if rand.Intn(101) > move.Accuracy {
		fmt.Printf("%s %s %s!\n", pokemon.Name, yellow("dodged"), move.Name)
		return
	}
	var damage int
	if move.DamageClass.Name == "physical" {
		damage = 5 * (((2*attackerPokemon.Level/5+2)*move.Power*attackerPokemon.Attack/pokemon.Defense)/50 + 2)
	} else {
		damage = 5 * (((2*attackerPokemon.Level/5+2)*move.Power*attackerPokemon.SpecialAttack/pokemon.SpecialDefense)/50 + 2)
	}
	fmt.Printf("%s dealt: %s\n", boldRed("Damage"), red(fmt.Sprintf("%d", damage)))
	pokemon.Hp -= damage
}

func commandFind(_ *Config, area string) error {
	fmt.Printf("Looking for pokemon at %s\n", orange(area))
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
	pokemon.Moves = make(map[string]Move)
	fmt.Printf("You found a %s!\nYou are now able to catch %s using the %s command\nor you can %s it using the %s command\n", yellow(pokemonName), blue("catch"), yellow(pokemonName), red("fight"), blue("battle"))
	catchablePokemon[pokemonName] = pokemon
	return nil
}

func commandPokedex(_ *Config, _ string) error {
	fmt.Println(orange("Your Pokedex:"))
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
	fmt.Printf("%s %s\n%s %d\n%s %d\n%s %d\n%s %d\n", blue("name:"), yellow(pokemon.Name), green("height:"), pokemon.Height, orange("weight:"), pokemon.Weight, boldGreen("hp:"), pokemon.Hp, boldRed("attack:"), pokemon.Attack)
	fmt.Printf("%s %d\n%s %d\n%s %d\n%s %d\n%s %d\n", blue("defense:"), pokemon.Defense, boldYellow("level:"), pokemon.Level, boldRed("special attack:"), pokemon.SpecialAttack, blue("special defense:"), pokemon.SpecialDefense, green("speed:"), pokemon.Speed)
	fmt.Println(boldYellow("type:"))
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
	fmt.Println(blue("Closing the Pokedex... Goodbye!"))
	os.Exit(0)
	return nil
}

func commandHelp(cfg *Config, _ string) error {
	fmt.Printf("%s\nUsage:\n\n", green("Welcome to the Pokedex!"))
	for _, command := range supportedCommands {
		fmt.Printf("%s: %s\n", blue(command.name), command.description)
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
		fmt.Println(yellow(pokemon.Pokemon.Name))
	}
	return nil
}

func commandCatch(_ *Config, pokemonName string) error {
	pokemon, ok := catchablePokemon[pokemonName]
	if !ok {
		fmt.Printf("You have not yet found %s or the pokemon does not exist\n", yellow(pokemonName))
		return nil
	}
	fmt.Printf("Throwing a Pokeball at %s...\n", yellow(pokemonName))
	const (
		MaxBaseExp = 635.0 // highest known base experience (e.g. Blissey)
		MinChance  = 25    // minimum capture chance %
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
		fmt.Printf("%s was caught!\n", yellow(pokemonName))
	} else {
		fmt.Printf("%s escaped!\n", yellow(pokemonName))
		return nil
	}
	pokemon.Level = 1
	for _, stat := range pokemon.Stats {
		if stat.StatInfo.Name == "hp" {
			pokemon.MaxHp = stat.BaseStat
		}
	}
	resetStats(&pokemon)
	PokeDex[pokemonName] = pokemon
	return nil
}

func resetStats(pokemon *PokemonInformation) {
	for _, stat := range pokemon.Stats {
		if pokemon.Level == 0 && stat.StatInfo.Name == "hp" {
			pokemon.MaxHp = stat.BaseStat
			pokemon.Level = 1
		}
		if stat.StatInfo.Name == "speed" {
			pokemon.Speed = stat.BaseStat
			continue
		}
		if stat.StatInfo.Name == "attack" {
			pokemon.Attack = stat.BaseStat
			continue
		}
		if stat.StatInfo.Name == "defense" {
			pokemon.Defense = stat.BaseStat
			continue
		}
		if stat.StatInfo.Name == "special-attack" {
			pokemon.SpecialAttack = stat.BaseStat
			continue
		}
		if stat.StatInfo.Name == "special-defense" {
			pokemon.SpecialDefense = stat.BaseStat
			continue
		}
	}
	pokemon.Hp = pokemon.MaxHp
}

func commandSave(_ *Config, _ string) error {
	fmt.Printf("%s your progress\nDo %s shut off the program\n", boldGreen("Saving"), boldRed("not"))
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
	fmt.Println(boldGreen("Save sucessful"))
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

func newAccount() error {
	err := os.Mkdir("save_folder", 0755)
	if err != nil {
		return err
	}
	err = commandSave(&Config{}, "")
	if err != nil {
		return err
	}
	return nil
}

func main() {
	if _, err := os.Stat("./save_folder"); os.IsNotExist(err) {
		fmt.Printf("Creating save_folder to safely store your progress\n")
		err = newAccount()
		if err != nil {
			fmt.Printf("Error creating save folder: %v", err)
		}
	}
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

		"battle": {
			name:        "battle",
			description: "Command to battle a given pokemon",
			callback:    commandBattle,
		},

		"learnmove": {
			name:        "learnmove",
			description: "Command to learn a move wich can be used in battle",
			callback:    commandLearnMove,
		},
	}
	rl, err := readline.NewEx(&readline.Config{
		Prompt:            "Pokedex > ",
		HistorySearchFold: true, // case-insensitive history search
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
	})
	if err != nil {
		fmt.Println("Error initializing readline:", err)
		return
	}
	defer rl.Close()

	for {
		input, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				// Handle Ctrl+C
				if len(input) == 0 {
					break
				}
				continue
			} else if err == io.EOF {
				// Handle Ctrl+D
				break
			}
			fmt.Println("Error reading input:", err)
			continue
		}

		// Add to history
		if strings.TrimSpace(input) != "" {
			rl.SaveHistory(input) // works in v1.5.1
		}

		cleanedInput := cleanInput(input)
		if len(cleanedInput) < 1 {
			fmt.Println("Input needs to be at least 1 character long")
			continue
		}

		command := cleanedInput[0]
		secondCommand := ""
		if len(cleanedInput) > 1 {
			secondCommand = cleanedInput[1]
		}

		if cmd, exists := supportedCommands[command]; exists {
			if err := cmd.callback(cfg, secondCommand); err != nil {
				fmt.Println(err)
			}
		} else {
			fmt.Println("Unknown command")
		}
	}
}
