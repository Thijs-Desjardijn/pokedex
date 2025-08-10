# pokedex

This is a pokedex CLI that has all the basics such as catching logic, storing the Pokémon in a pokedex, reviewing the pokemons in your pokedex and a lot more.

## Installation 

These instructions are for a Linux-based system.

First of all, make sure you have go installed using: 
```bash 
go version
```
If not, go to the [Go installation page](https://go.dev/doc/install) and install go.

After this make sure you have git installed using:
```bash
git version
```
if not, go to the [git instalation page](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git) and install git.

Now you can simply initialize a git environment and use a pull request to get the pokedex:
```bash
git init
git pull https://github.com/Thijs-Desjardijn/pokedex
```

Inside the root of the project run 
```bash
mkdir save_folder
```
this is to make a save folder where your progress stored.
## Usage

To use a command simply type the command name after you see "Pokedex >".

Casing and spaces do not matter, as they are handled by the program.

You can view all the commands that are available using the "help" command. Some commands require an extra part of the command like an area or a Pokémon name.